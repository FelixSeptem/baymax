package redaction

import (
	"encoding/json"
	"strings"
)

const maskValue = "***"

var defaultKeywords = []string{"token", "password", "secret", "api_key", "apikey"}

type Matcher interface {
	MatchKey(key string) bool
}

type Option func(*Redactor)

func WithMatcher(m Matcher) Option {
	return func(r *Redactor) {
		if m != nil {
			r.matchers = append(r.matchers, m)
		}
	}
}

type Redactor struct {
	enabled  bool
	keywords []string
	matchers []Matcher
}

func New(enabled bool, keywords []string, opts ...Option) *Redactor {
	r := &Redactor{
		enabled: enabled,
	}
	r.keywords = NormalizeKeywords(keywords)
	if len(r.keywords) == 0 {
		r.keywords = append([]string(nil), defaultKeywords...)
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

func NormalizeKeywords(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		chunks := strings.Split(item, ",")
		for _, chunk := range chunks {
			v := strings.ToLower(strings.TrimSpace(chunk))
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func DefaultKeywords() []string {
	return append([]string(nil), defaultKeywords...)
}

func (r *Redactor) SanitizeMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if r.isSensitiveKey(k) {
			out[k] = maskValue
			continue
		}
		out[k] = r.sanitizeAny(v)
	}
	return out
}

func (r *Redactor) SanitizeJSONText(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return in
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return in
	}
	sanitized := r.sanitizeAny(payload)
	raw, err := json.Marshal(sanitized)
	if err != nil {
		return in
	}
	return string(raw)
}

func (r *Redactor) sanitizeAny(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		return r.SanitizeMap(tv)
	case []any:
		out := make([]any, 0, len(tv))
		for _, item := range tv {
			out = append(out, r.sanitizeAny(item))
		}
		return out
	default:
		return v
	}
}

func (r *Redactor) isSensitiveKey(key string) bool {
	if !r.enabled {
		return false
	}
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	for _, kw := range r.keywords {
		if strings.Contains(k, kw) {
			return true
		}
	}
	for _, m := range r.matchers {
		if m.MatchKey(k) {
			return true
		}
	}
	return false
}
