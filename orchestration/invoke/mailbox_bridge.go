package invoke

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

type MailboxBridge struct {
	Mailbox         *mailbox.Mailbox
	publishObserver MailboxPublishObserver
}

type MailboxPublishPath string

const (
	PublishPathCommand        MailboxPublishPath = "command"
	PublishPathResult         MailboxPublishPath = "result"
	PublishPathDelayedCommand MailboxPublishPath = "delayed_command"
)

type MailboxPublishObserver func(ctx context.Context, path MailboxPublishPath, result mailbox.PublishResult)

type MailboxBridgeOption func(*MailboxBridge)

func WithPublishObserver(observer MailboxPublishObserver) MailboxBridgeOption {
	return func(b *MailboxBridge) {
		if b == nil {
			return
		}
		b.publishObserver = observer
	}
}

func NewMailboxBridge(mb *mailbox.Mailbox, opts ...MailboxBridgeOption) *MailboxBridge {
	bridge := &MailboxBridge{Mailbox: mb}
	for _, opt := range opts {
		if opt != nil {
			opt(bridge)
		}
	}
	return bridge
}

func NewInMemoryMailboxBridge() (*MailboxBridge, error) {
	mb, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		return nil, err
	}
	return NewMailboxBridge(mb), nil
}

func (b *MailboxBridge) PublishDelayedCommand(
	ctx context.Context,
	req Request,
	notBefore time.Time,
	expireAt time.Time,
) (mailbox.PublishResult, error) {
	if b == nil || b.Mailbox == nil {
		return mailbox.PublishResult{}, errors.New("mailbox bridge requires mailbox instance")
	}
	command := mailbox.NewCommandEnvelope(
		commandMessageID(req),
		commandIdempotencyKey(req),
		toCorrelation(req),
		strings.TrimSpace(req.AgentID),
		strings.TrimSpace(req.PeerID),
		req.Payload,
		notBefore,
		expireAt,
	)
	published, err := b.Mailbox.PublishCommand(ctx, command)
	if err != nil {
		return mailbox.PublishResult{}, err
	}
	b.observePublish(ctx, PublishPathDelayedCommand, published)
	return published, nil
}

func (b *MailboxBridge) InvokeSync(ctx context.Context, client Client, req Request) (Outcome, error) {
	if b == nil || b.Mailbox == nil {
		return Outcome{}, errors.New("mailbox bridge requires mailbox instance")
	}
	command := mailbox.NewCommandEnvelope(
		commandMessageID(req),
		commandIdempotencyKey(req),
		toCorrelation(req),
		strings.TrimSpace(req.AgentID),
		strings.TrimSpace(req.PeerID),
		req.Payload,
		time.Time{},
		time.Time{},
	)
	commandPublished, err := b.Mailbox.PublishCommand(ctx, command)
	if err != nil {
		return Outcome{}, err
	}
	b.observePublish(ctx, PublishPathCommand, commandPublished)

	outcome, invokeErr := invokeSync(ctx, client, req)
	resultEnvelope := mailbox.NewResultEnvelope(
		command,
		resultMessageID(req),
		resultIdempotencyKey(req),
		buildResultPayload(outcome),
	)
	resultPublished, err := b.Mailbox.PublishResult(ctx, resultEnvelope)
	if err != nil {
		if invokeErr != nil {
			return outcome, fmt.Errorf("%w; publish result envelope failed: %v", invokeErr, err)
		}
		return outcome, err
	}
	b.observePublish(ctx, PublishPathResult, resultPublished)
	return outcome, invokeErr
}

