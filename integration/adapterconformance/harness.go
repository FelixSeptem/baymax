package adapterconformance

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

type Category string

const (
	CategoryMCP   Category = "mcp"
	CategoryModel Category = "model"
	CategoryTool  Category = "tool"
)

type Scenario struct {
	ID           string
	Category     Category
	PriorityTier int
	TemplatePath string
	TraceComment string
}

var MinimumMatrix = []Scenario{
	{
		ID:           "mcp-normalization-fail-fast",
		Category:     CategoryMCP,
		PriorityTier: 1,
		TemplatePath: "examples/templates/mcp-adapter-template/main.go",
		TraceComment: "A21 template linkage: MCP adapter template baseline and fail-fast boundary",
	},
	{
		ID:           "model-run-stream-downgrade",
		Category:     CategoryModel,
		PriorityTier: 2,
		TemplatePath: "examples/templates/model-adapter-template/main.go",
		TraceComment: "A21 template linkage: Model adapter template run/stream equivalence and downgrade semantics",
	},
	{
		ID:           "tool-invoke-fail-fast",
		Category:     CategoryTool,
		PriorityTier: 3,
		TemplatePath: "examples/templates/tool-adapter-template/main.go",
		TraceComment: "A21 template linkage: Tool adapter template invocation and mandatory-input fail-fast",
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
