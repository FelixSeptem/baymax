package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/FelixSeptem/baymax/core/checkpoint"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestHostResumeProvenanceLive(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, field := range []string{"admission", "policy", "scope"} {
			t.Run(field+"/"+strconv.FormatBool(stream), func(t *testing.T) {
				var bound types.HostResumeResult
				var calls int
				v := types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
					calls++
					return bound, nil
				})
				e, cp, events := newDynamicActionResumeEventScenario(t, stream, WithHostResumeVerifier(v))
				bound = trustedResumeResult(types.HostResumeRequest{Checkpoint: e.dynamicCheckpoints["run-events"].checkpoint})
				original := types.HostAdmissionProvenance{AdmissionReference: "admission", PolicyRevision: "v1", SourceScopeDigest: "source"}
				current := original
				bound.Binding.Provenance, bound.Current.Provenance = &original, &current
				want := types.HostResumeScopeDrift
				switch field {
				case "admission":
					current.AdmissionReference = "other"
					want = types.HostResumeBindingMismatch
				case "policy":
					current.PolicyRevision = "v2"
					want = types.HostResumeAuthorityDenied
				case "scope":
					current.SourceScopeDigest = "other"
				}
				d := eventDecision(cp)
				before := len(events.events)
				if _, err := e.ResumeDynamicAction(context.Background(), d, events, stream); !errors.Is(err, want) {
					t.Fatalf("err=%v want=%v", err, want)
				}
				if len(events.events) != before || e.dynamicCheckpoints[d.RunID].resumed {
					t.Fatal("rejection changed source")
				}
				current = original
				result, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
				if err != nil || result.FinalAnswer != "confirmed" || calls != 2 || events.count("run.dynamic_action.resolved") != 1 {
					t.Fatalf("retry=%+v err=%v calls=%d", result, err, calls)
				}
			})
		}
	}
}

// Each stage is a new OS process with only a reopened source store and the
// trusted paused host record. No retained Engine or callback state can help.
func TestHostResumeProvenanceCrossProcess(t *testing.T) {
	if os.Getenv("BAYMAX_PROVENANCE_HELPER") == "1" {
		hostResumeProvenanceProcess(t)
		return
	}
	for _, stream := range []bool{false, true} {
		t.Run(strconv.FormatBool(stream), func(t *testing.T) {
			dir := t.TempDir()
			for _, stage := range []string{"pause", "admission", "policy", "scope", "restore", "duplicate"} {
				cmd := exec.Command(os.Args[0], "-test.run=^TestHostResumeProvenanceCrossProcess$", "-test.v")
				cmd.Env = append(os.Environ(), "BAYMAX_PROVENANCE_HELPER=1", "BAYMAX_PROVENANCE_DIR="+dir, "BAYMAX_PROVENANCE_STAGE="+stage, "BAYMAX_PROVENANCE_STREAM="+strconv.FormatBool(stream))
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%s: %v\n%s", stage, err, output)
				}
			}
		})
	}
}

func hostResumeProvenanceProcess(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	dir, stage := os.Getenv("BAYMAX_PROVENANCE_DIR"), os.Getenv("BAYMAX_PROVENANCE_STAGE")
	stream, _ := strconv.ParseBool(os.Getenv("BAYMAX_PROVENANCE_STREAM"))
	store, err := checkpoint.OpenFileStore(filepath.Join(dir, "source"), checkpoint.Options{Namespace: "runner", Capacity: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	options := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}
	type record struct {
		Request types.HostResumeRequest
		Binding types.HostResumeResult
	}
	path := filepath.Join(dir, "trusted-host.json")
	var host record
	if stage == "pause" {
		e, cp, _ := newDynamicActionResumeEventScenario(t, stream, WithNativeContinuation(options))
		ref, readErr := e.NativeCheckpoint(ctx, "run-events")
		if readErr != nil {
			t.Fatal(readErr)
		}
		host.Request = types.HostResumeRequest{NativeCheckpoint: &ref, Decision: eventDecision(cp)}
		host.Request.Decision.CheckpointID, host.Request.Decision.CheckpointVersion, host.Request.Decision.CheckpointDigest = ref.CheckpointID, ref.SchemaVersion, ref.Digest
		paused, readErr := store.Load(ctx, ref)
		if readErr != nil {
			t.Fatal(readErr)
		}
		host.Binding = trustedResumeResult(types.HostResumeRequest{Checkpoint: paused.Checkpoint})
		p := types.HostAdmissionProvenance{AdmissionReference: "trusted-admission", PolicyRevision: "policy-v1", SourceScopeDigest: "listen-scope"}
		host.Binding.Binding.Provenance, host.Binding.Current.Provenance = &p, &p
		raw, encodeErr := json.Marshal(host)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if err = os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	p := *host.Binding.Binding.Provenance
	host.Binding.Current.Provenance = &p
	want := types.HostResumeReason("")
	switch stage {
	case "admission":
		p.AdmissionReference = "other"
		want = types.HostResumeBindingMismatch
	case "policy":
		p.PolicyRevision = "v2"
		want = types.HostResumeAuthorityDenied
	case "scope":
		p.SourceScopeDigest = "other"
		want = types.HostResumeScopeDrift
	}
	modelCalls := 0
	model := &fakeModel{
		generate: func(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
			modelCalls++
			if len(r.ToolResult) != 1 {
				t.Fatal("saved tool outcome missing")
			}
			return types.ModelResponse{FinalAnswer: "restored"}, nil
		},
		stream: func(_ context.Context, r types.ModelRequest, emit func(types.ModelEvent) error) error {
			modelCalls++
			if len(r.ToolResult) != 1 {
				t.Fatal("saved tool outcome missing")
			}
			return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "restored"})
		},
	}
	e := New(model, WithNativeContinuation(options), WithHostResumeVerifier(types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
		return host.Binding, nil
	})))
	before, err := store.Load(ctx, *host.Request.NativeCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	events := &dynamicActionEventCollector{}
	result, err := e.ResumeFromCheckpoint(ctx, host.Request, events, stream)
	if want != "" {
		if !errors.Is(err, want) || modelCalls != 0 || len(events.events) != 0 {
			t.Fatalf("err=%v want=%v calls=%d events=%d", err, want, modelCalls, len(events.events))
		}
		after, readErr := store.Load(ctx, *host.Request.NativeCheckpoint)
		if readErr != nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("rejection changed checkpoint: %v", readErr)
		}
		if _, readErr = store.ReadResumeClaim(ctx, *host.Request.NativeCheckpoint); !errors.Is(readErr, types.ErrCheckpointNotFound) {
			t.Fatalf("rejection consumed claim: %v", readErr)
		}
		return
	}
	expectedCalls := 1
	if stage == "duplicate" {
		expectedCalls = 0
	}
	if err != nil || result.FinalAnswer != "restored" || result.Iterations != 2 || modelCalls != expectedCalls {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, modelCalls)
	}
}
