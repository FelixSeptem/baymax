package diagnosticsreplay

import "github.com/FelixSeptem/baymax/model/conformance"

const OpenAIEndpointProfileFixtureV1 = conformance.FixtureVersionOpenAIEndpointProfileV1

const (
	OpenAIEndpointProfileVerdictConformant = conformance.OpenAIEndpointProfileVerdictConformant
	OpenAIEndpointProfileVerdictBlocked    = conformance.OpenAIEndpointProfileVerdictBlocked
)

func ReplayOpenAIEndpointProfileFixtureJSON(raw []byte, projectionDigests map[string]string) (conformance.OpenAIEndpointProfileReplayResult, error) {
	fixture, err := conformance.ParseOpenAIEndpointProfileFixtureJSON(raw)
	if err != nil {
		return conformance.OpenAIEndpointProfileReplayResult{}, err
	}
	return conformance.ReplayOpenAIEndpointProfileFixture(fixture, projectionDigests)
}
