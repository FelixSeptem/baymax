package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	TypeMCP   = "mcp"
	TypeModel = "model"
	TypeTool  = "tool"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Options struct {
	Type    string
	Name    string
	Output  string
	Force   bool
	BaseDir string
}

type File struct {
	RelativePath string
	Content      string
}

type Plan struct {
	OutputDir string
	Files     []File
	Conflicts []string
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "[adapter-scaffold][invalid-args] " + e.Message
}

type ConflictError struct {
	Paths []string
}

func (e *ConflictError) Error() string {
	return "[adapter-scaffold][conflict] existing files: " + strings.Join(e.Paths, ", ")
}

func DefaultOutputPath(baseDir, scaffoldType, name string) string {
	return filepath.Join(baseDir, "examples", "adapters", fmt.Sprintf("%s-%s", scaffoldType, name))
}

func Generate(opts Options) (Plan, error) {
	plan, err := BuildPlan(opts)
	if err != nil {
		return Plan{}, err
	}
	if len(plan.Conflicts) > 0 && !opts.Force {
		return Plan{}, &ConflictError{Paths: plan.Conflicts}
	}
	if err := writePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func BuildPlan(opts Options) (Plan, error) {
	norm, err := normalizeOptions(opts)
	if err != nil {
		return Plan{}, err
	}

	templates, err := renderTemplates(norm)
	if err != nil {
		return Plan{}, err
	}

	relPaths := make([]string, 0, len(templates))
	for rel := range templates {
		relPaths = append(relPaths, rel)
	}
	sort.Strings(relPaths)

	plan := Plan{
		OutputDir: norm.Output,
		Files:     make([]File, 0, len(relPaths)),
	}
	for _, rel := range relPaths {
		target := filepath.Join(norm.Output, rel)
		if fileExists(target) {
			plan.Conflicts = append(plan.Conflicts, target)
		}
		plan.Files = append(plan.Files, File{
			RelativePath: rel,
			Content:      templates[rel],
		})
	}
	return plan, nil
}

func normalizeOptions(opts Options) (Options, error) {
	baseDir := strings.TrimSpace(opts.BaseDir)
	if baseDir == "" {
		baseDir = "."
	}
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return Options{}, fmt.Errorf("resolve base directory: %w", err)
	}

	typ := strings.ToLower(strings.TrimSpace(opts.Type))
	if typ != TypeMCP && typ != TypeModel && typ != TypeTool {
		return Options{}, &ValidationError{
			Message: "flag -type must be one of: mcp|model|tool",
		}
	}

	name := strings.ToLower(strings.TrimSpace(opts.Name))
	if !validName.MatchString(name) {
		return Options{}, &ValidationError{
			Message: "flag -name must match ^[a-z][a-z0-9-]*$",
		}
	}

	output := strings.TrimSpace(opts.Output)
	if output == "" {
		output = DefaultOutputPath(baseAbs, typ, name)
	} else if !filepath.IsAbs(output) {
		output = filepath.Join(baseAbs, output)
	}
	output = filepath.Clean(output)

	return Options{
		Type:    typ,
		Name:    name,
		Output:  output,
		Force:   opts.Force,
		BaseDir: baseAbs,
	}, nil
}

