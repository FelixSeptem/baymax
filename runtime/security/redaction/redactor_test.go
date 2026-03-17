package redaction

import "testing"

type customMatcher struct{}

func (customMatcher) MatchKey(key string) bool {
	return key == "credential"
}

func TestSanitizeMapKeywordHitAndMiss(t *testing.T) {
	r := New(true, []string{"token"})
	in := map[string]any{
		"token": "abc",
		"name":  "ok",
	}
	out := r.SanitizeMap(in)
	if out["token"] != "***" {
		t.Fatalf("token = %#v, want ***", out["token"])
	}
	if out["name"] != "ok" {
		t.Fatalf("name = %#v, want ok", out["name"])
	}
}

func TestSanitizeMapExtendedKeywords(t *testing.T) {
	r := New(true, []string{"credential"})
	in := map[string]any{"credential_id": "abc"}
	out := r.SanitizeMap(in)
	if out["credential_id"] != "***" {
		t.Fatalf("credential_id = %#v, want ***", out["credential_id"])
	}
}

func TestSanitizeMapTokenKeywordKeepsTokenizerMode(t *testing.T) {
	r := New(true, []string{"token"})
	in := map[string]any{
		"tokenizer_mode": "mixed_cjk_en",
		"bearer_token":   "abc",
	}
	out := r.SanitizeMap(in)
	if out["tokenizer_mode"] != "mixed_cjk_en" {
		t.Fatalf("tokenizer_mode = %#v, want mixed_cjk_en", out["tokenizer_mode"])
	}
	if out["bearer_token"] != "***" {
		t.Fatalf("bearer_token = %#v, want ***", out["bearer_token"])
	}
}

func TestSanitizeMapWithMatcherExtension(t *testing.T) {
	r := New(true, []string{"token"}, WithMatcher(customMatcher{}))
	in := map[string]any{"credential": "xyz"}
	out := r.SanitizeMap(in)
	if out["credential"] != "***" {
		t.Fatalf("credential = %#v, want ***", out["credential"])
	}
}

func TestSanitizeJSONText(t *testing.T) {
	r := New(true, []string{"password"})
	s := r.SanitizeJSONText(`{"password":"p","name":"n"}`)
	if s != `{"name":"n","password":"***"}` && s != `{"password":"***","name":"n"}` {
		t.Fatalf("sanitized json = %q", s)
	}
}
