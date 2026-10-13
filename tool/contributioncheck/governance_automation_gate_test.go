package contributioncheck

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoadmapStatusConsistencyGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-openspec-roadmap-status-consistency.sh")
	psPath := filepath.Join(root, "scripts", "check-openspec-roadmap-status-consistency.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"roadmap-status-drift",
		"openspec list --json",
		"openspec/changes/archive/INDEX.md",
		"docs/development-roadmap.md",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell roadmap status script missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell roadmap status script missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell roadmap status script must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell roadmap status script must import strict helper")
	}
}

func TestExampleImpactDeclarationGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-openspec-example-impact-declaration.sh")
	psPath := filepath.Join(root, "scripts", "check-openspec-example-impact-declaration.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"missing-example-impact-declaration",
		"invalid-example-impact-value",
		"新增示例",
		"修改示例",
		"无需示例变更（附理由）",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell example impact script missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell example impact script missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell example impact script must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell example impact script must import strict helper")
	}
}

func TestPublicSemanticDocumentationCommitmentGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellRaw, err := os.ReadFile(filepath.Join(root, "scripts", "check-openspec-documentation-impact.sh"))
	if err != nil {
		t.Fatalf("read shell documentation impact script: %v", err)
	}
	psRaw, err := os.ReadFile(filepath.Join(root, "scripts", "check-openspec-documentation-impact.ps1"))
	if err != nil {
		t.Fatalf("read powershell documentation impact script: %v", err)
	}
	shell, ps := string(shellRaw), string(psRaw)
	for _, token := range []string{
		"missing-public-semantic-documentation-commitment",
		"invalid-public-semantic-documentation-commitment",
		"public-semantic-doc-sync-missing",
		"public-semantic-documentation-drift",
		"Public Semantic Documentation Commitment",
		"对外语义变化",
		"文档同步状态",
		"维护文档",
		"维护 owner",
		"验证证据",
	} {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell documentation impact script missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell documentation impact script missing token %q", token)
		}
	}
}

