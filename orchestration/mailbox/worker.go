package mailbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkerHandler func(ctx context.Context, record Record) error

type Worker struct {
	mailbox    *Mailbox
	config     WorkerConfig
	consumerID string
	handler    WorkerHandler
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

	panicRecovered, handlerErr := invokeWorkerHandler(ctx, w.handler, claimed)
	stopHeartbeat()
	<-heartbeatDone
	select {
	case hbErr := <-heartbeatErrCh:
		if handlerErr == nil {
			handlerErr = hbErr
		}
	default:
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

func invokeWorkerHandler(ctx context.Context, handler WorkerHandler, claimed Record) (panicRecovered bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicRecovered = true
			err = fmt.Errorf("worker handler panic recovered: %v", r)
		}
	}()
	return false, handler(ctx, claimed)
}
