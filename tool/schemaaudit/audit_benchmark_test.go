package schemaaudit

import "testing"

func BenchmarkAuditOfflineDeterministic(b *testing.B) {
	facts, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}})
	if err != nil {
		b.Fatal(err)
	}
	tools := make([]AdmittedTool, 16)
	for i := range tools {
		tools[i] = AdmittedTool{Identity: "tool-" + string(rune('a'+i)), Source: "local", Capabilities: []string{"search"}, Priority: i, Admitted: true, Schema: facts}
	}
	in := Input{Version: AuditVersion, SnapshotID: "bench", Tools: tools, Policy: Policy{SchemaBytesBudget: facts.Bytes * 8, TokenBudget: facts.TokenEstimate * 8}, Cases: []TaskCase{{ID: "search", Expected: []string{"tool-a"}, Allowed: []string{"tool-a"}}}, Strategies: []StrategySpec{{Kind: StrategyFullAdmitted}, {Kind: StrategyPriorityTopK, K: 4}, {Kind: StrategyCapabilityFiltered, Capability: "search"}}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Audit(in); err != nil {
			b.Fatal(err)
		}
	}
}
