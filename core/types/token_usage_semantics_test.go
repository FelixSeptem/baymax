package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

// jsonFieldShape returns the ordered "Field=jsontag" shape of a struct type.
// It is used to pin protocol-facing structs against silent field renames,
// additions, or tag changes.
func jsonFieldShape(t reflect.Type) []string {
	shape := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		shape = append(shape, field.Name+"="+field.Tag.Get("json"))
	}
	return shape
}

// TestTokenUsageJSONShapeIsFrozen pins the provider-usage shape that the
// cache-usage observability contract deliberately does NOT touch. Cache
// accounting lives in the additive, defaultable ModelResponse.CacheUsage
// carrier and the request-side conformance projection; it must not be merged
// into TokenUsage without an explicit contract change.
func TestTokenUsageJSONShapeIsFrozen(t *testing.T) {
	want := []string{
		"InputTokens=input_tokens",
		"OutputTokens=output_tokens",
		"TotalTokens=total_tokens",
	}
	if got := jsonFieldShape(reflect.TypeOf(TokenUsage{})); !reflect.DeepEqual(got, want) {
		t.Fatalf("TokenUsage shape changed: got %v want %v", got, want)
	}
}

// TestUsageCarrierJSONShapeIsFrozen pins the two carriers that surface token
// usage. Both must stay non-pointer without omitempty: a zero usage is a real
// observation ("no accounting reported"), not an absent field.
func TestUsageCarrierJSONShapeIsFrozen(t *testing.T) {
	modelResponse, ok := reflect.TypeOf(ModelResponse{}).FieldByName("Usage")
	if !ok {
		t.Fatal("ModelResponse.Usage disappeared")
	}
	if modelResponse.Tag.Get("json") != "usage" {
		t.Fatalf("ModelResponse.Usage json tag changed: %q", modelResponse.Tag.Get("json"))
	}
	if modelResponse.Type.Kind() == reflect.Pointer {
		t.Fatal("ModelResponse.Usage became a pointer; absence semantics changed")
	}
	if modelResponse.Type != reflect.TypeOf(TokenUsage{}) {
		t.Fatalf("ModelResponse.Usage type changed: %v", modelResponse.Type)
	}

	runResult, ok := reflect.TypeOf(RunResult{}).FieldByName("TokenUsage")
	if !ok {
		t.Fatal("RunResult.TokenUsage disappeared")
	}
	if runResult.Tag.Get("json") != "token_usage" {
		t.Fatalf("RunResult.TokenUsage json tag changed: %q", runResult.Tag.Get("json"))
	}
	if runResult.Type.Kind() == reflect.Pointer {
		t.Fatal("RunResult.TokenUsage became a pointer; absence semantics changed")
	}
	if runResult.Type != reflect.TypeOf(TokenUsage{}) {
		t.Fatalf("RunResult.TokenUsage type changed: %v", runResult.Type)
	}
}

// TestTokenUsageDecodingSemanticsUnchanged pins the additive + nullable +
// default decoding behaviour: a payload written before cache accounting
// existed must still decode without error, missing fields must fall back to
// zero, and unknown provider fields must be ignored rather than rejected.
func TestTokenUsageDecodingSemanticsUnchanged(t *testing.T) {
	t.Run("missing fields default to zero", func(t *testing.T) {
		var usage TokenUsage
		if err := json.Unmarshal([]byte(`{}`), &usage); err != nil {
			t.Fatalf("decoding an empty usage object failed: %v", err)
		}
		if usage != (TokenUsage{}) {
			t.Fatalf("expected zero usage, got %+v", usage)
		}
	})

	t.Run("historical payload still decodes", func(t *testing.T) {
		var response ModelResponse
		raw := `{"final_answer":"ok","usage":{"input_tokens":11,"output_tokens":4,"total_tokens":15}}`
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			t.Fatalf("historical payload failed to decode: %v", err)
		}
		want := TokenUsage{InputTokens: 11, OutputTokens: 4, TotalTokens: 15}
		if response.Usage != want {
			t.Fatalf("usage changed: got %+v want %+v", response.Usage, want)
		}
	})

	t.Run("unknown cache fields are ignored", func(t *testing.T) {
		// A future provider payload that carries cache accounting must not
		// break decoding, and must not silently populate the existing fields.
		var response ModelResponse
		raw := `{"usage":{"input_tokens":11,"output_tokens":4,"total_tokens":15,` +
			`"cache_read_input_tokens":900,"cache_creation_input_tokens":300,"future_field":true}}`
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			t.Fatalf("payload with unknown cache fields failed to decode: %v", err)
		}
		want := TokenUsage{InputTokens: 11, OutputTokens: 4, TotalTokens: 15}
		if response.Usage != want {
			t.Fatalf("unknown fields leaked into usage: got %+v want %+v", response.Usage, want)
		}
	})

	t.Run("absent run usage decodes to zero rather than failing", func(t *testing.T) {
		var result RunResult
		raw := `{"run_id":"run-1","iterations":1,"latency_ms":7}`
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatalf("run result without token usage failed to decode: %v", err)
		}
		if result.TokenUsage != (TokenUsage{}) {
			t.Fatalf("expected zero token usage, got %+v", result.TokenUsage)
		}
	})
}
