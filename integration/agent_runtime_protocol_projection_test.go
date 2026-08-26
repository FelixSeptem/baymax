package integration

import (
	"testing"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestAgentRuntimeProtocolSourceDescriptorsAreOptInAndOwned(t *testing.T) {
	constructors := []struct {
		name string
		make func() (types.ProtocolDescriptor, error)
	}{
		{"runner", func() (types.ProtocolDescriptor, error) {
			return runner.ProtocolDescriptorForRuntime("runner-1", "v1", nil, nil)
		}},
		{"realtime", func() (types.ProtocolDescriptor, error) {
			return runner.RealtimeProtocolDescriptorForRuntime("realtime-1", "v1", nil, nil)
		}},
		{"workflow", func() (types.ProtocolDescriptor, error) {
			return workflow.ProtocolDescriptorForRuntime("workflow-1", "v1", nil, nil)
		}},
		{"composer", func() (types.ProtocolDescriptor, error) {
			return composer.ProtocolDescriptorForRuntime("composer-1", "v1", nil, nil)
		}},
		{"teams", func() (types.ProtocolDescriptor, error) {
			return teams.ProtocolDescriptorForRuntime("teams-1", "v1", nil, nil)
		}},
		{"scheduler", func() (types.ProtocolDescriptor, error) {
			return scheduler.ProtocolDescriptorForRuntime("scheduler-1", "v1", nil, nil)
		}},
		{"a2a", func() (types.ProtocolDescriptor, error) {
			return a2a.ProtocolDescriptorForRuntime("a2a-1", "v1", nil, nil)
		}},
	}
	for _, tc := range constructors {
		t.Run(tc.name, func(t *testing.T) {
			descriptor, err := tc.make()
			if err != nil {
				t.Fatalf("descriptor: %v", err)
			}
			if descriptor.Source == "" || descriptor.RuntimeID == "" || descriptor.ProfileVersion != "v1" {
				t.Fatalf("descriptor=%#v", descriptor)
			}
			if tc.name == "composer" && descriptor.Source != types.ProtocolSourceComposer {
				t.Fatalf("composer descriptor source=%q, want %q", descriptor.Source, types.ProtocolSourceComposer)
			}
			if err := descriptor.Validate(); err != nil {
				t.Fatalf("descriptor validation: %v", err)
			}
		})
	}
}