func TestPublicSemanticDocumentationCommitmentGateFixtures(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh executable not found")
	}
	root := repoRoot(t)
	runners := []struct {
		name string
		args func(string) []string
	}{
		{"powershell", func(dir string) []string {
			return []string{"pwsh", "-NoProfile", "-File", filepath.Join(dir, "scripts", "check-openspec-documentation-impact.ps1")}
		}},
	}
	if bash, ok := findBashExecutable(); ok {
		runners = append(runners, struct {
			name string
			args func(string) []string
		}{"shell", func(dir string) []string {
			return []string{bash, filepath.Join(dir, "scripts", "check-openspec-documentation-impact.sh")}
		}})
	}
	const commitment = "## Public Semantic Documentation Commitment\n\n- 对外语义变化：是\n- 文档同步状态：已同步\n- 维护文档：`README.md`\n- 维护 owner：maintainer\n- 验证证据：pwsh -File scripts/check-docs-consistency.ps1\n"
	const areas = "## Documentation Impact Assessment\n\n| Area | Outcome | Affected paths | Owner | Verification |\n| --- | --- | --- | --- | --- |\n| architecture | 无需文档变更（附理由） | — | owner | no architecture changes |\n| components | 无需文档变更（附理由） | — | owner | no component changes |\n| configuration | 无需文档变更（附理由） | — | owner | no configuration changes |\n| contract/API | 无需文档变更（附理由） | — | owner | no API changes |\n| diagnostics | 无需文档变更（附理由） | — | owner | no diagnostic changes |\n| examples | 无需文档变更（附理由） | — | owner | no example changes |\n| CLI/integration | 无需文档变更（附理由） | — | owner | no CLI changes |\n| best practices | 无需文档变更（附理由） | — | owner | no practice changes |\n| roadmap | 无需文档变更（附理由） | — | owner | no roadmap changes |\n"
	for _, tc := range []struct {
		name, proposal, expected string
	}{
		{"valid", commitment, ""},
		{"valid-committed", commitment, ""},
		{"valid-large-untracked", commitment, ""},
		{"unsynced-doc", commitment, "public-semantic-doc-sync-missing"},
		{"missing", "## Other Commitment\n", "missing-public-semantic-documentation-commitment"},
		{"invalid", strings.Replace(commitment, "对外语义变化：是", "对外语义变化：maybe", 1), "invalid-public-semantic-documentation-commitment"},
		{"missing-doc", strings.Replace(commitment, "维护文档：`README.md`", "维护文档：—", 1), "public-semantic-doc-sync-missing"},
		{"missing-doc-path", strings.Replace(commitment, "维护文档：`README.md`", "维护文档：`missing.md`", 1), "public-semantic-doc-sync-missing"},
		{"drift", strings.Replace(strings.Replace(commitment, "对外语义变化：是", "对外语义变化：否", 1), "文档同步状态：已同步", "文档同步状态：无需同步（附理由）：只改内部治理测试，不改变对外语义。", 1), "public-semantic-documentation-drift"},
		{"owner-drift", strings.Replace(commitment, "维护 owner：maintainer", "维护 owner：release owner", 1), "public-semantic-documentation-drift"},
		{"no-reason", strings.Replace(strings.Replace(commitment, "对外语义变化：是", "对外语义变化：否", 1), "文档同步状态：已同步", "文档同步状态：无需同步（附理由）：", 1), "invalid-public-semantic-documentation-commitment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			init := exec.Command("git", "init", "--quiet", "-b", "master")
			init.Dir = dir
			if out, err := init.CombinedOutput(); err != nil {
				t.Fatalf("init fixture git repo: %v: %s", err, out)
			}
			for _, subdir := range []string{"scripts", filepath.Join("openspec", "changes", "fixture")} {
				if err := os.MkdirAll(filepath.Join(dir, subdir), 0755); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"check-openspec-documentation-impact.ps1", "check-openspec-documentation-impact.sh"} {
				raw, err := os.ReadFile(filepath.Join(root, "scripts", name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "scripts", name), raw, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Fixture\n"), 0644); err != nil {
				t.Fatal(err)
			}
			for name, section := range map[string]string{"proposal.md": tc.proposal, "design.md": commitment, "tasks.md": commitment} {
				if err := os.WriteFile(filepath.Join(dir, "openspec", "changes", "fixture", name), []byte(section+"\n"+areas), 0644); err != nil {
					t.Fatal(err)
				}
			}
			config := exec.Command("git", "config", "user.email", "fixture@example.invalid")
			config.Dir = dir
			if out, err := config.CombinedOutput(); err != nil {
				t.Fatalf("configure fixture git email: %v: %s", err, out)
			}
			config = exec.Command("git", "config", "user.name", "Fixture")
			config.Dir = dir
			if out, err := config.CombinedOutput(); err != nil {
				t.Fatalf("configure fixture git name: %v: %s", err, out)
			}
			commit := exec.Command("git", "add", ".")
			commit.Dir = dir
			if out, err := commit.CombinedOutput(); err != nil {
				t.Fatalf("stage fixture baseline: %v: %s", err, out)
			}
			commit = exec.Command("git", "commit", "--quiet", "-m", "fixture baseline")
			commit.Dir = dir
			if out, err := commit.CombinedOutput(); err != nil {
				t.Fatalf("commit fixture baseline: %v: %s", err, out)
			}
			if tc.name == "valid" || tc.name == "valid-committed" || tc.name == "valid-large-untracked" {
				if tc.name == "valid-committed" {
					branch := exec.Command("git", "switch", "--quiet", "-c", "fixture-change")
					branch.Dir = dir
					if out, err := branch.CombinedOutput(); err != nil {
						t.Fatalf("create fixture change branch: %v: %s", err, out)
					}
				}
				if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Fixture\n\nUpdated by this change.\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if tc.name == "valid-committed" {
					commit = exec.Command("git", "add", "README.md")
					commit.Dir = dir
					if out, err := commit.CombinedOutput(); err != nil {
						t.Fatalf("stage updated fixture document: %v: %s", err, out)
					}
					commit = exec.Command("git", "commit", "--quiet", "-m", "update fixture documentation")
					commit.Dir = dir
					if out, err := commit.CombinedOutput(); err != nil {
						t.Fatalf("commit updated fixture document: %v: %s", err, out)
					}
				}
			}
			if tc.name == "valid-large-untracked" {
				cache := filepath.Join(dir, "zz-cache")
				if err := os.MkdirAll(cache, 0755); err != nil {
					t.Fatal(err)
				}
				// Enough bytes after README to reproduce grep -q SIGPIPE under pipefail.
				for i := 0; i < 4000; i++ {
					if err := os.WriteFile(filepath.Join(cache, fmt.Sprintf("%05d-%s", i, strings.Repeat("x", 100))), nil, 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, runner := range runners {
				t.Run(runner.name, func(t *testing.T) {
					args := runner.args(dir)
					cmd := exec.Command(args[0], args[1:]...)
					cmd.Dir = dir
					out, err := cmd.CombinedOutput()
					if isBashInfrastructureFailure(string(out)) && runner.name == "shell" {
						t.Skipf("bash infrastructure failure: %s", out)
					}
					if tc.expected == "" && err != nil {
						t.Fatalf("expected success: %v: %s", err, out)
					}
					if tc.expected != "" && (err == nil || !strings.Contains(string(out), tc.expected)) {
						t.Fatalf("expected %s: %v: %s", tc.expected, err, out)
					}
				})
			}
		})
	}
}

func TestDocsConsistencyIncludesRoadmapStatusConsistencyGate(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-docs-consistency.sh")
	psPath := filepath.Join(root, "scripts", "check-docs-consistency.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell docs consistency script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell docs consistency script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	if !strings.Contains(shell, "check-openspec-roadmap-status-consistency.sh") {
		t.Fatalf("shell docs consistency must invoke roadmap status consistency script")
	}
	if !strings.Contains(shell, "[docs-consistency][openspec-roadmap-status-consistency]") {
		t.Fatalf("shell docs consistency must expose roadmap status consistency failure label")
	}
	if !strings.Contains(ps, "check-openspec-roadmap-status-consistency.ps1") {
		t.Fatalf("powershell docs consistency must invoke roadmap status consistency script")
	}
	if !strings.Contains(ps, "[docs-consistency] openspec roadmap status consistency") {
		t.Fatalf("powershell docs consistency must expose roadmap status consistency step label")
	}
}

func TestQualityGateIncludesExampleImpactDeclarationGate(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-quality-gate.sh")
	psPath := filepath.Join(root, "scripts", "check-quality-gate.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell quality gate script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell quality gate script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	if !strings.Contains(shell, "check-openspec-example-impact-declaration.sh") {
		t.Fatalf("shell quality gate must invoke example impact declaration script")
	}
	if !strings.Contains(shell, "[quality-gate][openspec-example-impact-declaration]") {
		t.Fatalf("shell quality gate must expose example impact declaration failure label")
	}
	if !strings.Contains(ps, "check-openspec-example-impact-declaration.ps1") {
		t.Fatalf("powershell quality gate must invoke example impact declaration script")
	}
	if !strings.Contains(ps, "[quality-gate] openspec example impact declaration") {
		t.Fatalf("powershell quality gate must expose example impact declaration step label")
	}
}

func TestCIGovernanceRequiredCheckCandidates(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)

	requiredTokens := []string{
		"openspec-roadmap-status-consistency-gate:",
		"run: bash scripts/check-openspec-roadmap-status-consistency.sh",
		"openspec-example-impact-declaration-gate:",
		"run: bash scripts/check-openspec-example-impact-declaration.sh",
		"if: github.event_name == 'pull_request'",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(ci, token) {
			t.Fatalf("ci workflow missing token %q for governance required-check candidates", token)
		}
	}
}
