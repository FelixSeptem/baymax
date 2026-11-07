package contributioncheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type ownerEvidenceIndex struct {
	Version   string                  `json:"version"`
	Semantics []ownerEvidenceSemantic `json:"semantics"`
}

type ownerEvidenceSemantic struct {
	ID               string              `json:"id"`
	Owner            string              `json:"owner"`
	MainSpec         string              `json:"main_spec"`
	Tests            []string            `json:"tests"`
	Replay           ownerEvidenceReplay `json:"replay"`
	Gate             string              `json:"gate"`
	Documentation    []string            `json:"documentation"`
	RollbackBoundary string              `json:"rollback_boundary"`
	NonGoals         string              `json:"non_goals"`
}

type ownerEvidenceReplay struct {
	Mode    string `json:"mode"`
	Fixture string `json:"fixture,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

var evidenceIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var evidenceOwnerPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(?:/[a-z][a-z0-9-]*)+$`)

// ValidateOwnerEvidenceIndex checks repository evidence without executing it.
func ValidateOwnerEvidenceIndex(root string, raw []byte, sourceMap, qualityPS, qualitySH string) []string {
	var index ownerEvidenceIndex
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&index); err != nil {
		return []string{"invalid index: " + err.Error()}
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return []string{"invalid trailing JSON"}
	}
	issues := []string{}
	add := func(id, problem string) { issues = append(issues, fmt.Sprintf("%s: %s", id, problem)) }
	if index.Version != "owner-evidence-index.v1" {
		add("index", "unsupported version")
	}
	if len(index.Semantics) == 0 {
		add("index", "no semantics")
	}
	seen := make(map[string]bool)
	for _, entry := range index.Semantics {
		id := entry.ID
		if !evidenceIDPattern.MatchString(id) || seen[id] {
			add(id, "invalid or duplicate id")
		}
		seen[id] = true
		if !evidenceOwnerPattern.MatchString(entry.Owner) {
			add(id, "invalid owner")
		}
		matches := 0
		for _, line := range strings.Split(sourceMap, "\n") {
			cells := strings.Split(strings.TrimSpace(line), "|")
			if len(cells) < 7 || strings.TrimSpace(cells[1]) != id {
				continue
			}
			matches++
			if strings.TrimSpace(cells[4]) != "`"+entry.Owner+"`" {
				add(id, "documentation owner mismatch")
			}
		}
		if matches != 1 {
			add(id, "documentation owner row missing or duplicated")
		}
		checkPath := func(p, label string) {
			if p == "" || strings.Contains(p, "\\") || path.IsAbs(p) || path.Clean(p) != p || p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, ":") {
				add(id, "invalid "+label+" path")
				return
			}
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
			if err != nil || !info.Mode().IsRegular() {
				add(id, "missing "+label+": "+p)
			}
		}
		checkPath(entry.MainSpec, "spec")
		if len(entry.Tests) == 0 {
			add(id, "missing tests")
		}
		for _, p := range entry.Tests {
			checkPath(p, "test")
		}
		if len(entry.Documentation) == 0 {
			add(id, "missing documentation")
		}
		for _, p := range entry.Documentation {
			checkPath(p, "documentation")
		}
		switch entry.Replay.Mode {
		case "fixture":
			checkPath(entry.Replay.Fixture, "fixture")
			if entry.Replay.Reason != "" {
				add(id, "fixture replay has exemption reason")
			}
		case "not_applicable":
			if len(strings.TrimSpace(entry.Replay.Reason)) < 16 || entry.Replay.Fixture != "" {
				add(id, "replay exemption requires a concrete reason and no fixture")
			}
		default:
			add(id, "invalid replay mode")
		}
		if strings.TrimSpace(entry.RollbackBoundary) == "" || strings.TrimSpace(entry.NonGoals) == "" {
			add(id, "missing rollback or non-goal boundary")
		}
		checkPath(entry.Gate+".ps1", "PowerShell gate")
		checkPath(entry.Gate+".sh", "shell gate")
		if entry.Gate == "" || !regexp.MustCompile(`(?m)^\s*(?:Command\s*=\s*\{\s*)?pwsh\s+-File\s+`+regexp.QuoteMeta(entry.Gate)+`\.ps1\s*(?:\}|$)`).MatchString(qualityPS) {
			add(id, "PowerShell gate not wired")
		}
		if entry.Gate == "" || !regexp.MustCompile(`(?m)^\s*(?:if\s+!\s+)?bash\s+`+regexp.QuoteMeta(entry.Gate)+`\.sh(?:\s*;\s*then)?\s*$`).MatchString(qualitySH) {
			add(id, "shell gate not wired")
		}
	}
	return issues
}
