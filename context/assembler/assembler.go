package assembler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/context/guard"
	"github.com/FelixSeptem/baymax/context/journal"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusBypass  = "bypass"
)

type Assembler struct {
	cfgProvider func() runtimeconfig.ContextAssemblerConfig
	now         func() time.Time

	mu          sync.Mutex
	storageKey  string
	storage     journal.Storage
	prefixCache map[string]string
}

func New(cfgProvider func() runtimeconfig.ContextAssemblerConfig) *Assembler {
	return &Assembler{
		cfgProvider: cfgProvider,
		now:         time.Now,
		prefixCache: map[string]string{},
	}
}

func (a *Assembler) Assemble(ctx context.Context, req types.ContextAssembleRequest, modelReq types.ModelRequest) (types.ModelRequest, types.ContextAssembleResult, error) {
	start := a.now()
	cfg := a.cfgProvider()
	if !cfg.Enabled {
		return modelReq, types.ContextAssembleResult{
			Prefix: types.PrefixMetadata{
				SessionID:     req.SessionID,
				PrefixVersion: req.PrefixVersion,
			},
			LatencyMs: 0,
			Status:    StatusBypass,
		}, nil
	}

	storage, err := a.ensureStorage(cfg)
	if err != nil {
		return modelReq, failedResult(req, start, "storage.backend.not_ready"), err
	}
	req.PrefixVersion = strings.TrimSpace(req.PrefixVersion)
	if req.PrefixVersion == "" {
		req.PrefixVersion = strings.TrimSpace(cfg.PrefixVersion)
	}
	g := guard.New(cfg.Guard.FailFast)
	prefixHash, err := buildPrefixHash(req)
	if err != nil {
		return modelReq, failedResult(req, start, "prefix.build.failed"), err
	}

	sessionKey := stableSessionKey(req.SessionID, req.RunID, req.PrefixVersion)
	expected := a.cachedHash(sessionKey)
	guardResult, guardErr := g.Apply(req, prefixHash, expected)

	intent := journal.Entry{
		Time:          start,
		RunID:         req.RunID,
		SessionID:     req.SessionID,
		Phase:         "intent",
		PrefixVersion: req.PrefixVersion,
		PrefixHash:    prefixHash,
	}
	if err := storage.Append(ctx, intent); err != nil {
		return modelReq, failedResult(req, start, "journal.intent.write_failed"), err
	}
	if guardErr != nil {
		commit := journal.Entry{
			Time:          a.now(),
			RunID:         req.RunID,
			SessionID:     req.SessionID,
			Phase:         "commit",
			PrefixVersion: req.PrefixVersion,
			PrefixHash:    prefixHash,
			Status:        StatusFailed,
			Violation:     guardResult.GuardViolation,
		}
		_ = storage.Append(ctx, commit)
		return modelReq, failedResult(req, start, guardResult.GuardViolation), guardErr
	}

	a.rememberHash(sessionKey, prefixHash)
	commit := journal.Entry{
		Time:          a.now(),
		RunID:         req.RunID,
		SessionID:     req.SessionID,
		Phase:         "commit",
		PrefixVersion: req.PrefixVersion,
		PrefixHash:    prefixHash,
		Status:        StatusSuccess,
	}
	if err := storage.Append(ctx, commit); err != nil {
		return modelReq, failedResult(req, start, "journal.commit.write_failed"), err
	}
	modelReq.Input = guardResult.Input
	modelReq.Messages = guardResult.Messages
	return modelReq, types.ContextAssembleResult{
		Prefix: types.PrefixMetadata{
			SessionID:     req.SessionID,
			PrefixVersion: req.PrefixVersion,
			PrefixHash:    prefixHash,
		},
		LatencyMs:    a.now().Sub(start).Milliseconds(),
		Status:       StatusSuccess,
		GuardFailure: "",
	}, nil
}

func (a *Assembler) ensureStorage(cfg runtimeconfig.ContextAssemblerConfig) (journal.Storage, error) {
	key := strings.ToLower(strings.TrimSpace(cfg.Storage.Backend)) + "|" + strings.TrimSpace(cfg.JournalPath)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.storage != nil && a.storageKey == key {
		return a.storage, nil
	}
	s, err := journal.NewStorage(cfg.Storage.Backend, cfg.JournalPath)
	if err != nil {
		return nil, err
	}
	a.storage = s
	a.storageKey = key
	return a.storage, nil
}

func (a *Assembler) cachedHash(key string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.prefixCache[key]
}

func (a *Assembler) rememberHash(key, hash string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prefixCache[key] = hash
}

func stableSessionKey(sessionID, runID, prefixVersion string) string {
	base := strings.TrimSpace(sessionID)
	if base == "" {
		base = strings.TrimSpace(runID)
	}
	return base + "|" + strings.TrimSpace(prefixVersion)
}

func buildPrefixHash(req types.ContextAssembleRequest) (string, error) {
	systemMessages := make([]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		if strings.EqualFold(strings.TrimSpace(m.Role), "system") {
			systemMessages = append(systemMessages, strings.TrimSpace(m.Content))
		}
	}
	payload := map[string]any{
		"prefix_version":  req.PrefixVersion,
		"system_messages": systemMessages,
		"capabilities":    req.Capabilities.Normalized(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal prefix blocks: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func failedResult(req types.ContextAssembleRequest, start time.Time, violation string) types.ContextAssembleResult {
	return types.ContextAssembleResult{
		Prefix: types.PrefixMetadata{
			SessionID:     req.SessionID,
			PrefixVersion: req.PrefixVersion,
		},
		LatencyMs:    time.Since(start).Milliseconds(),
		Status:       StatusFailed,
		GuardFailure: violation,
	}
}
