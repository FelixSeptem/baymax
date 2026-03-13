package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestNewProviderAcceptsExternalAliases(t *testing.T) {
	cfg := runtimeconfig.ContextAssemblerCA2ExternalConfig{
		Endpoint: "http://127.0.0.1:8080/retrieve",
		Method:   "POST",
		Mapping: runtimeconfig.ContextAssemblerCA2ExternalMappingConfig{
			Request: runtimeconfig.ContextAssemblerCA2RequestMappingConfig{
				Mode:       "plain",
				QueryField: "query",
			},
			Response: runtimeconfig.ContextAssemblerCA2ResponseMappingConfig{
				ChunksField: "chunks",
			},
		},
	}
	for _, name := range []string{
		runtimeconfig.ContextStage2ProviderHTTP,
		runtimeconfig.ContextStage2ProviderRAG,
		runtimeconfig.ContextStage2ProviderDB,
		runtimeconfig.ContextStage2ProviderElasticsearch,
	} {
		p, err := NewWithConfig(Config{Name: name, External: cfg})
		if err != nil {
			t.Fatalf("new provider(%s): %v", name, err)
		}
		if p.Name() != name {
			t.Fatalf("provider name = %q, want %q", p.Name(), name)
		}
	}
}

func TestFileProviderFetch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"s1","content":"c1"}`,
		`{"session_id":"s1","content":"c2"}`,
		`{"session_id":"s2","content":"x"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	p, err := New("file", path)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	resp, err := p.Fetch(context.Background(), Request{SessionID: "s1", MaxItems: 2})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(resp.Chunks) != 2 || resp.Chunks[0] != "c1" || resp.Chunks[1] != "c2" {
		t.Fatalf("unexpected chunks: %#v", resp.Chunks)
	}
	if resp.Meta["source"] != "file" || resp.Meta["reason"] != "ok" {
		t.Fatalf("unexpected meta: %#v", resp.Meta)
	}
}

