package manifest

import (
	"sort"
	"strings"
)

const (
	SandboxBackendLinuxNSJail = "linux_nsjail"
	SandboxBackendLinuxBwrap  = "linux_bwrap"
	SandboxBackendOCIRuntime  = "oci_runtime"
	SandboxBackendWindowsJob  = "windows_job"
)

const (
	SandboxSessionModePerCall    = "per_call"
	SandboxSessionModePerSession = "per_session"
)

type SandboxProfile struct {
	ID                    string
	Backend               string
	HostOS                string
	HostArch              string
	SessionModesSupported []string
}

var defaultSandboxProfilePack = []SandboxProfile{
	{
		ID:                    SandboxBackendLinuxNSJail,
		Backend:               SandboxBackendLinuxNSJail,
		HostOS:                "linux",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall, SandboxSessionModePerSession},
	},
	{
		ID:                    SandboxBackendLinuxBwrap,
		Backend:               SandboxBackendLinuxBwrap,
		HostOS:                "linux",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall, SandboxSessionModePerSession},
	},
	{
		ID:                    SandboxBackendOCIRuntime,
		Backend:               SandboxBackendOCIRuntime,
		HostOS:                "linux",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall, SandboxSessionModePerSession},
	},
	{
		ID:                    SandboxBackendWindowsJob,
		Backend:               SandboxBackendWindowsJob,
		HostOS:                "windows",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall, SandboxSessionModePerSession},
	},
}

func SandboxProfileByID(id string) (SandboxProfile, bool) {
	target := strings.ToLower(strings.TrimSpace(id))
	if target == "" {
		return SandboxProfile{}, false
	}
	for i := range defaultSandboxProfilePack {
		if strings.TrimSpace(defaultSandboxProfilePack[i].ID) == target {
			out := defaultSandboxProfilePack[i]
			out.SessionModesSupported = append([]string(nil), out.SessionModesSupported...)
			return out, true
		}
	}
	return SandboxProfile{}, false
}

func SupportedSandboxBackends() []string {
	out := make([]string, 0, len(defaultSandboxProfilePack))
	seen := map[string]struct{}{}
	for i := range defaultSandboxProfilePack {
		backend := strings.ToLower(strings.TrimSpace(defaultSandboxProfilePack[i].Backend))
		if backend == "" {
			continue
		}
		if _, ok := seen[backend]; ok {
			continue
		}
		seen[backend] = struct{}{}
		out = append(out, backend)
	}
	sort.Strings(out)
	return out
}

func IsSupportedSandboxBackend(backend string) bool {
	target := strings.ToLower(strings.TrimSpace(backend))
	if target == "" {
		return false
	}
	for i := range defaultSandboxProfilePack {
		if strings.TrimSpace(defaultSandboxProfilePack[i].Backend) == target {
			return true
		}
	}
	return false
}
