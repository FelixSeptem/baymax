package adapterconformance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
	adapterprofile "github.com/FelixSeptem/baymax/adapter/profile"
	"github.com/FelixSeptem/baymax/core/types"
)

type Category string

const (
	CategoryMCP   Category = "mcp"
	CategoryModel Category = "model"
	CategoryTool  Category = "tool"
)

type Scenario struct {
	ID                     string
	Category               Category
	PriorityTier           int
	TemplatePath           string
	TraceComment           string
	RequiredCapabilities   []string
	OptionalCapabilities   []string
	ConformanceProfile     string
	DefaultStrategy        string
	AllowStrategyOverride  bool
	ContractProfileVersion string
}

var MinimumMatrix = []Scenario{
	{
		ID:                     "mcp-normalization-fail-fast",
		Category:               CategoryMCP,
		PriorityTier:           1,
		TemplatePath:           "examples/templates/mcp-adapter-template/main.go",
		TraceComment:           "A21 template linkage: MCP adapter template baseline and fail-fast boundary",
		RequiredCapabilities:   []string{"mcp.invoke.required_input", "mcp.response.normalized"},
		OptionalCapabilities:   []string{"mcp.transport.sse"},
		ConformanceProfile:     "mcp-normalization-fail-fast",
		DefaultStrategy:        adaptercap.StrategyFailFast,
		AllowStrategyOverride:  true,
		ContractProfileVersion: adapterprofile.CurrentProfile,
	},
	{
		ID:                     "model-run-stream-downgrade",
		Category:               CategoryModel,
		PriorityTier:           2,
		TemplatePath:           "examples/templates/model-adapter-template/main.go",
		TraceComment:           "A21 template linkage: Model adapter template run/stream equivalence and downgrade semantics",
		RequiredCapabilities:   []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		OptionalCapabilities:   []string{"model.capability.token_count"},
		ConformanceProfile:     "model-run-stream-downgrade",
		DefaultStrategy:        adaptercap.StrategyFailFast,
		AllowStrategyOverride:  true,
		ContractProfileVersion: adapterprofile.CurrentProfile,
	},
	{
		ID:                     "tool-invoke-fail-fast",
		Category:               CategoryTool,
		PriorityTier:           3,
		TemplatePath:           "examples/templates/tool-adapter-template/main.go",
		TraceComment:           "A21 template linkage: Tool adapter template invocation and mandatory-input fail-fast",
		RequiredCapabilities:   []string{"tool.invoke.required_input"},
		OptionalCapabilities:   []string{"tool.schema.rich_validation"},
		ConformanceProfile:     "tool-invoke-fail-fast",
		DefaultStrategy:        adaptercap.StrategyFailFast,
		AllowStrategyOverride:  true,
		ContractProfileVersion: adapterprofile.CurrentProfile,
	},
}

var reasonPattern = regexp.MustCompile(`^[a-z]+(\.[a-z_]+)+$`)

type ConformanceError struct {
	Class      types.ErrorClass
	ReasonCode string
	Message    string
}