func TestHTTPProviderFetchPlainMapping(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("authorization header = %q", got)
		}
		if got := r.Header.Get("X-Tenant"); got != "tenant-a" {
			t.Fatalf("x-tenant header = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var reqPayload map[string]any
		if err := json.Unmarshal(body, &reqPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if reqPayload["q"] != "hello" || reqPayload["session"] != "s-1" || reqPayload["run"] != "r-1" {
			t.Fatalf("unexpected request payload: %#v", reqPayload)
		}
		resp := map[string]any{
			"data": map[string]any{
				"chunks": []any{"c1", "c2"},
				"src":    "ragflow",
				"why":    "ok",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	p, err := NewWithConfig(Config{
		Name: runtimeconfig.ContextStage2ProviderHTTP,
		External: runtimeconfig.ContextAssemblerCA2ExternalConfig{
			Endpoint: ts.URL,
			Method:   "POST",
			Auth: runtimeconfig.ContextAssemblerCA2ExternalAuthConfig{
				BearerToken: "token-1",
			},
			Headers: map[string]string{"X-Tenant": "tenant-a"},
			Mapping: runtimeconfig.ContextAssemblerCA2ExternalMappingConfig{
				Request: runtimeconfig.ContextAssemblerCA2RequestMappingConfig{
					Mode:           "plain",
					QueryField:     "q",
					SessionIDField: "session",
					RunIDField:     "run",
					MaxItemsField:  "topk",
				},
				Response: runtimeconfig.ContextAssemblerCA2ResponseMappingConfig{
					ChunksField: "data.chunks",
					SourceField: "data.src",
					ReasonField: "data.why",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	resp, err := p.Fetch(context.Background(), Request{
		RunID:     "r-1",
		SessionID: "s-1",
		Input:     "hello",
		MaxItems:  3,
	})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(resp.Chunks) != 2 || resp.Chunks[0] != "c1" || resp.Chunks[1] != "c2" {
		t.Fatalf("unexpected chunks: %#v", resp.Chunks)
	}
	if resp.Meta["source"] != "ragflow" || resp.Meta["reason"] != "ok" {
		t.Fatalf("unexpected meta: %#v", resp.Meta)
	}
}

func TestHTTPProviderFetchJSONRPC2Mapping(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var reqPayload map[string]any
		if err := json.Unmarshal(body, &reqPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if reqPayload["jsonrpc"] != "2.0" || reqPayload["method"] != "retrieve.search" {
			t.Fatalf("unexpected jsonrpc envelope: %#v", reqPayload)
		}
		params, ok := reqPayload["params"].(map[string]any)
		if !ok {
			t.Fatalf("params missing: %#v", reqPayload)
		}
		query, _ := params["query"].(string)
		if query != "hello-rpc" {
			t.Fatalf("params.query = %q, want hello-rpc", query)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"chunks": []any{"rpc-chunk"},
				"source": "external-rpc",
				"reason": "ok",
			},
		})
	}))
	defer ts.Close()

	p, err := NewWithConfig(Config{
		Name: runtimeconfig.ContextStage2ProviderRAG,
		External: runtimeconfig.ContextAssemblerCA2ExternalConfig{
			Endpoint: ts.URL,
			Method:   "POST",
			Auth: runtimeconfig.ContextAssemblerCA2ExternalAuthConfig{
				BearerToken: "",
				HeaderName:  "X-Token",
			},
			Mapping: runtimeconfig.ContextAssemblerCA2ExternalMappingConfig{
				Request: runtimeconfig.ContextAssemblerCA2RequestMappingConfig{
					Mode:           "jsonrpc2",
					MethodName:     "retrieve.search",
					JSONRPCVersion: "2.0",
					QueryField:     "query",
					SessionIDField: "session_id",
					RunIDField:     "run_id",
					MaxItemsField:  "max_items",
				},
				Response: runtimeconfig.ContextAssemblerCA2ResponseMappingConfig{
					ChunksField: "result.chunks",
					SourceField: "result.source",
					ReasonField: "result.reason",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	resp, err := p.Fetch(context.Background(), Request{
		RunID:     "r-rpc",
		SessionID: "s-rpc",
		Input:     "hello-rpc",
		MaxItems:  1,
	})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(resp.Chunks) != 1 || resp.Chunks[0] != "rpc-chunk" {
		t.Fatalf("unexpected chunks: %#v", resp.Chunks)
	}
	if resp.Meta["source"] != "external-rpc" {
		t.Fatalf("unexpected source: %#v", resp.Meta)
	}
	if resp.Meta["profile"] != runtimeconfig.ContextStage2ExternalProfileHTTPGeneric {
		t.Fatalf("profile = %#v, want http_generic", resp.Meta["profile"])
	}
}

func TestHTTPProviderFetchClassifiesSemanticError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "upstream denied",
			},
		})
	}))
	defer ts.Close()

	p, err := NewWithConfig(Config{
		Name: runtimeconfig.ContextStage2ProviderHTTP,
		External: runtimeconfig.ContextAssemblerCA2ExternalConfig{
			Endpoint: ts.URL,
			Profile:  runtimeconfig.ContextStage2ExternalProfileRAGFlowLike,
			Mapping: runtimeconfig.ContextAssemblerCA2ExternalMappingConfig{
				Request: runtimeconfig.ContextAssemblerCA2RequestMappingConfig{
					Mode:       "plain",
					QueryField: "query",
				},
				Response: runtimeconfig.ContextAssemblerCA2ResponseMappingConfig{
					ChunksField:       "chunks",
					ErrorField:        "error",
					ErrorMessageField: "error.message",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	_, err = p.Fetch(context.Background(), Request{Input: "q"})
	if err == nil {
		t.Fatal("expected semantic error, got nil")
	}
	var fetchErr *FetchError
	if !errors.As(err, &fetchErr) {
		t.Fatalf("error = %T, want *FetchError", err)
	}
	if fetchErr.Layer != ErrorLayerSemantic || fetchErr.Code != "upstream_error" {
		t.Fatalf("fetchErr = %#v", fetchErr)
	}
}

func TestHTTPProviderFetchClassifiesTransportTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{"chunks": []string{"x"}})
	}))
	defer ts.Close()

	p, err := NewWithConfig(Config{
		Name: runtimeconfig.ContextStage2ProviderHTTP,
		External: runtimeconfig.ContextAssemblerCA2ExternalConfig{
			Endpoint: ts.URL,
			Profile:  runtimeconfig.ContextStage2ExternalProfileHTTPGeneric,
			Mapping: runtimeconfig.ContextAssemblerCA2ExternalMappingConfig{
				Request: runtimeconfig.ContextAssemblerCA2RequestMappingConfig{Mode: "plain", QueryField: "query"},
				Response: runtimeconfig.ContextAssemblerCA2ResponseMappingConfig{
					ChunksField: "chunks",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = p.Fetch(ctx, Request{Input: "q"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	var fetchErr *FetchError
	if !errors.As(err, &fetchErr) {
		t.Fatalf("error = %T, want *FetchError", err)
	}
	if fetchErr.Layer != ErrorLayerTransport || fetchErr.Code != "timeout" {
		t.Fatalf("fetchErr = %#v", fetchErr)
	}
}
