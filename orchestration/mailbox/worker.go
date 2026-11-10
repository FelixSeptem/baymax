package mailbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkerHandler func(ctx context.Context, record Record) error

// HandlerResult is the explicit result protocol for a configured continuation
// worker. A nil Pause is successful handler completion; ordinary handler
// failures remain the returned error and use the existing lifecycle policy.
// Pauses are never encoded in arbitrary errors.
type HandlerResult struct {
	Pause *HandlerPause
}

// HandlerPause identifies one pre-effect, source-owned worker safe point. The
// checkpoint material is opaque to Mailbox and bounded before persistence.
type HandlerPause struct {
	Reason                 MailboxPauseReason
	PendingActionReference string
	CompletedEffectRefs    []string
	ConsumedBudget         int64
	OriginalDeadline       time.Time
	ExpiresAt              time.Time
	CheckpointReference    string
}

type ContinuationWorkerHandler func(ctx context.Context, record Record) (HandlerResult, error)
type ContinuationWorkerResumer func(ctx context.Context, continuation MailboxContinuation, record Record) (HandlerResult, error)
type HandlerResumeRevalidator func(ctx context.Context, continuation MailboxContinuation) error

type Worker struct {
	mailbox             *Mailbox
	config              WorkerConfig
	consumerID          string
	handler             WorkerHandler
	continuationHandler ContinuationWorkerHandler
	continuationResumer ContinuationWorkerResumer
	revalidateResume    HandlerResumeRevalidator
}

// NewContinuationWorker creates an opt-in handler-continuation worker. The
// existing NewWorker constructor remains lifecycle-only and cannot advertise a
// host handler execution safe point.
func NewContinuationWorker(mb *Mailbox, cfg WorkerConfig, handler ContinuationWorkerHandler, resumer ContinuationWorkerResumer, revalidate HandlerResumeRevalidator, consumerID string) (*Worker, error) {
	if handler == nil {
		return nil, errors.New("continuation worker handler is required")
	}
	if resumer == nil {
		return nil, errors.New("continuation worker resumer is required")
	}
	if revalidate == nil {
		return nil, errors.New("continuation worker revalidation callback is required")
	}
	caps, err := mb.ContinuationCapabilities(context.Background())
	if err != nil {
		return nil, err
	}
	if !caps.CrossProcessRestore {
		return nil, ErrMailboxContinuationUnsupported
	}
	worker, err := NewWorker(mb, cfg, func(context.Context, Record) error { return nil }, consumerID)
	if err != nil {
		return nil, err
	}
	worker.continuationHandler = handler
	worker.continuationResumer = resumer
	worker.revalidateResume = revalidate
	mb.enableHandlerContinuation()
	return worker, nil
}

func NewWorker(mb *Mailbox, cfg WorkerConfig, handler WorkerHandler, consumerID string) (*Worker, error) {
	if mb == nil {
		return nil, errors.New("mailbox is required")
	}
	if handler == nil {
		return nil, errors.New("worker handler is required")
	}
	normalized, err := NormalizeWorkerConfig(cfg)
	if err != nil {
		return nil, err
	}
	resolvedConsumer := strings.TrimSpace(consumerID)
	if resolvedConsumer == "" {
		resolvedConsumer = DefaultWorkerConsumerID
	}
	return &Worker{
		mailbox:    mb,
		config:     normalized,
		consumerID: resolvedConsumer,
		handler:    handler,
	}, nil
}

func (w *Worker) Config() WorkerConfig {
	if w == nil {
		return WorkerConfig{}
	}
	return w.config
}

