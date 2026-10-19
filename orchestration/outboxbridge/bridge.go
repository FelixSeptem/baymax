// Package outboxbridge binds a host-owned application outbox to one native owner.
package outboxbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

const Profile = "application_outbox_dispatch.v1"

type Target string

const (
	TargetMailbox   Target = "mailbox"
	TargetScheduler Target = "scheduler"
)

type Descriptor struct {
	Profile        string            `json:"profile"`
	Target         Target            `json:"target"`
	OutboxID       string            `json:"outbox_id"`
	OccurrenceID   string            `json:"occurrence_id,omitempty"`
	Revision       string            `json:"revision"`
	AuthorityRef   string            `json:"authority_ref"`
	EffectRef      string            `json:"effect_ref"`
	EffectKey      string            `json:"effect_key"`
	RunID          string            `json:"run_id"`
	MessageID      string            `json:"message_id,omitempty"`
	SourceTaskID   string            `json:"source_task_id,omitempty"`
	AttemptID      string            `json:"attempt_id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	Mailbox        *MailboxBinding   `json:"mailbox,omitempty"`
	Scheduler      *SchedulerBinding `json:"scheduler,omitempty"`
}
type MailboxBinding struct {
	ToAgent   string    `json:"to_agent"`
	FromAgent string    `json:"from_agent,omitempty"`
	NotBefore time.Time `json:"not_before,omitempty"`
	ExpireAt  time.Time `json:"expire_at,omitempty"`
}
type SchedulerBinding struct {
	AgentID     string    `json:"agent_id"`
	MaxAttempts int       `json:"max_attempts,omitempty"`
	NotBefore   time.Time `json:"not_before,omitempty"`
}

type Evidence struct {
	EffectCommitted bool   `json:"effect_committed"`
	OutboxConfirmed bool   `json:"outbox_confirmed"`
	Digest          string `json:"digest"`
}
type Decision struct {
	Eligible bool
	Digest   string
	Canceled bool
	Evidence Evidence
}
type Admission func(context.Context, Descriptor, string) (Decision, error)

type Config struct {
	Mailbox   *mailbox.Mailbox
	Scheduler *scheduler.Scheduler
	Admission Admission
}
type Bridge struct {
	mailbox   *mailbox.Mailbox
	scheduler *scheduler.Scheduler
	admission Admission
}

type Facts struct {
	DispatchAccepted bool     `json:"dispatch_accepted"`
	Target           Target   `json:"target"`
	MessageID        string   `json:"message_id,omitempty"`
	TaskID           string   `json:"task_id,omitempty"`
	TargetState      string   `json:"target_state,omitempty"`
	DeliveryAttempt  int      `json:"delivery_attempt,omitempty"`
	AttemptID        string   `json:"attempt_id,omitempty"`
	Effect           Evidence `json:"effect"`
	OutboxConfirmed  bool     `json:"outbox_confirmed"`
}
type Result struct {
	Facts     Facts
	Digest    string
	Duplicate bool
}

// Inspect reads the native owner's current record. It never invokes admission,
// changes state, acknowledges delivery, completes a task or confirms an outbox.
func (b *Bridge) Inspect(ctx context.Context, d Descriptor) (Result, error) {
	n, digest, err := Canonical(d)
	if err != nil {
		return Result{}, err
	}
	result := Result{Digest: digest}
	result.Facts.Target = n.Target
	switch n.Target {
	case TargetMailbox:
		if b.mailbox == nil {
			return Result{}, &Error{Code: ErrUnavailableTarget}
		}
		snap, e := b.mailbox.Snapshot(ctx)
		if e != nil {
			return Result{}, &Error{Code: ErrTargetFailed, err: e}
		}
		id := nativeID(n.IdempotencyKey)
		for _, record := range snap.Records {
			if record.Envelope.MessageID == id {
				result.Facts.DispatchAccepted = true
				result.Facts.MessageID = id
				result.Facts.TargetState = string(record.State)
				result.Facts.DeliveryAttempt = record.DeliveryAttempt
				result.Facts.Effect = Evidence{Digest: digest}
				return result, nil
			}
		}
		return Result{}, &Error{Code: ErrNotFound}
	case TargetScheduler:
		if b.scheduler == nil {
			return Result{}, &Error{Code: ErrUnavailableTarget}
		}
		record, ok, e := b.scheduler.Get(ctx, nativeID(n.IdempotencyKey))
		if e != nil {
			return Result{}, &Error{Code: ErrTargetFailed, err: e}
		}
		if !ok {
			return Result{}, &Error{Code: ErrNotFound}
		}
		result.Facts.DispatchAccepted = true
		result.Facts.TaskID = record.Task.TaskID
		result.Facts.TargetState = string(record.State)
		result.Facts.Effect = Evidence{Digest: digest}
		if at, ok := record.CurrentAttempt, true; ok {
			result.Facts.AttemptID = at
		}
		return result, nil
	default:
		return Result{}, &Error{Code: ErrUnsupportedTarget}
	}
}

// Reconcile observes current host admission/effect evidence and native target
// state after a crash. It never performs a business effect or native terminal
// action; the host decides whether to ack/complete/confirm in its own owner.
func (b *Bridge) Reconcile(ctx context.Context, d Descriptor) (Result, error) {
	n, digest, err := Canonical(d)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, &Error{Code: ErrCanceled, err: err}
	}
	var decision Decision
	func() {
		defer func() {
			if recover() != nil {
				err = &Error{Code: ErrAdmissionFailed}
			}
		}()
		decision, err = b.admission(ctx, n, digest)
	}()
	if err != nil {
		return Result{}, &Error{Code: ErrAdmissionFailed, err: err}
	}
	if decision.Canceled {
		return Result{}, &Error{Code: ErrCanceled}
	}
	if !decision.Eligible || decision.Digest != digest {
		return Result{}, &Error{Code: ErrAdmissionDenied}
	}
	result, err := b.Inspect(ctx, n)
	if err != nil {
		return Result{}, err
	}
	result.Facts.Effect = decision.Evidence
	result.Facts.OutboxConfirmed = decision.Evidence.OutboxConfirmed
	return result, nil
}

type Error struct {
	Code string
	err  error
}

func (e *Error) Error() string { return e.Code }
func (e *Error) Unwrap() error { return e.err }

const (
	ErrInvalidDescriptor  = "outbox.invalid_descriptor"
	ErrUnsupportedProfile = "outbox.unsupported_profile"
	ErrUnsupportedTarget  = "outbox.unsupported_target"
	ErrUnavailableTarget  = "outbox.unavailable_target"
	ErrAdmissionDenied    = "outbox.admission_denied"
	ErrAdmissionFailed    = "outbox.admission_failed"
	ErrBindingMismatch    = "outbox.binding_mismatch"
	ErrCanceled           = "outbox.canceled"
	ErrTargetFailed       = "outbox.target_failed"
	ErrNotFound           = "outbox.not_found"
)

func classifyTargetError(err error) *Error {
	if errors.Is(err, mailbox.ErrDispatchBindingMismatch) || errors.Is(err, scheduler.ErrDispatchBindingMismatch) {
		return &Error{Code: ErrBindingMismatch, err: err}
	}
	return &Error{Code: ErrTargetFailed, err: err}
}

func New(c Config) (*Bridge, error) {
	if c.Admission == nil {
		return nil, &Error{Code: ErrAdmissionFailed, err: errors.New("admission required")}
	}
	if c.Mailbox == nil && c.Scheduler == nil {
		return nil, &Error{Code: ErrUnavailableTarget}
	}
	return &Bridge{mailbox: c.Mailbox, scheduler: c.Scheduler, admission: c.Admission}, nil
}

func (d Descriptor) normalized() (Descriptor, error) {
	o := d
	o.Profile = strings.TrimSpace(o.Profile)
	if o.Profile == "" {
		o.Profile = Profile
	}
	o.Target = Target(strings.ToLower(strings.TrimSpace(string(o.Target))))
	for p := range map[*string]bool{&o.OutboxID: true, &o.OccurrenceID: true, &o.Revision: true, &o.AuthorityRef: true, &o.EffectRef: true, &o.EffectKey: true, &o.RunID: true, &o.MessageID: true, &o.SourceTaskID: true, &o.AttemptID: true, &o.IdempotencyKey: true} {
		*p = strings.TrimSpace(*p)
		if len(*p) > 256 {
			return Descriptor{}, fmt.Errorf("reference too long")
		}
		for i := 0; i < len(*p); i++ {
			c := (*p)[i]
			if !isReferenceChar(c) {
				return Descriptor{}, fmt.Errorf("reference contains unsupported character")
			}
		}
	}
	if o.Profile != Profile {
		return Descriptor{}, fmt.Errorf("unsupported profile")
	}
	if o.OutboxID == "" || o.Revision == "" || o.AuthorityRef == "" || o.EffectRef == "" || o.EffectKey == "" || o.RunID == "" || o.IdempotencyKey == "" {
		return Descriptor{}, fmt.Errorf("required reference missing")
	}
	switch o.Target {
	case TargetMailbox:
		if o.Mailbox == nil || strings.TrimSpace(o.Mailbox.ToAgent) == "" || o.Scheduler != nil {
			return Descriptor{}, fmt.Errorf("invalid mailbox binding")
		}
		b := *o.Mailbox
		b.ToAgent = strings.TrimSpace(b.ToAgent)
		b.FromAgent = strings.TrimSpace(b.FromAgent)
		b.NotBefore = b.NotBefore.UTC()
		b.ExpireAt = b.ExpireAt.UTC()
		if !b.ExpireAt.IsZero() && !b.NotBefore.IsZero() && b.ExpireAt.Before(b.NotBefore) {
			return Descriptor{}, fmt.Errorf("expiry before delay")
		}
		o.Mailbox = &b
	case TargetScheduler:
		if o.Scheduler == nil || strings.TrimSpace(o.Scheduler.AgentID) == "" || o.Mailbox != nil {
			return Descriptor{}, fmt.Errorf("invalid scheduler binding")
		}
		b := *o.Scheduler
		b.AgentID = strings.TrimSpace(b.AgentID)
		if b.MaxAttempts <= 0 {
			b.MaxAttempts = 3
		}
		if b.MaxAttempts > 100 {
			return Descriptor{}, fmt.Errorf("max attempts exceeded")
		}
		b.NotBefore = b.NotBefore.UTC()
		o.Scheduler = &b
	default:
		return Descriptor{}, fmt.Errorf("unsupported target")
	}
	return o, nil
}

func isReferenceChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._:/@+-", rune(c))
}
func Canonical(d Descriptor) (Descriptor, string, error) {
	if profile := strings.TrimSpace(d.Profile); profile != "" && profile != Profile {
		return Descriptor{}, "", &Error{Code: ErrUnsupportedProfile}
	}
	if target := strings.ToLower(strings.TrimSpace(string(d.Target))); target != "" && target != string(TargetMailbox) && target != string(TargetScheduler) {
		return Descriptor{}, "", &Error{Code: ErrUnsupportedTarget}
	}
	n, e := d.normalized()
	if e != nil {
		return Descriptor{}, "", &Error{Code: ErrInvalidDescriptor, err: e}
	}
	raw, _ := json.Marshal(n)
	if len(raw) > 16*1024 {
		return Descriptor{}, "", &Error{Code: ErrInvalidDescriptor, err: errors.New("descriptor too large")}
	}
	sum := sha256.Sum256(raw)
	return n, "sha256:" + hex.EncodeToString(sum[:]), nil
}
func nativeID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "outbox-" + hex.EncodeToString(sum[:])
}

func (b *Bridge) Dispatch(ctx context.Context, d Descriptor) (res Result, err error) {
	n, digest, e := Canonical(d)
	if e != nil {
		return Result{}, e
	}
	if err := ctx.Err(); err != nil {
		return Result{}, &Error{Code: ErrCanceled, err: err}
	}
	decision := Decision{}
	func() {
		defer func() {
			if recover() != nil {
				err = &Error{Code: ErrAdmissionFailed}
			}
		}()
		decision, err = b.admission(ctx, n, digest)
	}()
	if err != nil {
		return Result{}, &Error{Code: ErrAdmissionFailed, err: err}
	}
	if decision.Canceled {
		return Result{}, &Error{Code: ErrCanceled}
	}
	if !decision.Eligible || decision.Digest != digest {
		return Result{}, &Error{Code: ErrAdmissionDenied}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, &Error{Code: ErrCanceled, err: err}
	}
	res.Digest = digest
	res.Facts.Target = n.Target
	res.Facts.Effect = decision.Evidence
	res.Facts.OutboxConfirmed = decision.Evidence.OutboxConfirmed
	switch n.Target {
	case TargetMailbox:
		if b.mailbox == nil {
			return Result{}, &Error{Code: ErrUnavailableTarget}
		}
		r, e := b.mailbox.Publish(ctx, mailbox.Envelope{MessageID: nativeID(n.IdempotencyKey), IdempotencyKey: nativeID(n.IdempotencyKey), Kind: mailbox.KindCommand, FromAgent: n.Mailbox.FromAgent, ToAgent: n.Mailbox.ToAgent, RunID: n.RunID, TaskID: n.SourceTaskID, NotBefore: n.Mailbox.NotBefore, ExpireAt: n.Mailbox.ExpireAt, DispatchBinding: digest, Payload: map[string]any{"outbox_binding": digest, "effect_ref": n.EffectRef}})
		if e != nil {
			return Result{}, classifyTargetError(e)
		}
		res.Facts.DispatchAccepted = true
		res.Facts.MessageID = r.Record.Envelope.MessageID
		res.Facts.TargetState = string(r.Record.State)
		res.Facts.DeliveryAttempt = r.Record.DeliveryAttempt
		res.Duplicate = r.Duplicate
	case TargetScheduler:
		if b.scheduler == nil {
			return Result{}, &Error{Code: ErrUnavailableTarget}
		}
		r, e := b.scheduler.Enqueue(ctx, scheduler.Task{TaskID: nativeID(n.IdempotencyKey), RunID: n.RunID, AgentID: n.Scheduler.AgentID, MaxAttempts: n.Scheduler.MaxAttempts, NotBefore: n.Scheduler.NotBefore, DispatchBinding: digest, Payload: map[string]any{"outbox_binding": digest, "effect_ref": n.EffectRef}})
		if e != nil {
			return Result{}, classifyTargetError(e)
		}
		res.Facts.DispatchAccepted = true
		res.Facts.TaskID = r.Task.TaskID
		res.Facts.TargetState = string(r.State)
		// Scheduler.Enqueue converges existing task identity but intentionally
		// does not expose a duplicate bit; the native record is authoritative.
		res.Duplicate = false
	}
	return res, nil
}
