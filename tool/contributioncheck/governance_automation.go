package contributioncheck

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	GovernanceCodeRoadmapStatusDrift              = "roadmap-status-drift"
	GovernanceCodeMissingExampleImpactDeclaration = "missing-example-impact-declaration"
	GovernanceCodeInvalidExampleImpactValue       = "invalid-example-impact-value"

	ExampleImpactValueAddExample          = "新增示例"
	ExampleImpactValueModifyExample       = "修改示例"
	ExampleImpactValueNoExampleWithReason = "无需示例变更（附理由）"

	MinExampleImpactDeclarationChangeID = 70
)

var (
	roadmapChangeSlugPattern    = regexp.MustCompile(`[a-z0-9]+(?:-[a-z0-9]+)+`)
	changeNumericIDPattern      = regexp.MustCompile(`-a([0-9]+)(?:-|$)`)
	exampleImpactHeadingPattern = regexp.MustCompile(`(?i)^##\s*(example impact assessment|示例影响评估)\s*$`)
)

var allowedExampleImpactValues = []string{
	ExampleImpactValueAddExample,
	ExampleImpactValueModifyExample,
	ExampleImpactValueNoExampleWithReason,
}

type GovernanceAutomationIssue struct {
	Code    string
	Message string
}

func ValidateRoadmapStatusConsistency(activeChanges, archivedChanges []string, roadmap string) []GovernanceAutomationIssue {
	issues := make([]GovernanceAutomationIssue, 0)

	activeSet := toSet(activeChanges)
	archivedSet := toSet(archivedChanges)

	roadmapInProgress, roadmapArchived := parseRoadmapStatusProjection(roadmap)
	roadmapInProgressSet := toSet(roadmapInProgress)
	roadmapArchivedSet := toSet(roadmapArchived)

	for _, change := range sortedUniqueSlice(activeChanges) {
		if _, ok := roadmapInProgressSet[change]; ok {
			continue
		}
		issues = append(issues, GovernanceAutomationIssue{
			Code:    GovernanceCodeRoadmapStatusDrift,
			Message: "roadmap missing in-progress change from openspec list: " + change,
		})
	}

	for _, change := range sortedKeys(roadmapInProgressSet) {
		if _, ok := activeSet[change]; ok {
			continue
		}
		if _, archived := archivedSet[change]; archived {
			issues = append(issues, GovernanceAutomationIssue{
				Code:    GovernanceCodeRoadmapStatusDrift,
				Message: "roadmap marks archived change as in-progress: " + change,
			})
			continue
		}
		issues = append(issues, GovernanceAutomationIssue{
			Code:    GovernanceCodeRoadmapStatusDrift,
			Message: "roadmap in-progress entry is not active in openspec list: " + change,
		})
	}

	for _, change := range sortedKeys(roadmapArchivedSet) {
		if _, ok := archivedSet[change]; ok {
			continue
		}
		if _, active := activeSet[change]; active {
			issues = append(issues, GovernanceAutomationIssue{
				Code:    GovernanceCodeRoadmapStatusDrift,
				Message: "roadmap marks active change as archived: " + change,
			})
			continue
		}
		issues = append(issues, GovernanceAutomationIssue{
			Code:    GovernanceCodeRoadmapStatusDrift,
			Message: "roadmap archived entry is not present in archive index: " + change,
		})
	}

	return issues
}

func ValidateProposalExampleImpactDeclarations(changeProposals map[string]string) []GovernanceAutomationIssue {
	issues := make([]GovernanceAutomationIssue, 0, len(changeProposals))
	changeNames := make([]string, 0, len(changeProposals))
	for changeName := range changeProposals {
		changeNames = append(changeNames, changeName)
	}
	sort.Strings(changeNames)
	for _, changeName := range changeNames {
		issues = append(issues, ValidateProposalExampleImpactDeclaration(changeName, changeProposals[changeName])...)
	}
	return issues
}

func ValidateProposalExampleImpactDeclaration(changeName, proposal string) []GovernanceAutomationIssue {
	if !shouldValidateExampleImpactDeclaration(changeName) {
		return nil
	}

	value, found := extractExampleImpactDeclarationValue(proposal)
	if !found {
		return []GovernanceAutomationIssue{{
			Code:    GovernanceCodeMissingExampleImpactDeclaration,
			Message: fmt.Sprintf("%s missing Example Impact Assessment declaration", changeName),
		}}
	}
	if isAllowedExampleImpactValue(value) {
		return nil
	}
	return []GovernanceAutomationIssue{{
		Code: GovernanceCodeInvalidExampleImpactValue,
		Message: fmt.Sprintf(
			"%s uses unsupported Example Impact Assessment value %q (allowed: %s)",
			changeName,
			value,
			strings.Join(allowedExampleImpactValues, ", "),
		),
	}}
}