func writePlan(plan Plan) error {
	for _, item := range plan.Files {
		target := filepath.Join(plan.OutputDir, item.RelativePath)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", target, err)
		}
		if err := os.WriteFile(target, []byte(item.Content), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

func IsConflictError(err error) bool {
	var ce *ConflictError
	return errors.As(err, &ce)
}

type templateData struct {
	AdapterName       string
	AdapterTypeName   string
	PackageName       string
	Type              string
	ConformanceID     string
	ConformanceLabel  string
	ConformanceCatRef string
	BaymaxCompat      string
	RequiredCaps      []string
	OptionalCaps      []string
}

func renderTemplates(opts Options) (map[string]string, error) {
	data, err := buildTemplateData(opts)
	if err != nil {
		return nil, err
	}
	templates := map[string]string{
		"README.md":                      renderReadme(data),
		"adapter.go":                     renderAdapterSource(data),
		"adapter_test.go":                renderAdapterTestSource(data),
		"conformance_bootstrap_test.go":  renderConformanceBootstrapTest(data),
		"capability_negotiation_test.go": renderCapabilityNegotiationTest(data),
		"adapter-manifest.json":          renderManifest(data),
	}
	return templates, nil
}

func buildTemplateData(opts Options) (templateData, error) {
	conformanceID := ""
	conformanceLabel := ""
	conformanceCatRef := ""
	switch opts.Type {
	case TypeMCP:
		conformanceID = "mcp-normalization-fail-fast"
		conformanceLabel = "MCP normalization + fail-fast"
		conformanceCatRef = "CategoryMCP"
	case TypeModel:
		conformanceID = "model-run-stream-downgrade"
		conformanceLabel = "Model run/stream + downgrade"
		conformanceCatRef = "CategoryModel"
	case TypeTool:
		conformanceID = "tool-invoke-fail-fast"
		conformanceLabel = "Tool invoke + fail-fast"
		conformanceCatRef = "CategoryTool"
	default:
		return templateData{}, fmt.Errorf("unsupported scaffold type: %s", opts.Type)
	}
	adapterTypeName := toUpperCamel(opts.Name) + toUpperCamel(opts.Type) + "Adapter"
	requiredCaps := []string{}
	optionalCaps := []string{}
	switch opts.Type {
	case TypeMCP:
		requiredCaps = []string{"mcp.invoke.required_input", "mcp.response.normalized"}
		optionalCaps = []string{"mcp.transport.sse"}
	case TypeModel:
		requiredCaps = []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"}
		optionalCaps = []string{"model.capability.token_count"}
	case TypeTool:
		requiredCaps = []string{"tool.invoke.required_input"}
		optionalCaps = []string{"tool.schema.rich_validation"}
	}
	return templateData{
		AdapterName:       opts.Name,
		AdapterTypeName:   adapterTypeName,
		PackageName:       packageName(opts.Name),
		Type:              opts.Type,
		ConformanceID:     conformanceID,
		ConformanceLabel:  conformanceLabel,
		ConformanceCatRef: conformanceCatRef,
		BaymaxCompat:      ">=0.26.0-rc.1 <0.27.0",
		RequiredCaps:      requiredCaps,
		OptionalCaps:      optionalCaps,
	}, nil
}

func renderReadme(data templateData) string {
	return fmt.Sprintf(`# %s Adapter Scaffold

This directory is generated by Baymax A23 scaffold generator.

## Scaffold Contract

- type: %s
- name: %s
- conformance bootstrap mapping: %s (%s)
- manifest: adapter-manifest.json (runtime compatibility + capability contract)
- generation policy: deterministic + offline + fail-fast conflict

## Next Steps

1. Replace placeholder adapter logic in adapter.go with real integration code.
2. Expand adapter_test.go with category-specific behavior checks.
3. Keep conformance_bootstrap_test.go and ensure it keeps mapping to A22 minimum matrix.
4. Keep adapter-manifest.json aligned with conformance profile and capability assertions.
5. Use capability_negotiation_test.go as A27 fallback/override baseline.

## Validation

Run local package tests:
- go test . -count=1

Run repository harness checks:
- bash scripts/check-adapter-conformance.sh
- bash scripts/check-adapter-manifest-contract.sh
- bash scripts/check-adapter-scaffold-drift.sh
- pwsh -File scripts/check-adapter-conformance.ps1
- pwsh -File scripts/check-adapter-manifest-contract.ps1
- pwsh -File scripts/check-adapter-scaffold-drift.ps1
`, data.AdapterTypeName, data.Type, data.AdapterName, data.ConformanceID, data.ConformanceLabel)
}

func renderAdapterSource(data templateData) string {
	switch data.Type {
	case TypeMCP:
		return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	mcpstdio "github.com/FelixSeptem/baymax/mcp/stdio"
)

type %sTransport struct{}

func (%sTransport) Initialize(context.Context) error { return nil }

func (%sTransport) ListTools(context.Context) ([]types.MCPToolMeta, error) {
	return []types.MCPToolMeta{
		{
			Name:        "echo",
			Description: "generated mcp adapter placeholder",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"input": map[string]any{"type": "string"},
				},
				"required": []any{"input"},
			},
		},
	}, nil
}

func (%sTransport) CallTool(_ context.Context, name string, args map[string]any) (mcpstdio.Response, error) {
	raw, ok := args["input"]
	if !ok {
		return mcpstdio.Response{Error: "missing required input"}, nil
	}
	input, ok := raw.(string)
	if !ok {
		return mcpstdio.Response{Error: "missing required input"}, nil
	}
	return mcpstdio.Response{
		Content: fmt.Sprintf("tool=%%s input=%%s", name, input),
		Structured: map[string]any{
			"tool":  name,
			"input": input,
		},
	}, nil
}

func (%sTransport) Close() error { return nil }

func New%sClient() *mcpstdio.Client {
	return mcpstdio.NewClient(%sTransport{}, mcpstdio.Config{
		ReadPoolSize:  1,
		WritePoolSize: 1,
		CallTimeout:   2 * time.Second,
		Retry:         0,
		Backoff:       time.Millisecond,
		QueueSize:     8,
		Backpressure:  types.BackpressureBlock,
	})
}
`, data.PackageName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterTypeName)
	case TypeModel:
		return fmt.Sprintf(`package %s

import (
	"context"

	"github.com/FelixSeptem/baymax/core/types"
)

type %s struct{}

func (%s) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{
		FinalAnswer: "%s model adapter placeholder",
	}, nil
}

