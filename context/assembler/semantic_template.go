package assembler

import (
	"fmt"
	"strings"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type semanticPromptTemplate struct {
	raw       string
	allowed   map[string]struct{}
	variables []string
}

func newSemanticPromptTemplate(cfg runtimeconfig.ContextAssemblerCA3SemanticTemplateConfig) (*semanticPromptTemplate, error) {
	raw := strings.TrimSpace(cfg.Prompt)
	if raw == "" {
		return nil, fmt.Errorf("semantic template prompt is empty")
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedPlaceholders))
	for _, placeholder := range cfg.AllowedPlaceholders {
		token := strings.ToLower(strings.TrimSpace(placeholder))
		if token == "" {
			continue
		}
		allowed[token] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("semantic template allowed_placeholders is empty")
	}
	variables, err := parseTemplateVariables(raw, allowed)
	if err != nil {
		return nil, err
	}
	return &semanticPromptTemplate{
		raw:       raw,
		allowed:   allowed,
		variables: variables,
	}, nil
}

func (t *semanticPromptTemplate) Render(vars map[string]string) (string, error) {
	if t == nil {
		return "", fmt.Errorf("semantic template is nil")
	}
	out := t.raw
	for _, token := range t.variables {
		value := strings.TrimSpace(vars[token])
		out = strings.ReplaceAll(out, "{{"+token+"}}", value)
		out = strings.ReplaceAll(out, "{{ "+token+" }}", value)
	}
	return strings.TrimSpace(out), nil
}

func parseTemplateVariables(raw string, allowed map[string]struct{}) ([]string, error) {
	if strings.Count(raw, "{{") != strings.Count(raw, "}}") {
		return nil, fmt.Errorf("semantic template has unbalanced placeholders")
	}
	parts := strings.Split(raw, "{{")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for i := 1; i < len(parts); i++ {
		right := strings.SplitN(parts[i], "}}", 2)
		if len(right) != 2 {
			return nil, fmt.Errorf("semantic template has invalid placeholder")
		}
		key := strings.ToLower(strings.TrimSpace(right[0]))
		if key == "" {
			return nil, fmt.Errorf("semantic template has empty placeholder")
		}
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("semantic template placeholder %q not allowed", key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out, nil
}
