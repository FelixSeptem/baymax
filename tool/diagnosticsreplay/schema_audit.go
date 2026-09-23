package diagnosticsreplay

import "github.com/FelixSeptem/baymax/tool/schemaaudit"

// ToolSchemaPressureSelectionAuditV1 is the stable offline audit namespace.
const ToolSchemaPressureSelectionAuditV1 = schemaaudit.AuditVersion

const (
	ReasonCodeSchemaAuditOverflow            = schemaaudit.CodeOverflow
	ReasonCodeSchemaAuditGoldSetConflict     = schemaaudit.CodeGoldSetConflict
	ReasonCodeSchemaAuditStrategyDrift       = schemaaudit.CodeUnsupportedStrategy
	ReasonCodeSchemaAuditReplayNotIdempotent = schemaaudit.CodeReplayNotIdempotent
	ReasonCodeSchemaAuditParityDrift         = schemaaudit.CodeRunStreamParityDrift
)

// ReplayToolSchemaPressureSelectionAuditJSON replays a bounded audit fixture.
// It is a thin adapter so diagnostics tooling and gates share the package owner
// for normalization, metrics, defaults, and stable error classifications.
func ReplayToolSchemaPressureSelectionAuditJSON(raw []byte) (schemaaudit.Result, error) {
	return schemaaudit.ReplayJSON(raw)
}

// CompareToolSchemaPressureSelectionAuditParity compares normalized offline
// Run/Stream audit semantics without introducing a runtime diagnostics path.
func CompareToolSchemaPressureSelectionAuditParity(run, stream schemaaudit.Result) error {
	return schemaaudit.CompareParity(run, stream)
}
