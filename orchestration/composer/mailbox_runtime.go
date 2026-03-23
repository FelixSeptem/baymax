package composer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/invoke"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	mailboxFallbackFileInitFailed = "mailbox.backend.file_init_failed"
	mailboxConfiguredDisabled     = "disabled"
	mailboxReasonDelayedPublish   = "mailbox.publish.delayed"
	mailboxReasonDuplicatePublish = "mailbox.publish.duplicate"
	mailboxLifecyclePathPrefix    = "lifecycle."
)

type schedulerManagedMailbox struct {
	mailbox *mailbox.Mailbox
	bridge  *invoke.MailboxBridge
}

func (c *Composer) initMailbox(cfg runtimeconfig.Config) error {
	enabled := cfg.Mailbox.Enabled
	configuredBackend := strings.TrimSpace(strings.ToLower(cfg.Mailbox.Backend))
	if configuredBackend == "" {
		configuredBackend = runtimeconfig.MailboxBackendMemory
	}
	effectiveBackend := configuredBackend
	path := strings.TrimSpace(cfg.Mailbox.Path)
	if !enabled {
		effectiveBackend = runtimeconfig.MailboxBackendMemory
		configuredBackend = mailboxConfiguredDisabled
	}

	storeInit, err := mailbox.NewStoreWithFallback(effectiveBackend, path, mailbox.Policy{
		MaxAttempts:    cfg.Mailbox.Retry.MaxAttempts,
		BackoffInitial: cfg.Mailbox.Retry.BackoffInitial,
		BackoffMax:     cfg.Mailbox.Retry.BackoffMax,
		JitterRatio:    cfg.Mailbox.Retry.JitterRatio,
		TTL:            cfg.Mailbox.TTL,
		DLQEnabled:     cfg.Mailbox.DLQ.Enabled,
	})
	if err != nil {
		return err
	}
	if !enabled {
		storeInit.Fallback = false
		storeInit.FallbackReason = ""
		storeInit.Requested = mailboxConfiguredDisabled
	}
	opts := make([]mailbox.Option, 0, 1)
	if cfg.Mailbox.Worker.Enabled {
		opts = append(opts, mailbox.WithLifecycleObserver(c.recordMailboxLifecycle))
	}
	mailboxRuntime, err := mailbox.New(storeInit.Store, opts...)
	if err != nil {
		return err
	}
	runtime := &schedulerManagedMailbox{
		mailbox: mailboxRuntime,
		bridge: invoke.NewMailboxBridge(
			mailboxRuntime,
			invoke.WithPublishObserver(c.recordMailboxPublish),
		),
	}

	c.schedulerMu.Lock()
	c.mailbox = runtime
	c.mailboxEnabled = enabled
	c.mailboxPath = path
	c.mailboxConfiguredBackend = configuredBackend
	c.mailboxBackend = strings.TrimSpace(strings.ToLower(storeInit.Backend))
	c.mailboxFallback = storeInit.Fallback
	c.mailboxFallbackReason = strings.TrimSpace(storeInit.FallbackReason)
	if c.mailboxFallback && c.mailboxFallbackReason == "" {
		c.mailboxFallbackReason = mailboxFallbackFileInitFailed
	}
	c.mailboxSignature = c.mailboxConfigSignature(cfg)
	c.schedulerMu.Unlock()
	return nil
}

func (c *Composer) refreshMailboxForNextAttempt() {
	if c == nil || !c.managedMailbox || c.runtimeMgr == nil {
		return
	}
	cfg := c.runtimeMgr.EffectiveConfig()
	signature := c.mailboxConfigSignature(cfg)

	c.schedulerMu.RLock()
	if c.mailboxSignature == signature {
		c.schedulerMu.RUnlock()
		return
	}
	c.schedulerMu.RUnlock()
	_ = c.initMailbox(cfg)
}

func (c *Composer) mailboxConfigSignature(cfg runtimeconfig.Config) string {
	return fmt.Sprintf(
		"%t|%s|%s|%d|%d|%.4f|%d|%t|%d|%d|%t|%d|%s|%d|%d|%t|%s",
		cfg.Mailbox.Enabled,
		strings.TrimSpace(strings.ToLower(cfg.Mailbox.Backend)),
		strings.TrimSpace(cfg.Mailbox.Path),
		cfg.Mailbox.Retry.MaxAttempts,
		cfg.Mailbox.Retry.BackoffInitial.Milliseconds(),
		cfg.Mailbox.Retry.JitterRatio,
		cfg.Mailbox.TTL.Milliseconds(),
		cfg.Mailbox.DLQ.Enabled,
		cfg.Mailbox.Query.PageSizeDefault,
		cfg.Mailbox.Query.PageSizeMax,
		cfg.Mailbox.Worker.Enabled,
		cfg.Mailbox.Worker.PollInterval.Milliseconds(),
		strings.TrimSpace(strings.ToLower(cfg.Mailbox.Worker.HandlerErrorPolicy)),
		cfg.Mailbox.Worker.InflightTimeout.Milliseconds(),
		cfg.Mailbox.Worker.HeartbeatInterval.Milliseconds(),
		cfg.Mailbox.Worker.ReclaimOnConsume,
		strings.TrimSpace(strings.ToLower(cfg.Mailbox.Worker.PanicPolicy)),
	)
}

