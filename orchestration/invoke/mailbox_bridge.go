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
	Mailbox *mailbox.Mailbox
}

func NewMailboxBridge(mb *mailbox.Mailbox) *MailboxBridge {
	return &MailboxBridge{Mailbox: mb}
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
	return b.Mailbox.PublishCommand(ctx, command)
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
	if _, err := b.Mailbox.PublishCommand(ctx, command); err != nil {
		return Outcome{}, err
	}

	outcome, invokeErr := InvokeSync(ctx, client, req)
	resultEnvelope := mailbox.NewResultEnvelope(
		command,
		resultMessageID(req),
		resultIdempotencyKey(req),
		buildResultPayload(outcome),
	)
	if _, err := b.Mailbox.PublishResult(ctx, resultEnvelope); err != nil {
		if invokeErr != nil {
			return outcome, fmt.Errorf("%w; publish result envelope failed: %v", invokeErr, err)
		}
		return outcome, err
	}
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
	if _, err := b.Mailbox.PublishCommand(ctx, command); err != nil {
		return a2a.AsyncSubmitAck{}, err
	}

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
		if _, err := b.Mailbox.PublishResult(cbCtx, result); err != nil {
			return err
		}
		if sink != nil {
			return sink.Deliver(cbCtx, report)
		}
		return nil
	})

	return InvokeAsync(ctx, client, req, wrapped)
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