func (%s) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "%s model adapter placeholder",
	})
}
`, data.PackageName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterName, data.AdapterTypeName, data.AdapterName)
	case TypeTool:
		return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

type %s struct{}

func (%s) Name() string { return "local.%s" }

func (%s) Description() string {
	return "generated tool adapter placeholder"
}

func (%s) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
		},
		"required": []any{"input"},
	}
}

func (%s) Invoke(_ context.Context, args map[string]any) (types.ToolResult, error) {
	raw, ok := args["input"]
	if !ok {
		return types.ToolResult{}, fmt.Errorf("missing required input")
	}
	input, ok := raw.(string)
	if !ok || strings.TrimSpace(input) == "" {
		return types.ToolResult{}, fmt.Errorf("missing required input")
	}
	return types.ToolResult{
		Content: fmt.Sprintf("echo=%%s", input),
	}, nil
}
`, data.PackageName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterName, data.AdapterTypeName, data.AdapterTypeName, data.AdapterTypeName)
	default:
		return ""
	}
}

func renderAdapterTestSource(data templateData) string {
	switch data.Type {
	case TypeMCP:
		return fmt.Sprintf(`package %s

import (
	"context"
	"testing"
)

func Test%sClientEcho(t *testing.T) {
	client := New%sClient()
	defer func() { _ = client.Close() }()

	res, err := client.CallTool(context.Background(), "echo", map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("call tool: %%v", err)
	}
	if res.Content != "tool=echo input=hello" {
		t.Fatalf("unexpected content: %%s", res.Content)
	}
}
`, data.PackageName, data.AdapterTypeName, data.AdapterTypeName)
	case TypeModel:
		return fmt.Sprintf(`package %s

import (
	"context"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
)

func Test%sRunAndStreamSemanticEquivalent(t *testing.T) {
	eng := runner.New(%s{})
	runRes, err := eng.Run(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("run: %%v", err)
	}
	streamRes, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("stream: %%v", err)
	}
	if normalize(runRes.FinalAnswer) != normalize(streamRes.FinalAnswer) {
		t.Fatalf("semantic mismatch run=%%q stream=%%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}
}

func normalize(in string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(in))), " ")
}
`, data.PackageName, data.AdapterTypeName, data.AdapterTypeName)
	case TypeTool:
		return fmt.Sprintf(`package %s

import (
	"context"
	"testing"
)

func Test%sInvokeFailFast(t *testing.T) {
	tool := %s{}
	res, err := tool.Invoke(context.Background(), map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("invoke: %%v", err)
	}
	if res.Content != "echo=hello" {
		t.Fatalf("unexpected content: %%s", res.Content)
	}
	if _, err := tool.Invoke(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected fail-fast for missing required input")
	}
}
`, data.PackageName, data.AdapterTypeName, data.AdapterTypeName)
	default:
		return ""
	}
}

