Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

function Assert-ContainsLiteral {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string]$Literal
    )

    $fullPath = Join-Path $repoRoot $FilePath
    if (-not (Test-Path -LiteralPath $fullPath)) {
        throw "[agent-eval-tracing-interop-gate][$Assertion] missing file: $FilePath"
    }
    $content = Get-Content -LiteralPath $fullPath -Raw
    if (-not $content.Contains($Literal)) {
        throw "[agent-eval-tracing-interop-gate][$Assertion] missing marker '$Literal' in $FilePath"
    }
}

function Assert-PatternAbsentAcrossRepo {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$Pattern
    )

    $archiveRoot = [Regex]::Escape((Join-Path $repoRoot "openspec\changes\archive"))
    $files = Get-ChildItem -Path $repoRoot -Recurse -File | Where-Object {
        $_.FullName -notmatch $archiveRoot
    }

    $matches = @()
    foreach ($file in $files) {
        $hit = Select-String -Path $file.FullName -Pattern $Pattern -ErrorAction SilentlyContinue
        if ($hit) {
            $matches += $hit
            if ($matches.Count -ge 10) {
                break
            }
        }
    }

    if ($matches.Count -gt 0) {
        $preview = ($matches | Select-Object -First 10 | ForEach-Object {
                "$($_.Path):$($_.LineNumber): $($_.Line.Trim())"
            }) -join "`n"
        throw "[agent-eval-tracing-interop-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
    }
}

function Assert-NoParallelA61Changes {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion
    )

    $changeRoot = Join-Path $repoRoot "openspec/changes"
    $canonical = "introduce-otel-tracing-and-agent-eval-interoperability-contract-a61"
    $violations = @()
    $dirs = Get-ChildItem -Path $changeRoot -Directory | Where-Object { $_.Name -ne "archive" }
    foreach ($dir in $dirs) {
        $lower = $dir.Name.ToLowerInvariant()
        if ($dir.Name -ne $canonical -and $lower.Contains("eval") -and ($lower.Contains("otel") -or $lower.Contains("tracing"))) {
            $violations += $dir.Name
        }
    }
    if ($violations.Count -gt 0) {
        throw "[agent-eval-tracing-interop-gate][$Assertion] parallel tracing/eval proposal detected: $($violations -join ', ')"
    }
}

function Invoke-AgentEvalTracingStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[agent-eval-tracing-interop-gate] $Label"
    & $Command
}

Invoke-AgentEvalTracingStep -Label "assertion control_plane_absent: contract marker" -Command {
    Assert-ContainsLiteral -Assertion "control_plane_absent" -FilePath "openspec/changes/introduce-otel-tracing-and-agent-eval-interoperability-contract-a61/specs/runtime-otel-tracing-and-agent-eval-interoperability-contract/spec.md" -Literal "embedded library behavior"
}

Invoke-AgentEvalTracingStep -Label "assertion control_plane_absent: gate spec marker" -Command {
    Assert-ContainsLiteral -Assertion "control_plane_absent" -FilePath "openspec/changes/introduce-otel-tracing-and-agent-eval-interoperability-contract-a61/specs/go-quality-gate/spec.md" -Literal "control_plane_absent"
}

Invoke-AgentEvalTracingStep -Label "assertion control_plane_absent: active change set closure" -Command {
    Assert-NoParallelA61Changes -Assertion "control_plane_absent"
}

Invoke-AgentEvalTracingStep -Label "assertion control_plane_absent: reject eval execution control-plane key drift" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "control_plane_absent" -Pattern "runtime\.eval\.execution\.[a-zA-Z0-9_.-]*(control_plane|controlplane|scheduler_service|orchestrator_endpoint|controller_endpoint|hosted_scheduler|remote_scheduler)"
}

Invoke-AgentEvalTracingStep -Label "assertion a61_field_reuse_required: upstream reuse marker" -Command {
    Assert-ContainsLiteral -Assertion "a61_field_reuse_required" -FilePath "openspec/changes/introduce-otel-tracing-and-agent-eval-interoperability-contract-a61/specs/runtime-otel-tracing-and-agent-eval-interoperability-contract/spec.md" -Literal "Tracing and eval outputs SHALL reuse canonical upstream explainability fields"
}

Invoke-AgentEvalTracingStep -Label "assertion a61_field_reuse_required: gate spec marker" -Command {
    Assert-ContainsLiteral -Assertion "a61_field_reuse_required" -FilePath "openspec/changes/introduce-otel-tracing-and-agent-eval-interoperability-contract-a61/specs/go-quality-gate/spec.md" -Literal "Quality gate SHALL include tracing and eval interoperability contract checks"
}

Invoke-AgentEvalTracingStep -Label "assertion a61_field_reuse_required: roadmap closure marker" -Command {
    Assert-ContainsLiteral -Assertion "a61_field_reuse_required" -FilePath "docs/development-roadmap.md" -Literal "A61 tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在 A61 内以增量任务吸收，不再新开平行提案。"
}