func (b *MailboxBridge) InvokeAsync(
	ctx context.Context,
	client AsyncClient,
	req AsyncRequest,
	sink a2a.ReportSink,
) (a2a.AsyncSubmitAck, error) {
	if b == nil || b.Mailbox == nil {
		return a2a.AsyncSubmitAck{}, errors.New("mailbox bridge requires mailbox instance")
	}
	command := mailbox.NewCommandEnvelope(
		commandMessageIDFromAsync(req),
		commandIdempotencyKeyFromAsync(req),
		mailbox.Correlation{
			RunID:      strings.TrimSpace(req.WorkflowID),
			TaskID:     strings.TrimSpace(req.TaskID),
			WorkflowID: strings.TrimSpace(req.WorkflowID),
			TeamID:     strings.TrimSpace(req.TeamID),
		},
		strings.TrimSpace(req.AgentID),
		strings.TrimSpace(req.PeerID),
		req.Payload,
		time.Time{},
		time.Time{},
	)
	commandPublished, err := b.Mailbox.PublishCommand(ctx, command)
	if err != nil {
		return a2a.AsyncSubmitAck{}, err
	}
	b.observePublish(ctx, PublishPathCommand, commandPublished)

	wrapped := a2a.ReportSinkFunc(func(cbCtx context.Context, report a2a.AsyncReport) error {
		result, err := mailbox.NewAsyncResultEnvelope(mailbox.AsyncReport{
			ReportKey:  strings.TrimSpace(report.ReportKey),
			OutcomeKey: strings.TrimSpace(report.OutcomeKey),
			Correlation: mailbox.Correlation{
				RunID:      strings.TrimSpace(report.WorkflowID),
				TaskID:     strings.TrimSpace(report.TaskID),
				WorkflowID: strings.TrimSpace(report.WorkflowID),
				TeamID:     strings.TrimSpace(report.TeamID),
			},
			FromAgent: strings.TrimSpace(report.AgentID),
			ToAgent:   strings.TrimSpace(report.PeerID),
			Payload: map[string]any{
				"status":            strings.TrimSpace(string(report.Status)),
				"result":            cloneMap(report.Result),
				"error_class":       strings.TrimSpace(string(report.ErrorClass)),
				"error_layer":       strings.TrimSpace(report.ErrorLayer),
				"error_code":        strings.TrimSpace(report.ErrorCode),
				"error_message":     strings.TrimSpace(report.ErrorMessage),
				"delivery_attempt":  report.DeliveryAttempt,
				"business_terminal": report.BusinessTerminal,
			},
		})
		if err != nil {
			return err
		}
		published, err := b.Mailbox.PublishResult(cbCtx, result)
		if err != nil {
			return err
		}
		b.observePublish(cbCtx, PublishPathResult, published)
		if sink != nil {
			return sink.Deliver(cbCtx, report)
		}
		return nil
	})

	return invokeAsync(ctx, client, req, wrapped)
}

func buildResultPayload(outcome Outcome) map[string]any {
	payload := map[string]any{
		"task_id": strings.TrimSpace(outcome.TaskID),
		"status":  strings.TrimSpace(string(outcome.TerminalStatus)),
		"result":  cloneMap(outcome.Result),
	}
	if outcome.Error != nil {
		payload["error"] = map[string]any{
			"message":   strings.TrimSpace(outcome.Error.Message),
			"class":     strings.TrimSpace(string(outcome.Error.Class)),
			"layer":     strings.TrimSpace(outcome.Error.Layer),
			"code":      strings.TrimSpace(outcome.Error.Code),
			"retryable": outcome.Error.Retryable,
		}
	}
	return payload
}

func toCorrelation(req Request) mailbox.Correlation {
	return mailbox.Correlation{
		RunID:      strings.TrimSpace(req.WorkflowID),
		TaskID:     strings.TrimSpace(req.TaskID),
		WorkflowID: strings.TrimSpace(req.WorkflowID),
		TeamID:     strings.TrimSpace(req.TeamID),
	}
}

func commandMessageID(req Request) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return taskID + ":command"
}

func resultMessageID(req Request) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return taskID + ":result"
}

func commandIdempotencyKey(req Request) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(req.WorkflowID) + "|" + strings.TrimSpace(req.TeamID)
	}
	return strings.TrimSpace(taskID + "|sync|command")
}

func resultIdempotencyKey(req Request) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(req.WorkflowID) + "|" + strings.TrimSpace(req.TeamID)
	}
	return strings.TrimSpace(taskID + "|sync|result")
}

func commandMessageIDFromAsync(req AsyncRequest) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return taskID + ":command"
}

func commandIdempotencyKeyFromAsync(req AsyncRequest) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(req.WorkflowID) + "|" + strings.TrimSpace(req.TeamID)
	}
	return strings.TrimSpace(taskID + "|async|command")
}

func (b *MailboxBridge) observePublish(ctx context.Context, path MailboxPublishPath, result mailbox.PublishResult) {
	if b == nil || b.publishObserver == nil {
		return
	}
	b.publishObserver(ctx, path, result)
}
