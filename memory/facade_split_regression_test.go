package memory

import (
	"errors"
	"testing"
)

func TestFacadeSplitRegressionDegradeWithoutMemoryFallbackContractFields(t *testing.T) {
	f := &Facade{
		mode:            ModeExternalSPI,
		provider:        "mem0",
		profile:         ProfileMem0,
		contractVersion: ContractVersionMemoryV1,
		fallbackPolicy:  FallbackPolicyDegradeWithoutMemory,
	}

	queryResp, err := f.queryWithFallback(QueryRequest{OperationID: "q-op", Namespace: "ns"}, errors.New("upstream timeout"))
	if err != nil {
		t.Fatalf("queryWithFallback error: %v", err)
	}
	if !queryResp.FallbackUsed || queryResp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("query fallback metadata mismatch: %#v", queryResp)
	}
	if queryResp.Mode != ModeExternalSPI || queryResp.Provider != "mem0" {
		t.Fatalf("query effective mode/provider mismatch: %#v", queryResp)
	}
	if queryResp.ReasonCode != ReasonCodeFallbackUsed || queryResp.Total != 0 || len(queryResp.Records) != 0 {
		t.Fatalf("query degrade_without_memory mismatch: %#v", queryResp)
	}

	upsertResp, err := f.upsertWithFallback(UpsertRequest{OperationID: "u-op", Namespace: "ns"}, errors.New("upstream timeout"))
	if err != nil {
		t.Fatalf("upsertWithFallback error: %v", err)
	}
	if !upsertResp.FallbackUsed || upsertResp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("upsert fallback metadata mismatch: %#v", upsertResp)
	}
	if upsertResp.Mode != ModeExternalSPI || upsertResp.Provider != "mem0" {
		t.Fatalf("upsert effective mode/provider mismatch: %#v", upsertResp)
	}
	if upsertResp.ReasonCode != ReasonCodeFallbackUsed || upsertResp.Upserted != 0 {
		t.Fatalf("upsert degrade_without_memory mismatch: %#v", upsertResp)
	}

	deleteResp, err := f.deleteWithFallback(DeleteRequest{OperationID: "d-op", Namespace: "ns"}, errors.New("upstream timeout"))
	if err != nil {
		t.Fatalf("deleteWithFallback error: %v", err)
	}
	if !deleteResp.FallbackUsed || deleteResp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("delete fallback metadata mismatch: %#v", deleteResp)
	}
	if deleteResp.Mode != ModeExternalSPI || deleteResp.Provider != "mem0" {
		t.Fatalf("delete effective mode/provider mismatch: %#v", deleteResp)
	}
	if deleteResp.ReasonCode != ReasonCodeFallbackUsed || deleteResp.Deleted != 0 {
		t.Fatalf("delete degrade_without_memory mismatch: %#v", deleteResp)
	}
}

func TestFacadeSplitRegressionDegradeToBuiltinFallbackIdentity(t *testing.T) {
	builtin := &fakeEngine{
		queryResp: QueryResponse{
			ReasonCode: "",
			Records:    []Record{{ID: "r-1", Content: "memory"}},
			Total:      1,
		},
		upsertResp: UpsertResponse{
			ReasonCode: "",
			Upserted:   2,
		},
		deleteResp: DeleteResponse{
			ReasonCode: "",
			Deleted:    3,
		},
	}
	f := &Facade{
		mode:            ModeExternalSPI,
		provider:        "mem0",
		profile:         ProfileMem0,
		contractVersion: ContractVersionMemoryV1,
		fallbackPolicy:  FallbackPolicyDegradeToBuiltin,
		builtin:         builtin,
	}

	queryResp, err := f.queryWithFallback(QueryRequest{OperationID: "q-op", Namespace: "ns"}, errors.New("upstream down"))
	if err != nil {
		t.Fatalf("queryWithFallback error: %v", err)
	}
	if queryResp.Mode != ModeBuiltinFilesystem || queryResp.Provider != ModeBuiltinFilesystem {
		t.Fatalf("query fallback identity mismatch: %#v", queryResp)
	}
	if !queryResp.FallbackUsed || queryResp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("query fallback metadata mismatch: %#v", queryResp)
	}
	if queryResp.ReasonCode != ReasonCodeOK || queryResp.Total != 1 || len(queryResp.Records) != 1 {
		t.Fatalf("query fallback payload mismatch: %#v", queryResp)
	}

	upsertResp, err := f.upsertWithFallback(UpsertRequest{OperationID: "u-op", Namespace: "ns"}, errors.New("upstream down"))
	if err != nil {
		t.Fatalf("upsertWithFallback error: %v", err)
	}
	if upsertResp.Mode != ModeBuiltinFilesystem || upsertResp.Provider != ModeBuiltinFilesystem {
		t.Fatalf("upsert fallback identity mismatch: %#v", upsertResp)
	}
	if !upsertResp.FallbackUsed || upsertResp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("upsert fallback metadata mismatch: %#v", upsertResp)
	}
	if upsertResp.ReasonCode != ReasonCodeOK || upsertResp.Upserted != 2 {
		t.Fatalf("upsert fallback payload mismatch: %#v", upsertResp)
	}

	deleteResp, err := f.deleteWithFallback(DeleteRequest{OperationID: "d-op", Namespace: "ns"}, errors.New("upstream down"))
	if err != nil {
		t.Fatalf("deleteWithFallback error: %v", err)
	}
	if deleteResp.Mode != ModeBuiltinFilesystem || deleteResp.Provider != ModeBuiltinFilesystem {
		t.Fatalf("delete fallback identity mismatch: %#v", deleteResp)
	}
	if !deleteResp.FallbackUsed || deleteResp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("delete fallback metadata mismatch: %#v", deleteResp)
	}
	if deleteResp.ReasonCode != ReasonCodeOK || deleteResp.Deleted != 3 {
		t.Fatalf("delete fallback payload mismatch: %#v", deleteResp)
	}
}

func TestFacadeSplitRegressionFailFastFallbackOperationMapping(t *testing.T) {
	f := &Facade{
		mode:            ModeExternalSPI,
		provider:        "mem0",
		profile:         ProfileMem0,
		contractVersion: ContractVersionMemoryV1,
		fallbackPolicy:  FallbackPolicyFailFast,
	}

	for _, tc := range []struct {
		name      string
		operation string
		run       func() error
	}{
		{
			name:      "query",
			operation: OperationQuery,
			run: func() error {
				_, err := f.queryWithFallback(QueryRequest{Namespace: "ns"}, errors.New("boom"))
				return err
			},
		},
		{
			name:      "upsert",
			operation: OperationUpsert,
			run: func() error {
				_, err := f.upsertWithFallback(UpsertRequest{Namespace: "ns"}, errors.New("boom"))
				return err
			},
		},
		{
			name:      "delete",
			operation: OperationDelete,
			run: func() error {
				_, err := f.deleteWithFallback(DeleteRequest{Namespace: "ns"}, errors.New("boom"))
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("expected fail_fast error")
			}
			var memErr *Error
			if !errors.As(err, &memErr) {
				t.Fatalf("error type = %T, want *memory.Error", err)
			}
			if memErr.Operation != tc.operation {
				t.Fatalf("operation = %q, want %q", memErr.Operation, tc.operation)
			}
			if memErr.Code != ReasonCodeStorageUnavailable {
				t.Fatalf("code = %q, want %q", memErr.Code, ReasonCodeStorageUnavailable)
			}
		})
	}
}