func (w *Worker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("worker is nil")
	}
	if !w.config.Enabled {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		processed, err := w.RunOnce(ctx)
		if err != nil {
			return err
		}
		if processed {
			continue
		}
		timer := time.NewTimer(w.config.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("worker is nil")
	}
	claimed, ok, err := w.mailbox.ConsumeWithLease(
		ctx,
		w.consumerID,
		w.config.InflightTimeout,
		w.config.ReclaimOnConsume,
	)
	if err != nil || !ok {
		return false, err
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	heartbeatErrCh := make(chan error, 1)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(w.config.HeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if _, hbErr := w.mailbox.Heartbeat(
					heartbeatCtx,
					claimed.Envelope.MessageID,
					w.consumerID,
					w.config.InflightTimeout,
				); hbErr != nil {
					select {
					case heartbeatErrCh <- hbErr:
					default:
					}
					return
				}
			}
		}
	}()

	var panicRecovered bool
	var handlerErr error
	var handlerResult HandlerResult
	if w.continuationHandler != nil {
		panicRecovered, handlerResult, handlerErr = invokeContinuationWorkerHandler(ctx, w.continuationHandler, claimed)
	} else {
		panicRecovered, handlerErr = invokeWorkerHandler(ctx, w.handler, claimed)
	}
	stopHeartbeat()
	<-heartbeatDone
	select {
	case hbErr := <-heartbeatErrCh:
		if handlerErr == nil {
			handlerErr = hbErr
		}
	default:
	}
	if handlerErr == nil && handlerResult.Pause != nil {
		if err := w.pauseHandler(ctx, claimed, *handlerResult.Pause); err != nil {
			return true, err
		}
		return true, nil
	}

	if handlerErr != nil {
		reason := CanonicalizeLifecycleReason(handlerErr.Error(), LifecycleReasonHandlerError)
		opts := ActionOptions{}
		if panicRecovered {
			reason = LifecycleReasonHandlerError
			opts.PanicRecovered = true
		}
		switch w.config.HandlerErrorPolicy {
		case WorkerHandlerErrorPolicyNack:
			_, handleErr := w.mailbox.NackWithOptions(ctx, claimed.Envelope.MessageID, w.consumerID, reason, opts)
			return true, handleErr
		default:
			_, handleErr := w.mailbox.RequeueWithOptions(ctx, claimed.Envelope.MessageID, w.consumerID, reason, opts)
			return true, handleErr
		}
	}
	_, err = w.mailbox.Ack(ctx, claimed.Envelope.MessageID, w.consumerID)
	if err != nil {
		return true, err
	}
	return true, nil
}

func invokeContinuationWorkerHandler(ctx context.Context, handler ContinuationWorkerHandler, claimed Record) (panicRecovered bool, result HandlerResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicRecovered = true
			err = fmt.Errorf("worker handler panic recovered: %v", r)
		}
	}()
	result, err = handler(ctx, claimed)
	return false, result, err
}

func (w *Worker) pauseHandler(ctx context.Context, claimed Record, pause HandlerPause) error {
	if w == nil || w.mailbox == nil || claimed.Envelope.HostPlanBinding == nil {
		return ErrMailboxContinuationUnsupported
	}
	if !validMailboxPauseReason(pause.Reason) || strings.TrimSpace(pause.CheckpointReference) == "" || len(pause.CheckpointReference) > 1024 || pause.ExpiresAt.IsZero() {
		return ErrMailboxContinuationMalformed
	}
	now := w.mailbox.nowTime()
	if !pause.ExpiresAt.After(now) || pause.ExpiresAt.Sub(now) > 24*time.Hour || (!pause.OriginalDeadline.IsZero() && !pause.OriginalDeadline.After(now)) {
		return ErrMailboxContinuationExpired
	}
	binding := *claimed.Envelope.HostPlanBinding
	caps, err := w.mailbox.ContinuationCapabilities(ctx)
	if err != nil {
		return err
	}
	continuation, err := SealMailboxContinuation(MailboxContinuation{
		Ref:                        MailboxContinuationRef{OwnerNamespace: "mailbox", Deployment: caps.Deployment},
		Binding:                    binding,
		MessageID:                  claimed.Envelope.MessageID,
		DeliveryID:                 claimed.DeliveryID,
		LeaseID:                    claimed.LeaseID,
		ConsumerID:                 claimed.ConsumerID,
		AttemptCausation:           claimed.AttemptCausation,
		DeliveryAttempt:            claimed.DeliveryAttempt,
		LifecycleState:             claimed.State,
		ParentRunID:                binding.ParentRunID,
		RunID:                      binding.RunID,
		HostCorrelation:            binding.HostCorrelation,
		AuthorityDigest:            binding.AuthorityDigest,
		PlanDigest:                 binding.PlanDigest,
		WriteScope:                 binding.WriteScope,
		EffectNamespace:            binding.EffectNamespace,
		Occurrence:                 binding.Occurrence,
		PauseReason:                pause.Reason,
		PendingActionReference:     strings.TrimSpace(pause.PendingActionReference),
		CompletedEffectRefs:        append([]string(nil), pause.CompletedEffectRefs...),
		HandlerCheckpointReference: strings.TrimSpace(pause.CheckpointReference),
		ConsumedBudget:             pause.ConsumedBudget,
		OriginalDeadline:           pause.OriginalDeadline.UTC(),
		Generation:                 1,
		CreatedAt:                  now,
		ExpiresAt:                  pause.ExpiresAt.UTC(),
	})
	if err != nil {
		return err
	}
	return w.mailbox.SaveHostPlanContinuation(ctx, continuation)
}

