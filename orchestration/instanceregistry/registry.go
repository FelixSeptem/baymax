// Package instanceregistry records live instances and session→instance
// affinity in a shared store so a routing layer can forward sticky sessions
// across pods. This package owns storage and contracts only: it contains no
// HTTP, load-balancing or routing logic. With no registry configured, hosts
// keep running single-instance and affinity lookups report unavailability.
package instanceregistry

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	// ErrAffinityLost means the session is bound to an instance that is no
	// longer alive; the host decides whether to replay or reject.
	ErrAffinityLost = errors.New("instance registry: session affinity lost")
	// ErrAffinityUnavailable means affinity cannot be answered right now
	// (backend unreachable or registry not configured).
	ErrAffinityUnavailable = errors.New("instance registry: affinity unavailable")
	// ErrSessionUnbound means the session has no recorded affinity binding.
	ErrSessionUnbound = errors.New("instance registry: session is unbound")
	// ErrUnknownInstance means the instance ID is not registered.
	ErrUnknownInstance = errors.New("instance registry: unknown instance")
)

// InstanceInfo is one live instance's self-reported identity. Endpoint is
// opaque to this package; the host routing layer interprets it.
type InstanceInfo struct {
	InstanceID string    `json:"instance_id"`
	Endpoint   string    `json:"endpoint"`
	StartedAt  time.Time `json:"started_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// Registry tracks live instances (heartbeat TTL) and session affinity.
// Implementations must treat the backend TTL as authoritative for liveness.
type Registry interface {
	// Register inserts or heartbeats one instance. Re-registering the same
	// instance refreshes its TTL and updates the endpoint.
	Register(ctx context.Context, instanceID, endpoint string, ttl time.Duration) error
	// List returns currently alive instances.
	List(ctx context.Context) ([]InstanceInfo, error)
	// Unregister removes an instance and drops its session bindings.
	Unregister(ctx context.Context, instanceID string) error
	// BindSession records session→instance affinity with a TTL.
	BindSession(ctx context.Context, sessionID, instanceID string, ttl time.Duration) error
	// ResolveSession returns the owning instance ID. An expired or missing
	// binding reports ErrSessionUnbound; a binding to a dead instance reports
	// ErrAffinityLost.
	ResolveSession(ctx context.Context, sessionID string) (string, error)
}

// normalizeInstanceID trims and validates a bounded instance/session ID.
func normalizeInstanceID(in string) (string, error) {
	out := strings.TrimSpace(in)
	if out == "" || len(out) > 256 || strings.ContainsAny(out, " \t\r\n") {
		return "", errors.New("instance registry: instance id must be a bounded single token")
	}
	return out, nil
}