Invoke-AgentEvalTracingStep -Label "assertion a61_field_reuse_required: reject duplicated upstream alias fields" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "a61_field_reuse_required" -Pattern "runtime\.eval\.[a-zA-Z0-9_.-]*(policy_decision_path|deny_source|winner_stage|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action|budget_snapshot|budget_decision|degrade_action)"
}

Invoke-AgentEvalTracingStep -Label "runtime config tracing+eval schema and rollback suites" -Command {
    Invoke-NativeStrict -Label "go test ./runtime/config -run 'Test(RuntimeObservabilityConfigDefaults|RuntimeObservabilityConfigEnvOverridePrecedence|RuntimeObservabilityConfigValidationRejectsInvalidValues|RuntimeObservabilityConfigInvalidBoolFailsFast|RuntimeObservabilityTracingEndpointFallbackToExportOTLPEndpoint|ManagerRuntimeObservabilityTracingInvalidReloadRollsBack|RuntimeEvalConfigDefaults|RuntimeEvalConfigEnvOverridePrecedence|RuntimeEvalConfigValidationRejectsInvalidValues|RuntimeEvalConfigInvalidBoolFailsFast|ManagerRuntimeEvalInvalidReloadRollsBack|BuildEvalSummaryThresholdBoundaries|AggregateEvalShardMetricsLocalAndDistributedEquivalence|AggregateEvalShardMetricsResumeIdempotent|RuntimeEvalExecutionConfigBoundaryNoControlPlaneDependency)' -count=1" -Command {
        go test ./runtime/config -run 'Test(RuntimeObservabilityConfigDefaults|RuntimeObservabilityConfigEnvOverridePrecedence|RuntimeObservabilityConfigValidationRejectsInvalidValues|RuntimeObservabilityConfigInvalidBoolFailsFast|RuntimeObservabilityTracingEndpointFallbackToExportOTLPEndpoint|ManagerRuntimeObservabilityTracingInvalidReloadRollsBack|RuntimeEvalConfigDefaults|RuntimeEvalConfigEnvOverridePrecedence|RuntimeEvalConfigValidationRejectsInvalidValues|RuntimeEvalConfigInvalidBoolFailsFast|ManagerRuntimeEvalInvalidReloadRollsBack|BuildEvalSummaryThresholdBoundaries|AggregateEvalShardMetricsLocalAndDistributedEquivalence|AggregateEvalShardMetricsResumeIdempotent|RuntimeEvalExecutionConfigBoundaryNoControlPlaneDependency)' -count=1
    }
}

Invoke-AgentEvalTracingStep -Label "tracing semconv/export + diagnostics additive suites" -Command {
    Invoke-NativeStrict -Label "go test ./observability/trace ./observability/event ./runtime/diagnostics -run 'Test(CanonicalSemconvTopologyV1CoversCoreDomains|CanonicalAttributeMapInjectsSchemaAndFiltersUnknownKeys|RunStreamSemanticEquivalenceAllowsOrderingDifferences|RunStreamSemanticEquivalenceDetectsTopologyDrift|ExportRuntime.*|RuntimeRecorderParsesA61TracingEvalAdditiveFields|RuntimeRecorderA61ParserCompatibilityAdditiveNullableDefault|StoreRunA61TracingEvalAdditiveFieldsPersistAndReplayIdempotent|StoreRunA61TracingEvalAdditiveFieldsBoundedCardinality)' -count=1" -Command {
        go test ./observability/trace ./observability/event ./runtime/diagnostics -run 'Test(CanonicalSemconvTopologyV1CoversCoreDomains|CanonicalAttributeMapInjectsSchemaAndFiltersUnknownKeys|RunStreamSemanticEquivalenceAllowsOrderingDifferences|RunStreamSemanticEquivalenceDetectsTopologyDrift|ExportRuntime.*|RuntimeRecorderParsesA61TracingEvalAdditiveFields|RuntimeRecorderA61ParserCompatibilityAdditiveNullableDefault|StoreRunA61TracingEvalAdditiveFieldsPersistAndReplayIdempotent|StoreRunA61TracingEvalAdditiveFieldsBoundedCardinality)' -count=1
    }
}

Invoke-AgentEvalTracingStep -Label "replay fixtures and drift taxonomy suites (A61)" -Command {
    Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContract.*(Otel|Eval|A61)' -count=1" -Command {
        go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContract.*(Otel|Eval|A61)' -count=1
    }
}

Write-Host "[agent-eval-tracing-interop-gate] contributioncheck parity suites for A61 gate"
Invoke-NativeStrict -Label "go test ./tool/contributioncheck -run 'Test(AgentEvalTracingInteropGateScriptParity|QualityGateIncludesAgentEvalTracingInteropGate|AgentEvalTracingInteropRoadmapAndContractIndexClosureMarkers)' -count=1" -Command {
    go test ./tool/contributioncheck -run 'Test(AgentEvalTracingInteropGateScriptParity|QualityGateIncludesAgentEvalTracingInteropGate|AgentEvalTracingInteropRoadmapAndContractIndexClosureMarkers)' -count=1
}

Write-Host "[agent-eval-tracing-interop-gate] done"
