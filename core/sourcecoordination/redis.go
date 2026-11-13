package sourcecoordination

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/redis/go-redis/v9"
)

// RedisEvidenceReference is the bounded evidence string advertised by the
// distributed coordinator. It names the contract the fence/CAS pairs were
// verified against.
const RedisEvidenceReference = "redis-coordination.v1"

const defaultRedisKeyPrefix = "baymax"

// RedisOptions configures the distributed coordinator.
type RedisOptions struct {
	Options
	// KeyPrefix scopes every key. Empty falls back to "baymax".
	KeyPrefix string
}

// Redis implements Coordinator on top of a Redis primitives client. The lease
// key TTL is authoritative for expiry; the persisted ExpiresAt mirrors it for
// callers that need a timestamp.
type Redis struct {
	client redis.UniversalClient
	opts   RedisOptions
	lease  string
	state  string
}

// NewRedis builds a distributed coordinator. The client is injected so tests can
// use an in-process Redis substitute without changing the wire semantics.
func NewRedis(client redis.UniversalClient, opts RedisOptions) (*Redis, error) {
	if client == nil {
		return nil, types.ErrOwnerUnavailable
	}
	if strings.TrimSpace(opts.OwnerNamespace) == "" {
		return nil, types.ErrOwnerUnavailable
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	prefix := strings.TrimSpace(opts.KeyPrefix)
	if prefix == "" {
		prefix = defaultRedisKeyPrefix
	}
	base := prefix + ":coord:" + strings.TrimSpace(opts.OwnerNamespace)
	r := &Redis{
		client: client,
		opts:   opts,
		lease:  base + ":lease",
		state:  base + ":state",
	}
	return r, nil
}

func (r *Redis) Acquire(ctx context.Context, writerID string, ttl time.Duration) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	if r == nil || strings.TrimSpace(writerID) == "" || ttl <= 0 {
		return Lease{}, types.ErrOwnerUnavailable
	}
	fence, err := randomFence()
	if err != nil {
		return Lease{}, types.ErrOwnerUnavailable
	}
	lease := Lease{
		Version:        ContractVersion,
		OwnerNamespace: r.opts.OwnerNamespace,
		WriterID:       writerID,
		Fence:          fence,
		ExpiresAt:      r.opts.Now().UTC().Add(ttl),
	}
	payload, err := json.Marshal(lease)
	if err != nil {
		return Lease{}, types.ErrOwnerUnavailable
	}
	res, err := r.client.Eval(ctx, acquireLeaseScript, []string{r.lease}, string(payload), writerID, strconv.FormatInt(ttl.Milliseconds(), 10)).Int()
	if err != nil {
		return Lease{}, types.ErrOwnerUnavailable
	}
	if res == 0 {
		return Lease{}, types.ErrOwnerUnavailable
	}
	return lease, nil
}

