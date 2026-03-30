package manifest

import (
	"reflect"
	"testing"
)

func TestSandboxProfilePackContainsMainstreamBackends(t *testing.T) {
	want := []string{
		SandboxBackendLinuxBwrap,
		SandboxBackendLinuxNSJail,
		SandboxBackendOCIRuntime,
		SandboxBackendWindowsJob,
	}
	got := SupportedSandboxBackends()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("supported backends mismatch: got=%#v want=%#v", got, want)
	}
}

func TestSandboxProfileByID(t *testing.T) {
	profile, ok := SandboxProfileByID(SandboxBackendWindowsJob)
	if !ok {
		t.Fatal("expected windows_job profile to exist")
	}
	if profile.Backend != SandboxBackendWindowsJob ||
		profile.HostOS != "windows" ||
		profile.HostArch != "amd64" {
		t.Fatalf("unexpected profile payload: %#v", profile)
	}
	if len(profile.SessionModesSupported) != 2 ||
		profile.SessionModesSupported[0] != SandboxSessionModePerCall ||
		profile.SessionModesSupported[1] != SandboxSessionModePerSession {
		t.Fatalf("unexpected session mode payload: %#v", profile.SessionModesSupported)
	}

	if _, ok := SandboxProfileByID("missing-profile"); ok {
		t.Fatal("missing profile should not be resolved")
	}
}