func (c *Composer) mailboxBridgeProvider() (*invoke.MailboxBridge, error) {
	if c == nil {
		return nil, errors.New("composer is nil")
	}
	c.schedulerMu.RLock()
	runtime := c.mailbox
	c.schedulerMu.RUnlock()
	if runtime == nil || runtime.bridge == nil {
		return nil, errors.New("managed mailbox bridge is not initialized")
	}
	return runtime.bridge, nil
}

func (c *Composer) recordMailboxPublish(
	_ context.Context,
	path invoke.MailboxPublishPath,
	published mailbox.PublishResult,
) {
	if c == nil || c.runtimeMgr == nil {
		return
	}
	record := published.Record
	envelope := record.Envelope
	configuredBackend, backend, fallback, fallbackReason := c.mailboxDiagnosticsMeta()
	attempt := envelope.Attempt
	if attempt <= 0 {
		attempt = max(record.DeliveryAttempt, 1)
	}
	reasonCode := ""
	switch {
	case fallback && fallbackReason != "":
		reasonCode = fallbackReason
	case path == invoke.PublishPathDelayedCommand:
		reasonCode = mailboxReasonDelayedPublish
	case published.Duplicate:
		reasonCode = mailboxReasonDuplicatePublish
	}
	c.runtimeMgr.RecordMailboxDiagnostic(runtimeconfig.MailboxDiagnosticRecord{
		Time:                  nonZeroMailboxRecordTime(c.now, record.UpdatedAt),
		MessageID:             strings.TrimSpace(envelope.MessageID),
		IdempotencyKey:        strings.TrimSpace(envelope.IdempotencyKey),
		CorrelationID:         strings.TrimSpace(envelope.CorrelationID),
		Kind:                  string(envelope.Kind),
		State:                 string(record.State),
		FromAgent:             strings.TrimSpace(envelope.FromAgent),
		ToAgent:               strings.TrimSpace(envelope.ToAgent),
		RunID:                 strings.TrimSpace(envelope.RunID),
		TaskID:                strings.TrimSpace(envelope.TaskID),
		WorkflowID:            strings.TrimSpace(envelope.WorkflowID),
		TeamID:                strings.TrimSpace(envelope.TeamID),
		Attempt:               attempt,
		ConsumerID:            strings.TrimSpace(record.ConsumerID),
		ReasonCode:            reasonCode,
		Backend:               strings.TrimSpace(backend),
		ConfiguredBackend:     strings.TrimSpace(configuredBackend),
		BackendFallback:       fallback,
		BackendFallbackReason: strings.TrimSpace(fallbackReason),
		PublishPath:           string(path),
	})
}

func (c *Composer) recordMailboxLifecycle(_ context.Context, event mailbox.LifecycleEvent) {
	if c == nil || c.runtimeMgr == nil {
		return
	}
	record := event.Record
	envelope := record.Envelope
	configuredBackend, backend, fallback, fallbackReason := c.mailboxDiagnosticsMeta()
	attempt := envelope.Attempt
	if attempt <= 0 {
		attempt = max(record.DeliveryAttempt, 1)
	}
	reasonCode := strings.TrimSpace(event.ReasonCode)
	if reasonCode != "" {
		reasonCode = mailbox.CanonicalizeLifecycleReason(reasonCode, mailbox.LifecycleReasonHandlerError)
	}
	if reasonCode == "" {
		switch event.Transition {
		case mailbox.TransitionDeadLetter:
			reasonCode = mailbox.LifecycleReasonRetryExhausted
		case mailbox.TransitionExpired:
			reasonCode = mailbox.LifecycleReasonExpired
		}
	}
	c.runtimeMgr.RecordMailboxDiagnostic(runtimeconfig.MailboxDiagnosticRecord{
		Time:                  nonZeroMailboxRecordTime(c.now, event.Time),
		MessageID:             strings.TrimSpace(envelope.MessageID),
		IdempotencyKey:        strings.TrimSpace(envelope.IdempotencyKey),
		CorrelationID:         strings.TrimSpace(envelope.CorrelationID),
		Kind:                  string(envelope.Kind),
		State:                 string(record.State),
		FromAgent:             strings.TrimSpace(envelope.FromAgent),
		ToAgent:               strings.TrimSpace(envelope.ToAgent),
		RunID:                 strings.TrimSpace(envelope.RunID),
		TaskID:                strings.TrimSpace(envelope.TaskID),
		WorkflowID:            strings.TrimSpace(envelope.WorkflowID),
		TeamID:                strings.TrimSpace(envelope.TeamID),
		Attempt:               attempt,
		ConsumerID:            strings.TrimSpace(record.ConsumerID),
		ReasonCode:            reasonCode,
		Backend:               strings.TrimSpace(backend),
		ConfiguredBackend:     strings.TrimSpace(configuredBackend),
		BackendFallback:       fallback,
		BackendFallbackReason: strings.TrimSpace(fallbackReason),
		PublishPath:           mailboxLifecyclePathPrefix + strings.TrimSpace(string(event.Transition)),
		Reclaimed:             event.Reclaimed,
		PanicRecovered:        event.PanicRecovered,
	})
}

func (c *Composer) mailboxDiagnosticsMeta() (configured, backend string, fallback bool, fallbackReason string) {
	c.schedulerMu.RLock()
	defer c.schedulerMu.RUnlock()
	return c.mailboxConfiguredBackend, c.mailboxBackend, c.mailboxFallback, c.mailboxFallbackReason
}

func nonZeroMailboxRecordTime(now func() time.Time, ts time.Time) time.Time {
	if !ts.IsZero() {
		return ts.UTC()
	}
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
