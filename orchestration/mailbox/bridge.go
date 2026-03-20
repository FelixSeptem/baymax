package mailbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Correlation struct {
	RunID      string
	TaskID     string
	WorkflowID string
	TeamID     string
}

type AsyncReport struct {
	ReportKey   string
	OutcomeKey  string
	Correlation Correlation
	FromAgent   string
	ToAgent     string
	Payload     map[string]any
}

func NewCommandEnvelope(
	messageID string,
	idempotencyKey string,
	correlation Correlation,
	fromAgent string,
	toAgent string,
	payload map[string]any,
	notBefore time.Time,
	expireAt time.Time,
) Envelope {
	return Envelope{
		MessageID:      strings.TrimSpace(messageID),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		CorrelationID:  strings.TrimSpace(messageID),
		Kind:           KindCommand,
		FromAgent:      strings.TrimSpace(fromAgent),
		ToAgent:        strings.TrimSpace(toAgent),
		TaskID:         strings.TrimSpace(correlation.TaskID),
		RunID:          strings.TrimSpace(correlation.RunID),
		WorkflowID:     strings.TrimSpace(correlation.WorkflowID),
		TeamID:         strings.TrimSpace(correlation.TeamID),
		Payload:        copyMap(payload),
		NotBefore:      notBefore,
		ExpireAt:       expireAt,
	}
}

func NewResultEnvelope(command Envelope, messageID, idempotencyKey string, payload map[string]any) Envelope {
	return Envelope{
		MessageID:      strings.TrimSpace(messageID),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		CorrelationID:  strings.TrimSpace(command.MessageID),
		Kind:           KindResult,
		FromAgent:      strings.TrimSpace(command.ToAgent),
		ToAgent:        strings.TrimSpace(command.FromAgent),
		TaskID:         strings.TrimSpace(command.TaskID),
		RunID:          strings.TrimSpace(command.RunID),
		WorkflowID:     strings.TrimSpace(command.WorkflowID),
		TeamID:         strings.TrimSpace(command.TeamID),
		Payload:        copyMap(payload),
	}
}

func NewAsyncResultEnvelope(report AsyncReport) (Envelope, error) {
	reportKey := strings.TrimSpace(report.ReportKey)
	if reportKey == "" {
		return Envelope{}, errors.New("report_key is required")
	}
	outcomeKey := strings.TrimSpace(report.OutcomeKey)
	if outcomeKey == "" {
		outcomeKey = "result"
	}
	messageID := fmt.Sprintf("%s:%s", reportKey, outcomeKey)
	return Envelope{
		MessageID:      messageID,
		IdempotencyKey: reportKey,
		CorrelationID:  strings.TrimSpace(report.Correlation.TaskID),
		Kind:           KindResult,
		FromAgent:      strings.TrimSpace(report.FromAgent),
		ToAgent:        strings.TrimSpace(report.ToAgent),
		TaskID:         strings.TrimSpace(report.Correlation.TaskID),
		RunID:          strings.TrimSpace(report.Correlation.RunID),
		WorkflowID:     strings.TrimSpace(report.Correlation.WorkflowID),
		TeamID:         strings.TrimSpace(report.Correlation.TeamID),
		Payload:        copyMap(report.Payload),
	}, nil
}

func PublishAsyncResult(ctx context.Context, m *Mailbox, report AsyncReport) (PublishResult, error) {
	if m == nil {
		return PublishResult{}, errors.New("mailbox is required")
	}
	envelope, err := NewAsyncResultEnvelope(report)
	if err != nil {
		return PublishResult{}, err
	}
	return m.PublishResult(ctx, envelope)
}
