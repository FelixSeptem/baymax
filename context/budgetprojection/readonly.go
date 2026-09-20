package budgetprojection

import (
	"reflect"
	"sort"
	"strings"
)

// projectionFieldAllowlist is the exhaustive set of fields a derived projection
// may carry. Adding any field outside this set (for example a write-back handle,
// an owner reference or an applied/applied-to marker) is rejected.
var projectionFieldAllowlist = map[string]string{
	"Version":   "string",
	"Complete":  "bool",
	"Remaining": "budgetprojection.BudgetRemainingProjection",
	"Pressure":  "budgetprojection.BudgetPressureLevel",
	"Notes":     "[]string",
	"Digest":    "string",
}

var projectionMethodDenylistPrefixes = []string{
	"Set",
	"Apply",
	"Write",
	"Commit",
	"Update",
	"Mutate",
	"Persist",
	"Store",
	"Save",
}

// DetectWritebackShape inspects the projection contract for shapes that would
// turn a read-only derived view into a writable one: unexpected fields,
// unexpected mutating methods, or a facts type that carries owner handles.
func DetectWritebackShape() []string {
	codes := make([]string, 0, 2)
	projectionType := reflect.TypeOf(BudgetProjection{})
	if projectionType.Kind() != reflect.Struct {
		return []string{CodeWritebackShapeDetected}
	}
	if projectionType.NumField() != len(projectionFieldAllowlist) {
		codes = append(codes, CodeWritebackShapeDetected)
	}
	for index := 0; index < projectionType.NumField(); index++ {
		field := projectionType.Field(index)
		expected, ok := projectionFieldAllowlist[field.Name]
		if !ok || field.Type.String() != expected {
			codes = append(codes, CodeWritebackShapeDetected)
		}
	}
	if !isReadOnlyMethodSet(projectionType) || !isReadOnlyMethodSet(reflect.TypeOf(&BudgetProjection{})) {
		codes = append(codes, CodeWritebackShapeDetected)
	}
	if !isReadOnlyFactsType(reflect.TypeOf(BudgetFacts{})) {
		codes = append(codes, CodeWritebackShapeDetected)
	}
	sort.Strings(codes)
	return dedupeStable(codes)
}

func isReadOnlyMethodSet(target reflect.Type) bool {
	if target == nil {
		return true
	}
	for index := 0; index < target.NumMethod(); index++ {
		name := target.Method(index).Name
		for _, prefix := range projectionMethodDenylistPrefixes {
			if strings.HasPrefix(name, prefix) {
				return false
			}
		}
	}
	return true
}

// factsFieldAllowlist enumerates the snapshot fields BudgetFacts may carry.
var factsFieldAllowlist = map[string]struct{}{
	"Iteration": {},
	"ToolCall":  {},
	"Time":      {},
	"Cost":      {},
	"Admission": {},
	"Parent":    {},
	"Plan":      {},
}

func isReadOnlyFactsType(target reflect.Type) bool {
	if target.Kind() != reflect.Struct {
		return false
	}
	if target.NumField() != len(factsFieldAllowlist) {
		return false
	}
	for index := 0; index < target.NumField(); index++ {
		if _, ok := factsFieldAllowlist[target.Field(index).Name]; !ok {
			return false
		}
	}
	for index := 0; index < target.NumMethod(); index++ {
		name := target.Method(index).Name
		for _, prefix := range projectionMethodDenylistPrefixes {
			if strings.HasPrefix(name, prefix) {
				return false
			}
		}
	}
	return true
}
