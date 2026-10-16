package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostResumeAdmissionGateAndOwnership(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"core/types/host_resume_admission.go", "tool/diagnosticsreplay/host_resume_admission.go"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"github.com/FelixSeptem/baymax/observability", "github.com/FelixSeptem/baymax/mcp/", "go func", ".Resume(", ".Record("} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("%s owns forbidden %q", name, forbidden)
			}
		}
	}
	for _, extension := range []string{"sh", "ps1"} {
		name := "scripts/check-host-resume-admission-contract." + extension
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{"TestHostResume", "testdata/host_resume_admission.invalid.json", "examples/host-resume-admission", "host_resume.replay_drift", "resume_in_progress", "run_stream_parity"} {
			if !strings.Contains(string(raw), required) {
				t.Fatalf("%s missing %q", name, required)
			}
		}
		quality, err := os.ReadFile(filepath.Join(root, "scripts/check-quality-gate."+extension))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(quality), name) {
			t.Fatalf("%s not wired into quality gate", name)
		}
	}
}