func parseRoadmapStatusProjection(roadmap string) ([]string, []string) {
	section := extractRoadmapCurrentStatusSection(roadmap)

	inProgress := make([]string, 0)
	archived := make([]string, 0)
	mode := ""

	for _, raw := range strings.Split(section, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "- 进行中："):
			mode = "in-progress"
			continue
		case strings.HasPrefix(line, "- 已归档："):
			mode = "archived"
			continue
		case strings.HasPrefix(line, "- 候选："):
			mode = "candidate"
			continue
		}

		if mode != "in-progress" && mode != "archived" {
			continue
		}
		if !strings.HasPrefix(line, "-") {
			continue
		}

		changeSlug := extractRoadmapChangeSlug(line)
		if changeSlug == "" {
			continue
		}

		if mode == "in-progress" {
			inProgress = append(inProgress, changeSlug)
			continue
		}
		archived = append(archived, changeSlug)
	}

	return sortedUniqueSlice(inProgress), sortedUniqueSlice(archived)
}

func extractRoadmapCurrentStatusSection(roadmap string) string {
	lines := strings.Split(roadmap, "\n")
	start := -1
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## 当前状态") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return roadmap
	}

	end := len(lines)
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func extractRoadmapChangeSlug(line string) string {
	trimmed := strings.TrimSpace(line)
	if firstTick := strings.Index(trimmed, "`"); firstTick >= 0 {
		rest := trimmed[firstTick+1:]
		if secondTick := strings.Index(rest, "`"); secondTick >= 0 {
			candidate := strings.TrimSpace(rest[:secondTick])
			if roadmapChangeSlugPattern.MatchString(candidate) {
				return candidate
			}
		}
	}

	matches := roadmapChangeSlugPattern.FindAllString(trimmed, -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.TrimSpace(matches[0])
}

func extractExampleImpactDeclarationValue(proposal string) (string, bool) {
	lines := strings.Split(proposal, "\n")
	inSection := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if !inSection {
			if exampleImpactHeadingPattern.MatchString(line) {
				inSection = true
			}
			continue
		}

		if strings.HasPrefix(line, "## ") {
			break
		}
		if line == "" {
			continue
		}

		candidate := normalizeExampleImpactCandidate(line)
		if candidate == "" {
			continue
		}

		if strings.HasPrefix(candidate, ExampleImpactValueNoExampleWithReason) {
			suffix := strings.TrimSpace(strings.TrimPrefix(candidate, ExampleImpactValueNoExampleWithReason))
			if suffix == "" {
				return ExampleImpactValueNoExampleWithReason, true
			}
			if strings.HasPrefix(suffix, "：") || strings.HasPrefix(suffix, ":") {
				reason := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(suffix, "："), ":"))
				if reason != "" {
					return ExampleImpactValueNoExampleWithReason, true
				}
			}
		}

		return candidate, true
	}

	return "", false
}

func normalizeExampleImpactCandidate(line string) string {
	candidate := strings.TrimSpace(line)
	candidate = strings.TrimLeft(candidate, "-*")
	candidate = strings.TrimSpace(candidate)
	if strings.HasPrefix(candidate, "[x]") || strings.HasPrefix(candidate, "[X]") || strings.HasPrefix(candidate, "[ ]") {
		candidate = strings.TrimSpace(candidate[3:])
	}
	candidate = strings.Trim(candidate, "`")
	return strings.TrimSpace(candidate)
}

func isAllowedExampleImpactValue(value string) bool {
	for _, allowed := range allowedExampleImpactValues {
		if value == allowed {
			return true
		}
	}
	return false
}

func shouldValidateExampleImpactDeclaration(changeName string) bool {
	id, ok := extractChangeNumericID(changeName)
	if !ok {
		return true
	}
	return id >= MinExampleImpactDeclarationChangeID
}

func extractChangeNumericID(changeName string) (int, bool) {
	matches := changeNumericIDPattern.FindAllStringSubmatch(strings.ToLower(strings.TrimSpace(changeName)), -1)
	if len(matches) == 0 {
		return 0, false
	}
	raw := matches[len(matches)-1][1]
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return id, true
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	return set
}

func sortedUniqueSlice(values []string) []string {
	set := toSet(values)
	return sortedKeys(set)
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
