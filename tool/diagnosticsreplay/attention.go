package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/FelixSeptem/baymax/runtime/diagnostics"
)

// AttentionFixture is a bounded offline snapshot and its complete expected view.
type AttentionFixture struct {
	Version  string                       `json:"version"`
	Request  diagnostics.AttentionRequest `json:"request"`
	Expected diagnostics.AttentionResult  `json:"expected"`
}

// ReplayAttentionJSON checks evidence without reading owners or executing actions.
// Errors contain a fixed classification and never echo fixture contents.
func ReplayAttentionJSON(raw []byte) error {
	drift := errors.New("attention.replay_drift")
	if len(raw) > 1<<20 || !utf8.Valid(raw) {
		return drift
	}
	scanner := json.NewDecoder(bytes.NewReader(raw))
	if !scanAttentionJSON(scanner, 0) {
		return drift
	}
	if _, err := scanner.Token(); err != io.EOF {
		return drift
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture AttentionFixture
	if err := decoder.Decode(&fixture); err != nil || fixture.Version != "attention.v1" {
		return drift
	}
	actual, err := diagnostics.ProjectAttention(fixture.Request)
	if err != nil || !reflect.DeepEqual(actual, fixture.Expected) {
		return drift
	}
	return nil
}

// Token validation detects duplicate (including case-alias) object keys before
// Go's typed decoder can silently replace them. Depth and byte limits bound work.
func scanAttentionJSON(decoder *json.Decoder, depth int) bool {
	if depth > 64 {
		return false
	}
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			name, ok := key.(string)
			// All schema keys are ASCII. Reject Unicode aliases before encoding/json
			// applies Unicode SimpleFold (for example long-s aliases to ASCII s).
			for _, c := range name {
				if c > 127 {
					return false
				}
			}
			name = strings.ToLower(name)
			if err != nil || !ok || seen[name] {
				return false
			}
			seen[name] = true
			if !scanAttentionJSON(decoder, depth+1) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim('}')
	case '[':
		for decoder.More() {
			if !scanAttentionJSON(decoder, depth+1) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim(']')
	default:
		return false
	}
}
