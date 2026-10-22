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
