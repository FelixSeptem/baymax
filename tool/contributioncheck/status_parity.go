package contributioncheck

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type StatusParityIssue struct {
	Code    string
	Message string
}

var changeIDPattern = regexp.MustCompile(`-a([0-9]+)$`)

func LoadOpenSpecStatusAuthority(repoRoot string) ([]string, []string, error) {
	cmd := exec.Command("openspec", "list", "--json")
	cmd.Dir = repoRoot
	raw, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("openspec list --json: %w", err)
	}
	active, err := parseActiveChanges(raw)
	if err != nil {
		return nil, nil, err
	}

	archivePath := filepath.Join(repoRoot, "openspec", "changes", "archive", "INDEX.md")
	archiveRaw, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read archive index: %w", err)
	}
	return active, parseArchivedChanges(string(archiveRaw)), nil
}

func ValidateStatusParity(activeChanges, archivedChanges []string, roadmap, readme string) []StatusParityIssue {
	issues := make([]StatusParityIssue, 0)

	activeSet := make(map[string]struct{}, len(activeChanges))
	for _, name := range activeChanges {
		activeSet[name] = struct{}{}
	}
	archivedSet := make(map[string]struct{}, len(archivedChanges))
	for _, name := range archivedChanges {
		archivedSet[name] = struct{}{}
	}

	roadmapInProgress := parseRoadmapInProgressChanges(roadmap)
	for _, change := range activeChanges {
		if _, ok := roadmapInProgress[change]; !ok {
			issues = append(issues, StatusParityIssue{
				Code:    "status-parity.snapshot-missing-active-change",
				Message: "roadmap missing active change in progress section: " + change,
			})
		}
	}
	for change := range roadmapInProgress {
		if _, ok := activeSet[change]; ok {
			continue
		}
		code := "status-parity.roadmap-inprogress-not-active"
		if _, archived := archivedSet[change]; archived {
			code = "status-parity.active-vs-archived-mismatch"
		}
		issues = append(issues, StatusParityIssue{
			Code:    code,
			Message: "roadmap in-progress entry conflicts with openspec authority: " + change,
		})
	}

	readmeStatuses := parseReadmeMilestoneStatus(readme)
	for _, change := range activeChanges {
		changeID := extractChangeID(change)
		if changeID == "" {
			continue
		}
		status, ok := readmeStatuses[changeID]
		if !ok {
			issues = append(issues, StatusParityIssue{
				Code:    "status-parity.snapshot-missing-active-milestone",
				Message: "README milestone snapshot missing active change id: " + changeID,
			})
			continue
		}
		if status != "进行中" {
			issues = append(issues, StatusParityIssue{
				Code:    "status-parity.stale-snapshot-mismatch",
				Message: "README milestone status must be 进行中 for active change " + changeID + ", got " + status,
			})
		}
	}

	for _, change := range archivedChanges {
		changeID := extractChangeID(change)
		if changeID == "" {
			continue
		}
		status, ok := readmeStatuses[changeID]
		if !ok {
			continue
		}
		if status == "进行中" {
			issues = append(issues, StatusParityIssue{
				Code:    "status-parity.active-vs-archived-mismatch",
				Message: "README marks archived change as in-progress: " + changeID,
			})
		}
	}

	return issues
}

func parseActiveChanges(raw []byte) ([]string, error) {
	var payload struct {
		Changes []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode openspec list payload: %w", err)
	}
	active := make([]string, 0, len(payload.Changes))
	for _, item := range payload.Changes {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.Status)) != "in-progress" {
			continue
		}
		active = append(active, strings.TrimSpace(item.Name))
	}
	return active, nil
}

func parseArchivedChanges(indexDoc string) []string {
	changes := make([]string, 0)
	for _, line := range strings.Split(indexDoc, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		parts := strings.Split(trimmed, "->")
		if len(parts) != 2 {
			continue
		}
		slug := strings.TrimSpace(parts[1])
		if slug == "" {
			continue
		}
		changes = append(changes, slug)
	}
	return changes
}

func parseRoadmapInProgressChanges(roadmap string) map[string]struct{} {
	results := map[string]struct{}{}
	lines := strings.Split(roadmap, "\n")
	inProgress := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "- 进行中：") {
			inProgress = true
			continue
		}
		if inProgress && strings.HasPrefix(line, "## ") {
			break
		}
		if !inProgress {
			continue
		}
		if !strings.HasPrefix(line, "-") {
			if line == "" {
				continue
			}
			continue
		}
		if strings.Count(line, "`") < 2 {
			continue
		}
		start := strings.Index(line, "`")
		end := strings.LastIndex(line, "`")
		if start < 0 || end <= start {
			continue
		}
		name := strings.TrimSpace(line[start+1 : end])
		if name == "" {
			continue
		}
		results[name] = struct{}{}
	}
	return results
}

func parseReadmeMilestoneStatus(readme string) map[string]string {
	status := map[string]string{}
	for _, raw := range strings.Split(readme, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "- A") {
			continue
		}
		id := ""
		if idx := strings.Index(line, "（"); idx > 0 {
			id = strings.TrimSpace(strings.TrimPrefix(line[:idx], "- "))
		}
		if id == "" {
			continue
		}
		switch {
		case strings.Contains(line, "进行中"):
			status[id] = "进行中"
		case strings.Contains(line, "已归档并稳定"):
			status[id] = "已归档并稳定"
		}
	}
	return status
}

func extractChangeID(changeName string) string {
	match := changeIDPattern.FindStringSubmatch(strings.TrimSpace(changeName))
	if len(match) != 2 {
		return ""
	}
	return "A" + match[1]
}
