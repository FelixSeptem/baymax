package types

import (
	"errors"
	"testing"
)

func TestDeploymentScopeValidatesLocalSequentialProfile(t *testing.T) {
	scope := SameHostSingleWriterScope("runner", "fixture:deployment-scope.v1")
	if err := scope.Validate(); err != nil {
		t.Fatalf("local scope should validate: %v", err)
	}
	if !scope.Supports(DeploymentRequest{Profile: DeploymentProfileSameHostSingleWriterSequentialReopen}) {
		t.Fatal("local scope should support sequential reopen")
	}
	if scope.Supports(DeploymentRequest{Profile: DeploymentProfileCrossMachine}) {
		t.Fatal("local scope must not support cross-machine restore")
	}
}

func TestDeploymentScopeRejectsForeignAndStaleWriter(t *testing.T) {
	scope := SameHostSingleWriterScope("runner", "fixture:deployment-scope.v1")
	if err := scope.ValidateWriter("other", "fence-1"); !errors.Is(err, ErrForeignWriter) {
		t.Fatalf("foreign writer error = %v", err)
	}
	if err := scope.ValidateWriter("runner", "old"); !errors.Is(err, ErrStaleWriter) {
		t.Fatalf("stale writer error = %v", err)
	}
}

func TestDeploymentScopeRejectsUnsupportedDistributedRequest(t *testing.T) {
	scope := SameHostSingleWriterScope("runner", "fixture:deployment-scope.v1")
	err := scope.Check(DeploymentRequest{
		Profile:    DeploymentProfileMultiInstance,
		WriterMode: WriterScopeMultiWriter,
	})
	if !errors.Is(err, ErrDeploymentUnsupported) {
		t.Fatalf("unsupported request error = %v", err)
	}
}

func TestCoordinatedScopeSupportsMultiInstanceButNotCrossMachine(t *testing.T) {
	scope := CoordinatedMultiWriterScope("runner", "fixture:durable-source-store-coordination.v1")
	if err := scope.Validate(); err != nil {
		t.Fatalf("coordinated scope = %v", err)
	}
	if !scope.Supports(DeploymentRequest{Profile: DeploymentProfileMultiInstance, WriterMode: WriterScopeMultiWriter, Fencing: FencingScopeDistributed, Transaction: TransactionScopeMultiWriterCAS}) {
		t.Fatal("coordinated scope did not support multi-instance CAS")
	}
	if scope.Supports(DeploymentRequest{Profile: DeploymentProfileCrossMachine}) {
		t.Fatal("coordinated file scope advertised cross-machine support")
	}
}

func TestCrossMachineScopeRequiresDistributedFencing(t *testing.T) {
	scope := CoordinatedCrossMachineScope("runner", "redis-coordination.v1")
	if err := scope.Validate(); err != nil {
		t.Fatalf("cross-machine scope should validate: %v", err)
	}
	if scope.Profile != DeploymentProfileCrossMachine || scope.FencingScope != FencingScopeDistributed || scope.TransactionScope != TransactionScopeDistributed {
		t.Fatalf("cross-machine scope dimensions = %+v", scope)
	}
	if !scope.Supports(DeploymentRequest{
		Profile:        DeploymentProfileCrossMachine,
		WriterMode:     WriterScopeMultiWriter,
		Fencing:        FencingScopeDistributed,
		Transaction:    TransactionScopeDistributed,
		OwnerNamespace: "runner",
		WriterFence:    scope.WriterFence,
	}) {
		t.Fatal("cross-machine scope did not support its own request")
	}

	// Source-owner fencing is same-host evidence and must not claim cross-machine.
	degraded := CoordinatedCrossMachineScope("runner", "redis-coordination.v1")
	degraded.FencingScope = FencingScopeSourceOwner
	if err := degraded.Validate(); !errors.Is(err, ErrDeploymentUnsupported) {
		t.Fatalf("source-owner fencing cross-machine scope error = %v", err)
	}

	// CAS transaction scope is likewise insufficient without distributed scope.
	degradedTx := CoordinatedCrossMachineScope("runner", "redis-coordination.v1")
	degradedTx.TransactionScope = TransactionScopeMultiWriterCAS
	if err := degradedTx.Validate(); !errors.Is(err, ErrDeploymentUnsupported) {
		t.Fatalf("CAS transaction cross-machine scope error = %v", err)
	}
}

func TestMultiInstanceScopeDoesNotClaimCrossMachine(t *testing.T) {
	local := CoordinatedMultiInstanceScope("runner", "fixture:durable-source-store-coordination.v1")
	if local.Supports(DeploymentRequest{Profile: DeploymentProfileCrossMachine}) {
		t.Fatal("same-host multi-instance scope advertised cross-machine support")
	}
	cross := CoordinatedCrossMachineScope("runner", "redis-coordination.v1")
	if cross.Supports(DeploymentRequest{Profile: DeploymentProfileMultiInstance}) {
		t.Fatal("cross-machine scope must not be downgraded to mixed multi-instance requests")
	}
}
