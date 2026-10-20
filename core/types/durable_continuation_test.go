package types

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func durableTestCheckpoint() PersistedRunCheckpoint {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	p := PersistedRunCheckpoint{Ref: NativeCheckpointRef{SchemaVersion: NativeContinuationVersion, CheckpointID: "cp", SourceComponent: "runner", OwnerNamespace: "test", RunID: "run", SessionID: "session", PauseReason: PauseReasonInputRequired, SourcePosition: "iteration:1", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, Generation: 1, Material: json.RawMessage(`{"outcomes":[]}`), DependencyProfile: "test-model", DependencyRevision: "v1", Mode: "run", Iteration: 1}
	p, _ = SealRunCheckpoint(p)
	return p
}

func TestDurableCheckpointRoundTripAndDetachedMaterial(t *testing.T) {
	p := durableTestCheckpoint()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	copy := p.Detached()
	copy.Material[0] = '['
	if p.Material[0] != '{' {
		t.Fatal("material aliases caller memory")
	}
	if _, err := json.Marshal(p); err != nil {
		t.Fatal(err)
	}
}

func TestDurableCheckpointValidationClassifications(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*PersistedRunCheckpoint)
		want error
	}{
		{"schema", func(p *PersistedRunCheckpoint) { p.Ref.SchemaVersion = "future" }, ErrCheckpointSchema},
		{"identity", func(p *PersistedRunCheckpoint) { p.Ref.RunID = strings.Repeat("x", 257) }, ErrCheckpointMalformed},
		{"digest", func(p *PersistedRunCheckpoint) { p.Ref.Digest = "changed" }, ErrCheckpointCorrupt},
		{"material", func(p *PersistedRunCheckpoint) { p.Material = json.RawMessage(`{"changed":true}`) }, ErrCheckpointCorrupt},
		{"entries", func(p *PersistedRunCheckpoint) { p.CompletedEntries = 1025 }, ErrCheckpointMalformed},
		{"expiry", func(p *PersistedRunCheckpoint) { p.Ref.ExpiresAt = p.Ref.CreatedAt }, ErrCheckpointMalformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := durableTestCheckpoint()
			tc.edit(&p)
			if err := p.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestDurableCheckpointBoundsAndLimits(t *testing.T) {
	p := durableTestCheckpoint()
	p.Ref.RunID = strings.Repeat("x", 256)
	p.CompletedEntries = 1024
	p, err := SealRunCheckpoint(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Material = make(json.RawMessage, NativeContinuationMaxMaterialBytes+1)
	if _, err := SealRunCheckpoint(p); !errors.Is(err, ErrCheckpointMalformed) {
		t.Fatalf("material limit: %v", err)
	}
	for _, l := range []ContinuationLimits{{EnvelopeBytes: -1}, {MaterialBytes: NativeContinuationMaxMaterialBytes + 1}, {Entries: 1025}} {
		if _, err := l.Normalized(); !errors.Is(err, ErrCheckpointMalformed) {
			t.Fatalf("limits accepted: %#v %v", l, err)
		}
	}
}

func TestDurableCheckpointPrivateMaterialDoesNotConsumePublicEnvelope(t *testing.T) {
	p := durableTestCheckpoint()
	p.Material = append(json.RawMessage(`"`), append(json.RawMessage(strings.Repeat("x", NativeContinuationMaxEnvelopeBytes)), '"')...)
	p, err := SealRunCheckpoint(p)
	if err != nil {
		t.Fatalf("private material within 1MiB rejected by public envelope: %v", err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDurableCheckpointRejectsTTLAfterRetentionOrDeadline(t *testing.T) {
	p := durableTestCheckpoint()
	p.Ref.ExpiresAt = p.Ref.CreatedAt.Add(24*time.Hour + time.Nanosecond)
	if _, err := SealRunCheckpoint(p); !errors.Is(err, ErrCheckpointMalformed) {
		t.Fatalf("long ttl: %v", err)
	}
	p = durableTestCheckpoint()
	p.Deadline = p.Ref.CreatedAt.Add(30 * time.Minute)
	p.Ref.ExpiresAt = p.Ref.CreatedAt.Add(time.Hour)
	if _, err := SealRunCheckpoint(p); !errors.Is(err, ErrCheckpointMalformed) {
		t.Fatalf("expiry after deadline: %v", err)
	}
}

func TestDurableResumeIdentityIncludesEveryBinding(t *testing.T) {
	p := durableTestCheckpoint()
	r := ResumeRequest{Checkpoint: p.Ref, Mode: "run", DependencyProfile: p.DependencyProfile, DependencyRevision: p.DependencyRevision, Generation: 1, IdempotencyKey: "key"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	other := r
	other.AuthorityDigest = "new"
	if r.IdentityDigest() == other.IdentityDigest() {
		t.Fatal("authority omitted from identity")
	}
	other = r
	other.Budget.Revision = "new"
	if r.IdentityDigest() == other.IdentityDigest() {
		t.Fatal("budget omitted from identity")
	}
	other = r
	other.Budget.ToolConsumed = 2
	if r.IdentityDigest() == other.IdentityDigest() {
		t.Fatal("tool accounting omitted")
	}
}
