package guard

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

var (
	ErrGuardViolation = errors.New("context guard violation")

	sensitivePattern = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret)\s*[:=]\s*([^\s,;]+)`)
)

type Result struct {
	Input          string
	Messages       []types.Message
	GuardViolation string
}

type Guard struct {
	FailFast bool
}

func New(failFast bool) Guard {
	return Guard{FailFast: failFast}
}

func (g Guard) Apply(req types.ContextAssembleRequest, prefixHash, expectedHash string) (Result, error) {
	out := Result{
		Input:    sanitizeText(req.Input),
		Messages: sanitizeMessages(req.Messages),
	}
	if strings.TrimSpace(req.RunID) == "" {
		out.GuardViolation = "schema.run_id.required"
		return out, g.failFastError(out.GuardViolation)
	}
	if strings.TrimSpace(req.PrefixVersion) == "" {
		out.GuardViolation = "schema.prefix_version.required"
		return out, g.failFastError(out.GuardViolation)
	}
	if strings.TrimSpace(prefixHash) == "" {
		out.GuardViolation = "hash.prefix.empty"
		return out, g.failFastError(out.GuardViolation)
	}
	if strings.TrimSpace(expectedHash) != "" && expectedHash != prefixHash {
		out.GuardViolation = "hash.prefix.drift"
		return out, g.failFastError(out.GuardViolation)
	}
	return out, nil
}

func (g Guard) failFastError(violation string) error {
	if !g.FailFast || strings.TrimSpace(violation) == "" {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrGuardViolation, violation)
}

func sanitizeMessages(messages []types.Message) []types.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]types.Message, 0, len(messages))
	for _, msg := range messages {
		out = append(out, types.Message{
			Role:    msg.Role,
			Content: sanitizeText(msg.Content),
		})
	}
	return out
}

func sanitizeText(input string) string {
	if strings.TrimSpace(input) == "" {
		return input
	}
	return sensitivePattern.ReplaceAllString(input, "$1=[REDACTED]")
}