func renderConformanceBootstrapTest(data templateData) string {
	return fmt.Sprintf(`package %s

import (
	"testing"

	"github.com/FelixSeptem/baymax/integration/adapterconformance"
)

func TestConformanceBootstrapAlignment(t *testing.T) {
	if err := adapterconformance.ValidateMinimumMatrix(adapterconformance.MinimumMatrix); err != nil {
		t.Fatalf("minimum matrix invalid: %%v", err)
	}

	// A22 mapping hint for this scaffold category.
	const expectedScenarioID = "%s"
	found := false
	for _, row := range adapterconformance.MinimumMatrix {
		if row.ID != expectedScenarioID {
			continue
		}
		found = true
		if row.Category != adapterconformance.%s {
			t.Fatalf("category mismatch for %%s: got=%%s", expectedScenarioID, row.Category)
		}
		break
	}
	if !found {
		t.Fatalf("missing A22 scenario mapping: %%s", expectedScenarioID)
	}

	if err := adapterconformance.ValidateManifestProfileAlignmentForScaffold(".", expectedScenarioID); err != nil {
		t.Fatalf("manifest-profile alignment failed: %%v", err)
	}
}
`, data.PackageName, data.ConformanceID, data.ConformanceCatRef)
}

func renderManifest(data templateData) string {
	required := quotedList(data.RequiredCaps)
	optional := quotedList(data.OptionalCaps)
	return fmt.Sprintf(`{
  "type": %q,
  "name": %q,
  "version": "0.1.0",
  "baymax_compat": %q,
  "capabilities": {
    "required": [%s],
    "optional": [%s]
  },
  "negotiation": {
    "default_strategy": "fail_fast",
    "allow_request_override": true
  },
  "conformance_profile": %q
}
`, data.Type, data.AdapterName, data.BaymaxCompat, required, optional, data.ConformanceID)
}

func renderCapabilityNegotiationTest(data templateData) string {
	return fmt.Sprintf(`package %s

import (
	"reflect"
	"testing"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
)

func Test%sNegotiationFallbackAndOverride(t *testing.T) {
	declared := adaptercap.Set{
		Required: []string{%s},
		Optional: []string{%s},
	}

	// default strategy is fail_fast.
	defaultOutcome, err := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, adaptercap.Request{
		Required: []string{%s},
		Optional: []string{%s},
	})
	if err != nil {
		t.Fatalf("default negotiation failed: %%v", err)
	}
	if defaultOutcome.Accepted {
		t.Fatal("expected fail_fast to reject missing optional request")
	}

	// request-level override hook to best_effort.
	overrideOutcome, err := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, adaptercap.Request{
		Required:         []string{%s},
		Optional:         []string{%s},
		StrategyOverride: adaptercap.StrategyBestEffort,
	})
	if err != nil {
		t.Fatalf("override negotiation failed: %%v", err)
	}
	if !overrideOutcome.Accepted || !overrideOutcome.Downgraded {
		t.Fatalf("expected best_effort downgrade path, got %%#v", overrideOutcome)
	}
	if !containsReason(overrideOutcome.Reasons, adaptercap.ReasonOptionalDowngraded) ||
		!containsReason(overrideOutcome.Reasons, adaptercap.ReasonStrategyOverrideApply) {
		t.Fatalf("unexpected override reasons: %%#v", overrideOutcome.Reasons)
	}
}

func Test%sNegotiationRunStreamEquivalent(t *testing.T) {
	declared := adaptercap.Set{
		Required: []string{%s},
		Optional: []string{%s},
	}
	req := adaptercap.Request{
		Required:         []string{%s},
		Optional:         []string{%s},
		StrategyOverride: adaptercap.StrategyBestEffort,
	}
	runOutcome, runErr := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, req)
	streamOutcome, streamErr := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, req)
	if runErr != nil || streamErr != nil {
		t.Fatalf("unexpected run/stream negotiation error runErr=%%v streamErr=%%v", runErr, streamErr)
	}
	if !reflect.DeepEqual(runOutcome, streamOutcome) {
		t.Fatalf("run/stream negotiation mismatch run=%%#v stream=%%#v", runOutcome, streamOutcome)
	}
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}
`, data.PackageName,
		data.AdapterTypeName,
		quotedList(data.RequiredCaps),
		quotedList(data.OptionalCaps),
		quotedList(data.RequiredCaps),
		quotedList(data.OptionalCaps),
		quotedList(data.RequiredCaps),
		quotedList(data.OptionalCaps),
		data.AdapterTypeName,
		quotedList(data.RequiredCaps),
		quotedList(data.OptionalCaps),
		quotedList(data.RequiredCaps),
		quotedList(data.OptionalCaps))
}

func quotedList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, ", ")
}

func packageName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func toUpperCamel(name string) string {
	parts := strings.Split(name, "-")
	builder := strings.Builder{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			builder.WriteString(part[1:])
		}
	}
	return builder.String()
}
