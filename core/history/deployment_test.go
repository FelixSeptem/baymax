package history

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func ownerStoreSource(id string) types.SourceRef {
	return types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: id}
}

func TestOpenOwnerStoreRequiresExplicitDurableBinding(t *testing.T) {
	_, err := OpenOwnerStore(OwnerStoreOptions{Source: ownerStoreSource("unbound")})
	if !errors.Is(err, types.ErrHistoryUnsupported) {
		t.Fatalf("unbound owner store error = %v, want ErrHistoryUnsupported", err)
	}
}

func TestOpenOwnerStoreRejectsCrossMachineRequestForFileJournal(t *testing.T) {
	source := ownerStoreSource("local-profile")
	_, err := OpenOwnerStore(OwnerStoreOptions{
		Source:      source,
		JournalPath: filepath.Join(t.TempDir(), "history.json"),
		DeploymentRequest: types.DeploymentRequest{
			Profile: types.DeploymentProfileCrossMachine,
		},
	})
	if !errors.Is(err, types.ErrDeploymentUnsupported) {
		t.Fatalf("cross-machine request error = %v, want ErrDeploymentUnsupported", err)
	}
}

func TestOwnerStoreReopensAndReconcilesCommittedFact(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := ownerStoreSource("reconcile")
	journalPath := filepath.Join(root, "history.json")
	ledgerPath := filepath.Join(root, "ledger.json")

	first, err := OpenOwnerStore(OwnerStoreOptions{Source: source, JournalPath: journalPath})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := first.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.completed"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := first.ReadAfter(ctx, nil, 10)
	if err != nil || len(page.Events) != 1 || page.Events[0].History.EventID != identity.EventID {
		t.Fatalf("committed source page = %+v err=%v", page, err)
	}
	duplicateIdentity, err := first.Append(ctx, page.Events[0])
	if err != nil || duplicateIdentity != identity {
		t.Fatalf("duplicate append = %+v err=%v, want original identity %+v", duplicateIdentity, err, identity)
	}
	conflicting := page.Events[0]
	conflicting.NativeType = "runner.conflicting"
	if _, err = first.Append(ctx, conflicting); !errors.Is(err, types.ErrHistoryConflict) {
		t.Fatalf("conflicting append error = %v, want ErrHistoryConflict", err)
	}

	reopened, err := OpenOwnerStore(OwnerStoreOptions{Source: source, JournalPath: journalPath, ProjectionLedgerPath: ledgerPath})
	if err != nil {
		t.Fatal(err)
	}
	result, err := reopened.ReconcileProjection(ctx, page.Events[0])
	if err != nil || result.Duplicate {
		t.Fatalf("first reconciliation = %+v err=%v", result, err)
	}
	duplicate, err := reopened.ReconcileProjection(ctx, page.Events[0])
	if err != nil || !duplicate.Duplicate || duplicate.Record.Digest != result.Record.Digest {
		t.Fatalf("duplicate reconciliation = %+v err=%v", duplicate, err)
	}
	caps, err := reopened.Capabilities(ctx)
	if err != nil || !caps.ProjectionRecovery || caps.Deployment.Profile != types.DeploymentProfileSameHostSingleWriterSequentialReopen {
		t.Fatalf("reopened capabilities = %+v err=%v", caps, err)
	}
}

func TestOwnerStoreCoordinatedProfileIsSameHostMultiInstance(t *testing.T) {
	source := ownerStoreSource("coordinated")
	coordinator, err := sourcecoordination.OpenFile(filepath.Join(t.TempDir(), "coord"), sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenOwnerStore(OwnerStoreOptions{Source: source, Coordinator: coordinator, WriterID: "writer", LeaseTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	caps, err := store.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caps.Deployment.Profile != types.DeploymentProfileMultiInstance || caps.Deployment.FencingScope != types.FencingScopeSourceOwner || caps.Deployment.TransactionScope != types.TransactionScopeMultiWriterCAS {
		t.Fatalf("coordinated deployment scope = %+v", caps.Deployment)
	}
}

func TestOwnerStoreRejectsStaleWriterAfterFreshBinding(t *testing.T) {
	ctx := context.Background()
	source := ownerStoreSource("stale-writer")
	coordPath := filepath.Join(t.TempDir(), "coord")
	firstCoordinator, err := sourcecoordination.OpenFile(coordPath, sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	first, err := OpenOwnerStore(OwnerStoreOptions{Source: source, Coordinator: firstCoordinator, WriterID: "first", LeaseTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.first"}); err != nil {
		t.Fatal(err)
	}
	secondCoordinator, err := sourcecoordination.OpenFile(coordPath, sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	second, err := OpenOwnerStore(OwnerStoreOptions{Source: source, Coordinator: secondCoordinator, WriterID: "second", LeaseTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = second.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.second"}); err != nil {
		t.Fatal(err)
	}
	if _, err = first.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.stale"}); !errors.Is(err, types.ErrSourceUnavailable) && !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("stale append error = %v, want source unavailable or stale writer", err)
	}
	page, err := second.ReadAfter(ctx, nil, 10)
	if err != nil || len(page.Events) != 2 {
		t.Fatalf("committed source facts = %+v err=%v", page, err)
	}
}

func TestOpenOwnerStoreAdmitsCrossMachineWithRedisCoordinator(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	source := ownerStoreSource("cross-machine")
	coordinator, err := sourcecoordination.NewRedis(client, sourcecoordination.RedisOptions{
		Options:   sourcecoordination.Options{OwnerNamespace: source.ID},
		KeyPrefix: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenOwnerStore(OwnerStoreOptions{
		Source:      source,
		Coordinator: coordinator,
		WriterID:    "writer-a",
		LeaseTTL:    time.Hour,
		DeploymentRequest: types.DeploymentRequest{
			Profile: types.DeploymentProfileCrossMachine,
		},
	})
	if err != nil {
		t.Fatalf("redis-coordinated owner store = %v", err)
	}
	caps, err := store.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caps.Deployment.Profile != types.DeploymentProfileCrossMachine || caps.Deployment.FencingScope != types.FencingScopeDistributed || caps.Deployment.TransactionScope != types.TransactionScopeDistributed {
		t.Fatalf("cross-machine deployment scope = %+v", caps.Deployment)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOwnerStoreDeploymentFixtureCoversFailClosedProfiles(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "owner_store_deployment.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version string `json:"version"`
		Cases   []struct {
			Name        string `json:"name"`
			Outcome     string `json:"outcome"`
			SideEffects bool   `json:"side_effects"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != "owner_store_deployment.v1" || len(fixture.Cases) != 8 {
		t.Fatalf("fixture coverage = %#v", fixture)
	}
	admittedCrossMachine := false
	for _, item := range fixture.Cases {
		if item.Name == "" || item.Outcome == "" || item.SideEffects {
			t.Fatalf("unsafe owner-store fixture case = %#v", item)
		}
		if item.Name == "cross_machine_redis_coordinator" && item.Outcome == "admitted" {
			admittedCrossMachine = true
		}
	}
	if !admittedCrossMachine {
		t.Fatal("fixture must keep an admitted cross_machine case backed by a distributed coordinator")
	}
}