// ResumeHandlerContinuation restores a persisted, handler-safe boundary using
// this explicitly configured worker. It never re-consumes/re-enqueues the
// message: the same live delivery is claimed through the mailbox owner.
func (w *Worker) ResumeHandlerContinuation(ctx context.Context, req MailboxResumeRequest) (HostPlanOutcome, error) {
	if w == nil || w.mailbox == nil || w.continuationResumer == nil {
		return HostPlanOutcome{}, ErrMailboxContinuationUnavailable
	}
	continuation, err := w.mailbox.LoadHostPlanContinuation(ctx, req.Ref)
	if err != nil {
		// A terminal duplicate must remain projectable after checkpoint expiry;
		// the atomic owner claim can return its retained result without starting
		// work. No other load failure is recoverable here.
		claim, claimErr := w.mailbox.ClaimHostPlanResume(ctx, req)
		if claimErr == nil && claim.Duplicate && claim.Result != nil {
			return *claim.Result, nil
		}
		return HostPlanOutcome{}, err
	}
	if strings.TrimSpace(continuation.HandlerCheckpointReference) == "" {
		return HostPlanOutcome{}, ErrMailboxContinuationUnsupported
	}
	if err := w.revalidateResume(ctx, continuation); err != nil {
		return HostPlanOutcome{}, err
	}
	claim, err := w.mailbox.ClaimHostPlanResume(ctx, req)
	if err != nil {
		if claim.Result != nil {
			return *claim.Result, nil
		}
		return HostPlanOutcome{}, err
	}
	if claim.Duplicate && claim.Result != nil {
		return *claim.Result, nil
	}
	if _, err := w.mailbox.MarkHostPlanResumeRunning(ctx, claim); err != nil {
		return HostPlanOutcome{}, err
	}
	page, err := w.mailbox.Query(ctx, QueryRequest{MessageID: continuation.MessageID})
	if err != nil || len(page.Items) != 1 {
		if err != nil {
			return HostPlanOutcome{}, err
		}
		return HostPlanOutcome{}, ErrMailboxContinuationUnavailable
	}
	record := page.Items[0]
	if record.State != StateInFlight || record.DeliveryID != continuation.DeliveryID || record.LeaseID != continuation.LeaseID || record.AttemptCausation != continuation.AttemptCausation || record.ConsumerID != continuation.ConsumerID || record.ConsumerID != w.consumerID {
		return HostPlanOutcome{}, ErrMailboxContinuationLeaseLost
	}
	// Establish the resumed handler's lease before entering user code. A later
	// heartbeat failure still follows the normal source lifecycle and is never
	// turned into a synthetic retry or second handler invocation.
	if _, err := w.mailbox.Heartbeat(ctx, continuation.MessageID, w.consumerID, w.config.InflightTimeout); err != nil {
		return HostPlanOutcome{}, ErrMailboxContinuationLeaseLost
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	heartbeatErrCh := make(chan error, 1)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(w.config.HeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if _, heartbeatErr := w.mailbox.Heartbeat(heartbeatCtx, continuation.MessageID, w.consumerID, w.config.InflightTimeout); heartbeatErr != nil {
					select {
					case heartbeatErrCh <- heartbeatErr:
					default:
					}
					return
				}
			}
		}
	}()
	_, result, resumeErr := invokeContinuationWorkerResumer(ctx, w.continuationResumer, continuation, record)
	stopHeartbeat()
	<-heartbeatDone
	select {
	case heartbeatErr := <-heartbeatErrCh:
		if resumeErr == nil {
			resumeErr = heartbeatErr
		}
	default:
	}
	if result.Pause != nil && resumeErr == nil {
		pause := *result.Pause
		now := w.mailbox.nowTime()
		if !validMailboxPauseReason(pause.Reason) || strings.TrimSpace(pause.CheckpointReference) == "" || len(pause.CheckpointReference) > 1024 || pause.ExpiresAt.IsZero() {
			return HostPlanOutcome{}, ErrMailboxContinuationMalformed
		}
		if !pause.ExpiresAt.After(now) || pause.ExpiresAt.Sub(now) > 24*time.Hour || (!pause.OriginalDeadline.IsZero() && !pause.OriginalDeadline.After(now)) {
			return HostPlanOutcome{}, ErrMailboxContinuationExpired
		}
		successor, sealErr := SealMailboxContinuation(MailboxContinuation{
			Ref:                        MailboxContinuationRef{OwnerNamespace: "mailbox", Deployment: continuation.Ref.Deployment},
			Binding:                    continuation.Binding,
			MessageID:                  continuation.MessageID,
			DeliveryID:                 continuation.DeliveryID,
			LeaseID:                    continuation.LeaseID,
			ConsumerID:                 continuation.ConsumerID,
			AttemptCausation:           continuation.AttemptCausation,
			DeliveryAttempt:            continuation.DeliveryAttempt,
			LifecycleState:             continuation.LifecycleState,
			ParentRunID:                continuation.ParentRunID,
			RunID:                      continuation.RunID,
			HostCorrelation:            continuation.HostCorrelation,
			AuthorityDigest:            continuation.AuthorityDigest,
			PlanDigest:                 continuation.PlanDigest,
			WriteScope:                 continuation.WriteScope,
			EffectNamespace:            continuation.EffectNamespace,
			Occurrence:                 continuation.Occurrence,
			PauseReason:                pause.Reason,
			PendingActionReference:     strings.TrimSpace(pause.PendingActionReference),
			CompletedEffectRefs:        append([]string(nil), pause.CompletedEffectRefs...),
			HandlerCheckpointReference: strings.TrimSpace(pause.CheckpointReference),
			ConsumedBudget:             pause.ConsumedBudget,
			OriginalDeadline:           pause.OriginalDeadline.UTC(),
			PredecessorCheckpointID:    continuation.Ref.CheckpointID,
			Generation:                 continuation.Generation + 1,
			CreatedAt:                  now,
			ExpiresAt:                  pause.ExpiresAt.UTC(),
		})
		if sealErr != nil {
			return HostPlanOutcome{}, sealErr
		}
		outcome, outcomeErr := HostPlanOutcomeFromRecord(record)
		if outcomeErr != nil {
			return HostPlanOutcome{}, outcomeErr
		}
		outcome.LifecycleResult = "paused"
		outcome.LifecycleReason = string(pause.Reason)
		outcome.TerminalOutcome = ""
		if _, commitErr := w.mailbox.CommitHostPlanResumePause(ctx, claim, successor, outcome); commitErr != nil {
			return HostPlanOutcome{}, commitErr
		}
		outcome.SuccessorContinuation = cloneContinuationRef(successor.Ref)
		return outcome, nil
	}
	if resumeErr != nil {
		reason := CanonicalizeLifecycleReason(resumeErr.Error(), LifecycleReasonHandlerError)
		if w.config.HandlerErrorPolicy == WorkerHandlerErrorPolicyNack {
			_, err = w.mailbox.Nack(ctx, continuation.MessageID, w.consumerID, reason)
		} else {
			_, err = w.mailbox.Requeue(ctx, continuation.MessageID, w.consumerID, reason)
		}
	} else {
		_, err = w.mailbox.Ack(ctx, continuation.MessageID, w.consumerID)
	}
	if err != nil {
		return HostPlanOutcome{}, err
	}
	page, err = w.mailbox.Query(ctx, QueryRequest{MessageID: continuation.MessageID})
	if err != nil || len(page.Items) != 1 {
		if err != nil {
			return HostPlanOutcome{}, err
		}
		return HostPlanOutcome{}, ErrMailboxContinuationUnavailable
	}
	outcome, err := HostPlanOutcomeFromRecord(page.Items[0])
	if err != nil {
		return HostPlanOutcome{}, err
	}
	if _, err := w.mailbox.CommitHostPlanResumeResult(ctx, claim, outcome); err != nil {
		return HostPlanOutcome{}, err
	}
	return outcome, nil
}

func invokeContinuationWorkerResumer(ctx context.Context, resumer ContinuationWorkerResumer, continuation MailboxContinuation, record Record) (panicRecovered bool, result HandlerResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicRecovered = true
			err = fmt.Errorf("worker continuation resumer panic recovered: %v", r)
		}
	}()
	result, err = resumer(ctx, continuation.Detached(), cloneRecord(record))
	return false, result, err
}

func invokeWorkerHandler(ctx context.Context, handler WorkerHandler, claimed Record) (panicRecovered bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicRecovered = true
			err = fmt.Errorf("worker handler panic recovered: %v", r)
		}
	}()
	return false, handler(ctx, claimed)
}
