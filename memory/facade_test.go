package memory

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type fakeEngine struct {
	queryResp  QueryResponse
	queryErr   error
	upsertResp UpsertResponse
	upsertErr  error
	deleteResp DeleteResponse
	deleteErr  error
}

func (f *fakeEngine) Query(req QueryRequest) (QueryResponse, error) {
	if f.queryErr != nil {
		return QueryResponse{}, f.queryErr
	}
	resp := f.queryResp
	if resp.Namespace == "" {
		resp.Namespace = req.Namespace
	}
	return resp, nil
}

func (f *fakeEngine) Upsert(req UpsertRequest) (UpsertResponse, error) {
	if f.upsertErr != nil {
		return UpsertResponse{}, f.upsertErr
	}
	resp := f.upsertResp
	if resp.Namespace == "" {
		resp.Namespace = req.Namespace
	}
	return resp, nil
}

func (f *fakeEngine) Delete(req DeleteRequest) (DeleteResponse, error) {
	if f.deleteErr != nil {
		return DeleteResponse{}, f.deleteErr
	}
	resp := f.deleteResp
	if resp.Namespace == "" {
		resp.Namespace = req.Namespace
	}
	return resp, nil
}

func TestResolveProfileUnknown(t *testing.T) {
	if _, err := ResolveProfile("missing-profile"); err == nil {
		t.Fatal("expected ResolveProfile unknown error")
	} else {
		var memErr *Error
		if !errors.As(err, &memErr) || memErr.Code != ReasonCodeProfileUnknown {
			t.Fatalf("error = %#v, want ReasonCodeProfileUnknown", err)
		}
	}
}

func TestFacadeFallbackDegradeToBuiltin(t *testing.T) {
	cfg := Config{
		Mode: ModeExternalSPI,
		External: ExternalConfig{
			Provider:        "mem0",
			Profile:         ProfileMem0,
			ContractVersion: ContractVersionMemoryV1,
		},
		Builtin: BuiltinConfig{
			RootDir: filepath.Join(t.TempDir(), "memory"),
			Compaction: FilesystemCompactionConfig{
				Enabled: true,
			},
		},
		Fallback: FallbackConfig{Policy: FallbackPolicyDegradeToBuiltin},
	}
	f, err := NewFacade(cfg, func(cfg ExternalConfig) (Engine, error) {
		return &fakeEngine{
			queryErr:  errors.New("upstream down"),
			upsertErr: errors.New("upstream down"),
		}, nil
	})
	if err != nil {
		t.Fatalf("NewFacade failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	_, err = f.Upsert(UpsertRequest{
		Namespace: "session",
		Records: []Record{
			{ID: "r-1", Content: "hello world"},
		},
	})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	resp, err := f.Query(QueryRequest{Namespace: "session", Query: "hello"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !resp.FallbackUsed {
		t.Fatalf("fallback_used = false, want true: %#v", resp)
	}
	if resp.FallbackReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("fallback_reason_code = %q, want %q", resp.FallbackReasonCode, ReasonCodeFallbackUsed)
	}
	if resp.Mode != ModeBuiltinFilesystem {
		t.Fatalf("mode = %q, want %q", resp.Mode, ModeBuiltinFilesystem)
	}
	if len(resp.Records) != 1 || resp.Records[0].ID != "r-1" {
		t.Fatalf("records mismatch: %#v", resp.Records)
	}
}

func TestFacadeFallbackFailFast(t *testing.T) {
	f, err := NewFacade(Config{
		Mode: ModeExternalSPI,
		External: ExternalConfig{
			Provider:        "zep",
			Profile:         ProfileZep,
			ContractVersion: ContractVersionMemoryV1,
		},
		Builtin: BuiltinConfig{
			RootDir: filepath.Join(t.TempDir(), "memory"),
			Compaction: FilesystemCompactionConfig{
				Enabled: true,
			},
		},
		Fallback: FallbackConfig{Policy: FallbackPolicyFailFast},
	}, func(cfg ExternalConfig) (Engine, error) {
		return &fakeEngine{queryErr: errors.New("unavailable")}, nil
	})
	if err != nil {
		t.Fatalf("NewFacade failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	_, err = f.Query(QueryRequest{Namespace: "ns"})
	if err == nil {
		t.Fatal("expected fail_fast query error")
	}
	var memErr *Error
	if !errors.As(err, &memErr) {
		t.Fatalf("error type = %T, want *memory.Error", err)
	}
}

func TestFacadeFallbackDegradeWithoutMemory(t *testing.T) {
	f, err := NewFacade(Config{
		Mode: ModeExternalSPI,
		External: ExternalConfig{
			Provider:        "openviking",
			Profile:         ProfileOpenViking,
			ContractVersion: ContractVersionMemoryV1,
		},
		Fallback: FallbackConfig{Policy: FallbackPolicyDegradeWithoutMemory},
	}, func(cfg ExternalConfig) (Engine, error) {
		return &fakeEngine{queryErr: errors.New("timeout")}, nil
	})
	if err != nil {
		t.Fatalf("NewFacade failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	resp, err := f.Query(QueryRequest{Namespace: "ns"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !resp.FallbackUsed {
		t.Fatalf("fallback_used = false, want true")
	}
	if resp.Total != 0 || len(resp.Records) != 0 {
		t.Fatalf("degrade_without_memory should return empty records: %#v", resp)
	}
	if resp.ReasonCode != ReasonCodeFallbackUsed {
		t.Fatalf("reason_code = %q, want %q", resp.ReasonCode, ReasonCodeFallbackUsed)
	}
}

func TestFacadeRejectsUnsupportedContractVersion(t *testing.T) {
	_, err := NewFacade(Config{
		Mode: ModeExternalSPI,
		External: ExternalConfig{
			Provider:        "mem0",
			Profile:         ProfileMem0,
			ContractVersion: "memory.v2",
		},
		Fallback: FallbackConfig{Policy: FallbackPolicyFailFast},
	}, func(cfg ExternalConfig) (Engine, error) {
		return &fakeEngine{}, nil
	})
	if err == nil {
		t.Fatal("expected contract version mismatch error")
	}
	var memErr *Error
	if !errors.As(err, &memErr) || memErr.Code != ReasonCodeContractVersionMismatch {
		t.Fatalf("error = %#v, want contract mismatch", err)
	}
}

func TestFacadeBuiltinModeDefaultsProfileAndContract(t *testing.T) {
	f, err := NewFacade(Config{
		Mode: ModeBuiltinFilesystem,
		Builtin: BuiltinConfig{
			RootDir: filepath.Join(t.TempDir(), "memory"),
			Compaction: FilesystemCompactionConfig{
				Enabled: true,
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("NewFacade failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	_, err = f.Upsert(UpsertRequest{
		Namespace: "session",
		Records: []Record{
			{ID: "r-1", Content: "abc"},
		},
	})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	resp, err := f.Query(QueryRequest{Namespace: "session"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Profile != ProfileGeneric {
		t.Fatalf("profile = %q, want %q", resp.Profile, ProfileGeneric)
	}
	if resp.ContractVersion != ContractVersionMemoryV1 {
		t.Fatalf("contract_version = %q, want %q", resp.ContractVersion, ContractVersionMemoryV1)
	}
	if strings.TrimSpace(resp.Provider) != ModeBuiltinFilesystem {
		t.Fatalf("provider = %q, want %q", resp.Provider, ModeBuiltinFilesystem)
	}
}