func (e *ConformanceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type Classification struct {
	Class      types.ErrorClass
	ReasonCode string
}

type OptionalCapabilityResult struct {
	Downgraded bool
	ReasonCode string
}

func ValidateMinimumMatrix(matrix []Scenario) error {
	if len(matrix) == 0 {
		return errors.New("empty conformance matrix")
	}
	covered := map[Category]bool{
		CategoryMCP:   false,
		CategoryModel: false,
		CategoryTool:  false,
	}
	priority := map[Category]int{}
	for _, row := range matrix {
		if row.ID == "" {
			return errors.New("scenario id must not be empty")
		}
		if !reasonPattern.MatchString(strings.ReplaceAll(row.ID, "-", ".")) {
			return fmt.Errorf("scenario id %q must be deterministic and namespaced-like", row.ID)
		}
		if row.TemplatePath == "" {
			return fmt.Errorf("scenario %q missing template path mapping", row.ID)
		}
		if !strings.Contains(row.TraceComment, "A21 template linkage") {
			return fmt.Errorf("scenario %q missing trace comment marker", row.ID)
		}
		if strings.TrimSpace(row.ConformanceProfile) == "" {
			return fmt.Errorf("scenario %q missing conformance profile", row.ID)
		}
		if strings.TrimSpace(row.ConformanceProfile) != strings.TrimSpace(row.ID) {
			return fmt.Errorf("scenario %q conformance profile must align with scenario id", row.ID)
		}
		if len(row.RequiredCapabilities) == 0 {
			return fmt.Errorf("scenario %q missing required capability assertions", row.ID)
		}
		if row.OptionalCapabilities == nil {
			return fmt.Errorf("scenario %q missing optional capability assertions", row.ID)
		}
		if strings.TrimSpace(row.DefaultStrategy) == "" {
			row.DefaultStrategy = adaptercap.StrategyFailFast
		}
		if !adaptercap.IsStrategy(row.DefaultStrategy) {
			return fmt.Errorf("scenario %q has invalid default strategy %q", row.ID, row.DefaultStrategy)
		}
		if strings.TrimSpace(row.ContractProfileVersion) == "" {
			return fmt.Errorf("scenario %q missing contract profile version", row.ID)
		}
		if _, err := adapterprofile.Parse(row.ContractProfileVersion); err != nil {
			return fmt.Errorf("scenario %q has invalid contract profile version %q", row.ID, row.ContractProfileVersion)
		}
		covered[row.Category] = true
		priority[row.Category] = row.PriorityTier
	}
	for cat, ok := range covered {
		if !ok {
			return fmt.Errorf("missing minimum conformance category coverage: %s", cat)
		}
	}
	if priority[CategoryMCP] >= priority[CategoryModel] || priority[CategoryMCP] >= priority[CategoryTool] {
		return errors.New("conformance priority must satisfy MCP > Model > Tool")
	}
	return nil
}

func ScenarioByID(id string) (Scenario, bool) {
	target := strings.TrimSpace(id)
	for _, row := range MinimumMatrix {
		if strings.TrimSpace(row.ID) == target {
			return row, true
		}
	}
	return Scenario{}, false
}

func ValidateManifestProfileAlignment(manifest adaptermanifest.Manifest, scenario Scenario) error {
	if err := adaptermanifest.Validate(manifest); err != nil {
		return err
	}
	normalized := manifest
	normalized.Type = strings.ToLower(strings.TrimSpace(normalized.Type))
	normalized.ContractProfileVersion = strings.ToLower(strings.TrimSpace(normalized.ContractProfileVersion))
	normalized.ConformanceProfile = strings.ToLower(strings.TrimSpace(normalized.ConformanceProfile))
	normalized.Capabilities.Required = normalizedCapabilities(normalized.Capabilities.Required)
	normalized.Capabilities.Optional = normalizedCapabilities(normalized.Capabilities.Optional)
	if normalized.Type != string(scenario.Category) {
		return fmt.Errorf("manifest-profile-mismatch: manifest type %q does not match scenario category %q", normalized.Type, scenario.Category)
	}
	if normalized.ConformanceProfile != strings.TrimSpace(scenario.ConformanceProfile) {
		return fmt.Errorf("manifest-profile-mismatch: conformance_profile %q does not match scenario %q", normalized.ConformanceProfile, scenario.ConformanceProfile)
	}
	expectedContractProfile := strings.TrimSpace(scenario.ContractProfileVersion)
	if expectedContractProfile == "" {
		expectedContractProfile = adapterprofile.CurrentProfile
	}
	if normalized.ContractProfileVersion != expectedContractProfile {
		return fmt.Errorf("manifest-profile-mismatch: contract_profile_version %q does not match scenario %q", normalized.ContractProfileVersion, expectedContractProfile)
	}
	expectedStrategy := strings.TrimSpace(scenario.DefaultStrategy)
	if expectedStrategy == "" {
		expectedStrategy = adaptercap.StrategyFailFast
	}
	actualStrategy := strings.TrimSpace(normalized.Negotiation.DefaultStrategy)
	if actualStrategy == "" {
		actualStrategy = adaptercap.StrategyFailFast
	}
	if actualStrategy != expectedStrategy {
		return fmt.Errorf("manifest-profile-mismatch: negotiation.default_strategy %q does not match scenario strategy %q", actualStrategy, expectedStrategy)
	}
	if normalized.Negotiation.AllowRequestOverride != scenario.AllowStrategyOverride {
		return fmt.Errorf("manifest-profile-mismatch: negotiation.allow_request_override %v does not match scenario %v", normalized.Negotiation.AllowRequestOverride, scenario.AllowStrategyOverride)
	}
	missingRequired := missingCapabilities(scenario.RequiredCapabilities, normalized.Capabilities.Required)
	if len(missingRequired) > 0 {
		return fmt.Errorf("missing required-capability coverage: %s", strings.Join(missingRequired, ", "))
	}
	missingOptional := missingCapabilities(scenario.OptionalCapabilities, normalized.Capabilities.Optional)
	if len(missingOptional) > 0 {
		return fmt.Errorf("missing optional-capability coverage: %s", strings.Join(missingOptional, ", "))
	}
	return nil
}

func ValidateManifestProfileAlignmentFile(path string, scenario Scenario) error {
	manifest, err := adaptermanifest.LoadFile(path)
	if err != nil {
		return err
	}
	return ValidateManifestProfileAlignment(manifest, scenario)
}

func ActivateAdapterManifest(path, runtimeVersion, scenarioID string, availableCapabilities []string) (adaptermanifest.ActivationResult, error) {
	return ActivateAdapterManifestWithRequest(path, runtimeVersion, scenarioID, availableCapabilities, adaptercap.Request{})
}

func ActivateAdapterManifestWithRequest(path, runtimeVersion, scenarioID string, availableCapabilities []string, request adaptercap.Request) (adaptermanifest.ActivationResult, error) {
	scenario, ok := ScenarioByID(scenarioID)
	if !ok {
		return adaptermanifest.ActivationResult{}, fmt.Errorf("unknown conformance scenario: %s", scenarioID)
	}
	manifest, err := adaptermanifest.LoadFile(path)
	if err != nil {
		return adaptermanifest.ActivationResult{}, err
	}
	if err := ValidateManifestProfileAlignment(manifest, scenario); err != nil {
		return adaptermanifest.ActivationResult{}, err
	}
	required := request.Required
	if required == nil {
		required = append([]string(nil), scenario.RequiredCapabilities...)
	}
	optional := request.Optional
	if optional == nil {
		optional = append([]string(nil), scenario.OptionalCapabilities...)
	}
	return adaptermanifest.ActivateWithRequest(manifest, runtimeVersion, availableCapabilities, adaptermanifest.CapabilityRequest{
		Required:         append([]string(nil), required...),
		Optional:         append([]string(nil), optional...),
		StrategyOverride: strings.TrimSpace(request.StrategyOverride),
	})
}

func ValidateManifestProfileAlignmentForScaffold(rootDir, scenarioID string) error {
	scenario, ok := ScenarioByID(scenarioID)
	if !ok {
		return fmt.Errorf("unknown conformance scenario: %s", scenarioID)
	}
	manifestPath := filepath.Join(rootDir, "adapter-manifest.json")
	return ValidateManifestProfileAlignmentFile(manifestPath, scenario)
}

func missingCapabilities(expected, actual []string) []string {
	actualSet := map[string]struct{}{}
	for _, capability := range actual {
		actualSet[strings.ToLower(strings.TrimSpace(capability))] = struct{}{}
	}
	missing := make([]string, 0)
	for _, capability := range expected {
		key := strings.ToLower(strings.TrimSpace(capability))
		if key == "" {
			continue
		}
		if _, ok := actualSet[key]; ok {
			continue
		}
		missing = append(missing, key)
	}
	sort.Strings(missing)
	return missing
}

func normalizedCapabilities(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func EnsureScenarioManifestExists(rootDir string, scenario Scenario) error {
	manifestPath := filepath.Join(rootDir, "adapter-manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("manifest-profile-mismatch: missing manifest file %s", manifestPath)
	}
	return ValidateManifestProfileAlignmentFile(manifestPath, scenario)
}

func NormalizeSemanticText(in string) string {
	trimmed := strings.TrimSpace(strings.ToLower(in))
	return strings.Join(strings.Fields(trimmed), " ")
}

func ValidateReasonCode(reason string) error {
	if !reasonPattern.MatchString(strings.TrimSpace(reason)) {
		return fmt.Errorf("invalid reason taxonomy: %s", reason)
	}
	return nil
}

func ClassifyAdapterResult(result types.ToolResult, err error, fallbackClass types.ErrorClass, fallbackReason string) Classification {
	if ce := asConformanceError(err); ce != nil {
		return Classification{Class: ce.Class, ReasonCode: ce.ReasonCode}
	}
	if result.Error != nil {
		reason := reasonFromDetails(result.Error.Details)
		if reason == "" {
			reason = inferMissingRequiredReason(result.Error.Message, fallbackReason)
		}
		if reason == "" {
			reason = fallbackReason
		}
		return Classification{Class: result.Error.Class, ReasonCode: reason}
	}
	if err != nil {
		return Classification{
			Class:      fallbackClass,
			ReasonCode: inferMissingRequiredReason(err.Error(), fallbackReason),
		}
	}
	return Classification{
		Class:      "",
		ReasonCode: fallbackReason,
	}
}

func ValidateMandatoryModelResponse(res types.ModelResponse) error {
	if strings.TrimSpace(res.FinalAnswer) == "" && len(res.ToolCalls) == 0 && res.ClarificationRequest == nil {
		return &ConformanceError{
			Class:      types.ErrModel,
			ReasonCode: "model.response.malformed",
			Message:    "missing mandatory model response terminal fields",
		}
	}
	return nil
}

func EvaluateOptionalTokenCount(supported bool) OptionalCapabilityResult {
	if supported {
		return OptionalCapabilityResult{
			Downgraded: false,
			ReasonCode: "model.capability.token_count_supported",
		}
	}
	return OptionalCapabilityResult{
		Downgraded: true,
		ReasonCode: "model.capability.token_count_unsupported_downgrade",
	}
}

func IsDeterministicClassification(a, b Classification) bool {
	return a.Class == b.Class && a.ReasonCode == b.ReasonCode
}

func MustRequireStringArg(args map[string]any, key string, class types.ErrorClass, reason string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return "", &ConformanceError{
			Class:      class,
			ReasonCode: reason,
			Message:    "missing required input",
		}
	}
	val, ok := raw.(string)
	if !ok || strings.TrimSpace(val) == "" {
		return "", &ConformanceError{
			Class:      class,
			ReasonCode: reason,
			Message:    "missing required input",
		}
	}
	return val, nil
}

func asConformanceError(err error) *ConformanceError {
	if err == nil {
		return nil
	}
	var ce *ConformanceError
	if errors.As(err, &ce) {
		return ce
	}
	return nil
}

func reasonFromDetails(details map[string]any) string {
	if details == nil {
		return ""
	}
	if raw, ok := details["reason_code"]; ok {
		if code, ok := raw.(string); ok {
			return strings.TrimSpace(code)
		}
	}
	return ""
}

func inferMissingRequiredReason(message, fallbackReason string) string {
	msg := strings.ToLower(message)
	if strings.Contains(msg, "missing required input") || strings.Contains(msg, "missing required") {
		parts := strings.Split(fallbackReason, ".")
		domain := parts[0]
		if domain == "" {
			domain = "adapter"
		}
		return domain + ".validation.missing_required_input"
	}
	return fallbackReason
}
