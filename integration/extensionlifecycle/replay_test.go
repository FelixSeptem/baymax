package extensionlifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/extension"
)

func TestReplayContractExtensionLifecycleFixtures(t *testing.T) {
	root := filepath.Join("..", "testdata", "diagnostics-replay", "extension-lifecycle-governance", "v1")
	if _, err := os.Stat(filepath.Join(root, "success.json")); err != nil {
		t.Fatal(err)
	}
	d := extension.Descriptor{Name: "lint", Kind: extension.KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:abc", Source: extension.SourceProjectExplicit}
	decision, err := extension.Admit(d, extension.AdmissionInput{RuntimeVersion: "0.2.0"})
	if err != nil || decision.Outcome != extension.AdmissionAllow || !decision.Activated {
		t.Fatalf("success replay decision=%#v err=%v", decision, err)
	}
	bad := d
	bad.Digest = ""
	decision, err = extension.Admit(bad, extension.AdmissionInput{RuntimeVersion: "0.2.0"})
	if err != nil || decision.Outcome != extension.AdmissionDeny || decision.Activated {
		t.Fatalf("invalid replay decision=%#v err=%v", decision, err)
	}
}
