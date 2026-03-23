package mailbox

import (
	"context"
	"errors"
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
	claimed, ok, err := w.mailbox.Consume(ctx, w.consumerID)
	if err != nil || !ok {
		return false, err
	}
	if err := w.handler(ctx, claimed); err != nil {
		reason := CanonicalizeLifecycleReason(err.Error(), LifecycleReasonHandlerError)
		switch w.config.HandlerErrorPolicy {
		case WorkerHandlerErrorPolicyNack:
			_, handleErr := w.mailbox.Nack(ctx, claimed.Envelope.MessageID, w.consumerID, reason)
			return true, handleErr
		default:
			_, handleErr := w.mailbox.Requeue(ctx, claimed.Envelope.MessageID, w.consumerID, reason)
			return true, handleErr
		}
	}
	_, err = w.mailbox.Ack(ctx, claimed.Envelope.MessageID, w.consumerID)
	if err != nil {
		return true, err
	}
	return true, nil
}
