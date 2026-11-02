// Package sourceadapter provides an explicit, bounded external-source delivery
// lifecycle. It does not own scheduling, authorization or business effect CAS.
package sourceadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxReferenceBytes = 256
	MaxKindBytes      = 64
	MaxPayloadBytes   = 64 << 10
	MaxCursorBytes    = 4096
	MaxPageSize       = 200
	MaxRecentReceipts = 200
	DefaultTimeout    = 5 * time.Second
)

var (
	ErrInvalid    = errors.New("source_adapter.invalid")
	ErrForeign    = errors.New("source_adapter.foreign")
	ErrConflict   = errors.New("source_adapter.conflict")
	ErrRetryable  = errors.New("source_adapter.retryable")
	ErrDependency = errors.New("source_adapter.dependency_failed")
	ErrClosed     = errors.New("source_adapter.closed")
	ErrBusy       = errors.New("source_adapter.busy")
)

type SourceIdentity struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
}

type Metadata struct {
	Source            SourceIdentity `json:"source"`
	IdentityNamespace string         `json:"identity_namespace"`
	CursorVersion     string         `json:"cursor_version"`
}

type Cursor struct {
	Metadata Metadata `json:"metadata"`
	Token    string   `json:"token"`
}

type Fact struct {
	ExternalID    string
	Reference     string
	PayloadDigest string
	Payload       []byte
	Next          Cursor
}

type Page struct {
	Metadata Metadata
	After    *Cursor
	Facts    []Fact
}

type Disposition string

const (
	Sent      Disposition = "sent"
	Gone      Disposition = "gone"
	Retryable Disposition = "retryable"
)

type Source interface {
	Poll(context.Context, *Cursor, int) (Page, error)
}
type Consumer interface {
	Consume(context.Context, Fact) (Disposition, error)
}

// Store errors may represent an uncertain commit. Reload and reconcile before
// retrying. A store must return detached snapshots and preserve metadata binding.
type Store interface {
	Load(context.Context) (State, error)
	CompareAndSwap(context.Context, uint64, State) (State, error)
}

type Options struct {
	OperationTimeout time.Duration
	PageLimit        int
}

func validRef(s string, max int) bool {
	if s == "" || len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (m Metadata) Validate() error {
	if !validRef(m.Source.Kind, MaxKindBytes) || !validRef(m.Source.Namespace, MaxReferenceBytes) || !validRef(m.Source.ID, MaxReferenceBytes) || !validRef(m.IdentityNamespace, MaxReferenceBytes) || !validRef(m.CursorVersion, MaxReferenceBytes) {
		return ErrInvalid
	}
	return nil
}

func (c Cursor) validate(m Metadata) error {
	if c.Metadata != m {
		return ErrForeign
	}
	if c.Token == "" || len(c.Token) > MaxCursorBytes || !utf8.ValidString(c.Token) {
		return ErrInvalid
	}
	return nil
}

func validDigest(s string) bool {
	if len(s) != 71 || s[:7] != "sha256:" {
		return false
	}
	b, err := hex.DecodeString(s[7:])
	return err == nil && hex.EncodeToString(b) == s[7:]
}

func digest(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

func receipt(m Metadata, f Fact) (Receipt, error) {
	if !validRef(f.ExternalID, MaxReferenceBytes) || !validRef(f.Reference, MaxReferenceBytes) || len(f.Payload) > MaxPayloadBytes || f.PayloadDigest != digest(f.Payload) {
		return Receipt{}, ErrInvalid
	}
	return ReferenceReceipt(m, f.ExternalID, f.Reference, f.PayloadDigest, f.Next)
}

// ReferenceReceipt constructs bounded, canonical reference evidence. It cannot
// verify a payload or authorize an effect; live Fact admission also hashes body.
func ReferenceReceipt(m Metadata, externalID, reference, payloadDigest string, next Cursor) (Receipt, error) {
	if err := m.Validate(); err != nil {
		return Receipt{}, err
	}
	if err := next.validate(m); err != nil {
		return Receipt{}, err
	}
	if !validRef(externalID, MaxReferenceBytes) || !validRef(reference, MaxReferenceBytes) || !validDigest(payloadDigest) {
		return Receipt{}, ErrInvalid
	}
	r := Receipt{ExternalID: externalID, Reference: reference, PayloadDigest: payloadDigest, Next: next}
	b, err := json.Marshal(r)
	if err != nil {
		return Receipt{}, ErrInvalid
	}
	r.Fingerprint = digest(b)
	return r, nil
}

func cloneCursor(c *Cursor) *Cursor {
	if c == nil {
		return nil
	}
	v := *c
	return &v
}

func sameCursor(a, b *Cursor) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func missing(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