func (r *Redis) Read(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if r == nil || r.client == nil {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	values, err := r.client.HMGet(ctx, r.state, "revision", "payload").Result()
	if err != nil {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	if len(values) == 0 {
		return Snapshot{}, nil
	}
	out := Snapshot{}
	if revision, ok := values[0].(string); ok && revision != "" {
		parsed, err := strconv.ParseUint(revision, 10, 64)
		if err != nil {
			return Snapshot{}, types.ErrOwnerUnavailable
		}
		out.Revision = parsed
	}
	if payload, ok := values[1].(string); ok {
		out.Payload = []byte(payload)
	}
	return out, nil
}

func (r *Redis) CompareAndSwap(ctx context.Context, lease Lease, revision uint64, payload []byte) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if r == nil || r.client == nil || len(payload) > maxPayloadBytes {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	if lease.OwnerNamespace != r.opts.OwnerNamespace {
		return Snapshot{}, types.ErrForeignWriter
	}
	res, err := r.client.Eval(ctx, compareAndSwapScript, []string{r.lease, r.state},
		r.opts.OwnerNamespace,
		lease.WriterID,
		lease.Fence,
		strconv.FormatUint(revision, 10),
		string(payload),
		strconv.Itoa(maxPayloadBytes),
	).Int64()
	if err != nil {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	switch res {
	case casResultOK:
		return Snapshot{Revision: revision + 1, Payload: append([]byte(nil), payload...)}, nil
	case casResultForeign:
		return Snapshot{}, types.ErrForeignWriter
	case casResultStale:
		return Snapshot{}, types.ErrStaleWriter
	case casResultConflict:
		return Snapshot{}, types.ErrDeploymentConflict
	default:
		return Snapshot{}, types.ErrOwnerUnavailable
	}
}

func (r *Redis) ValidateLease(ctx context.Context, lease Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil || r.client == nil {
		return types.ErrOwnerUnavailable
	}
	if lease.OwnerNamespace != r.opts.OwnerNamespace {
		return types.ErrForeignWriter
	}
	res, err := r.client.Eval(ctx, validateLeaseScript, []string{r.lease},
		r.opts.OwnerNamespace, lease.WriterID, lease.Fence,
	).Int64()
	if err != nil {
		return types.ErrOwnerUnavailable
	}
	switch res {
	case casResultOK:
		return nil
	case casResultForeign:
		return types.ErrForeignWriter
	case casResultStale:
		return types.ErrStaleWriter
	default:
		return types.ErrOwnerUnavailable
	}
}

// DeploymentScope only advertises cross-machine support when the backend is
// reachable and the lease key is readable. Any failure degrades to the explicit
// unsupported scope so absence can never imply a wider guarantee.
func (r *Redis) DeploymentScope() types.DeploymentScope {
	if r == nil || r.client == nil {
		return types.DefaultDeploymentScope("source-coordinator", "source coordinator is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.client.Ping(ctx).Err(); err != nil {
		return types.DefaultDeploymentScope(r.opts.OwnerNamespace, "redis coordinator is unreachable")
	}
	if err := r.client.Get(ctx, r.lease).Err(); err != nil && err != redis.Nil {
		return types.DefaultDeploymentScope(r.opts.OwnerNamespace, "redis coordinator lease key is unreadable")
	}
	return types.CoordinatedCrossMachineScope(r.opts.OwnerNamespace, RedisEvidenceReference)
}

// Close releases the injected client. Callers own the client, so closing it is
// opt-in through this method.
func (r *Redis) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

const (
	casResultOK       int64 = 1
	casResultForeign  int64 = -2
	casResultStale    int64 = -3
	casResultConflict int64 = -5
	casResultMissing  int64 = -4
)

// acquireLeaseScript returns 1 when the caller acquired the lease, 2 when it
// refreshed its own lease, and 0 when another writer holds it.
const acquireLeaseScript = `
local key = KEYS[1]
local payload = ARGV[1]
local writer = ARGV[2]
local ttlMs = tonumber(ARGV[3])
local ok = redis.call('SET', key, payload, 'NX', 'PX', ttlMs)
if ok then
  return 1
end
local current = redis.call('GET', key)
if not current then
  local retried = redis.call('SET', key, payload, 'NX', 'PX', ttlMs)
  if retried then return 1 end
  return 0
end
local okDecode, decoded = pcall(cjson.decode, current)
if not okDecode or type(decoded) ~= 'table' then
  return 0
end
if decoded['writer_id'] == writer then
  redis.call('SET', key, payload, 'PX', ttlMs)
  return 2
end
return 0
`

// compareAndSwapScript validates fence and revision inside a single atomic
// script so a writer can never observe a partial write.
const compareAndSwapScript = `
local leaseKey = KEYS[1]
local stateKey = KEYS[2]
local owner = ARGV[1]
local writer = ARGV[2]
local fence = ARGV[3]
local expected = tonumber(ARGV[4])
local payload = ARGV[5]
local maxBytes = tonumber(ARGV[6])

local current = redis.call('GET', leaseKey)
if not current then
  return -4
end
local okDecode, decoded = pcall(cjson.decode, current)
if not okDecode or type(decoded) ~= 'table' then
  return -4
end
if decoded['owner_namespace'] ~= owner then
  return -2
end
if decoded['writer_id'] ~= writer or decoded['fence'] ~= fence then
  return -3
end

local revisionRaw = redis.call('HGET', stateKey, 'revision')
local revision = 0
if revisionRaw then
  revision = tonumber(revisionRaw)
end
if revision ~= expected then
  return -5
end
if string.len(payload) > maxBytes then
  return -4
end
redis.call('HSET', stateKey, 'revision', tostring(revision + 1), 'payload', payload)
return 1
`

const validateLeaseScript = `
local leaseKey = KEYS[1]
local owner = ARGV[1]
local writer = ARGV[2]
local fence = ARGV[3]

local current = redis.call('GET', leaseKey)
if not current then
  return -4
end
local okDecode, decoded = pcall(cjson.decode, current)
if not okDecode or type(decoded) ~= 'table' then
  return -4
end
if decoded['owner_namespace'] ~= owner then
  return -2
end
if decoded['writer_id'] ~= writer or decoded['fence'] ~= fence then
  return -3
end
return 1
`

// Compile-time interface checks.
var (
	_ Coordinator    = (*Redis)(nil)
	_ LeaseValidator = (*Redis)(nil)
)

func (r *Redis) String() string {
	if r == nil {
		return "sourcecoordination.Redis(<nil>)"
	}
	return fmt.Sprintf("sourcecoordination.Redis(%s)", r.opts.OwnerNamespace)
}
