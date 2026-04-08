Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

function Get-EnvOrDefault {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [string]$Default = ""
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $Default
    }
    return $value
}

function Parse-Mode {
    $explicit = (Get-EnvOrDefault -Name "BAYMAX_A64_GATE_SELECTION_MODE" -Default "").Trim().ToLowerInvariant()
    if ($explicit -ne "") {
        if ($explicit -ne "fast" -and $explicit -ne "full") {
            throw "[a64-impacted-gate-selection] BAYMAX_A64_GATE_SELECTION_MODE must be fast|full, got: $explicit"
        }
        return $explicit
    }
    $scope = (Get-EnvOrDefault -Name "BAYMAX_QUALITY_GATE_SCOPE" -Default "full").Trim().ToLowerInvariant()
    if ($scope -eq "general") {
        return "fast"
    }
    return "full"
}

function Parse-ChangedFilesFromEnv {
    $raw = Get-EnvOrDefault -Name "BAYMAX_A64_CHANGED_FILES" -Default ""
    if ([string]::IsNullOrWhiteSpace($raw)) {
        return @()
    }
    $parts = $raw -split "[`r`n,;]"
    return @($parts | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" } | Sort-Object -Unique)
}

function Get-ChangedFilesFromGit {
    $tracked = Invoke-NativeCaptureStrict -Label "git diff --name-only HEAD" -Command {
        git diff --name-only HEAD
    }
    $untracked = Invoke-NativeCaptureStrict -Label "git ls-files --others --exclude-standard" -Command {
        git ls-files --others --exclude-standard
    }
    return @($tracked + $untracked | ForEach-Object { [string]$_ } | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" } | Sort-Object -Unique)
}

function New-A64SuitesMap {
    return @{
        S1  = @{
            Shell      = @(
                "go test ./context/assembler ./context/provider ./context/journal -count=1",
                "bash scripts/check-diagnostics-replay-contract.sh"
            )
            PowerShell = @(
                "go test ./context/assembler ./context/provider ./context/journal -count=1",
                "pwsh -File scripts/check-diagnostics-replay-contract.ps1"
            )
        }
        S2  = @{
            Shell      = @(
                "bash scripts/check-diagnostics-replay-contract.sh",
                "bash scripts/check-diagnostics-query-performance-regression.sh"
            )
            PowerShell = @(
                "pwsh -File scripts/check-diagnostics-replay-contract.ps1",
                "pwsh -File scripts/check-diagnostics-query-performance-regression.ps1"
            )
        }
        S3  = @{
            Shell      = @(
                "bash scripts/check-multi-agent-shared-contract.sh",
                "go test ./orchestration/scheduler ./orchestration/composer -count=1"
            )
            PowerShell = @(
                "pwsh -File scripts/check-multi-agent-shared-contract.ps1",
                "go test ./orchestration/scheduler ./orchestration/composer -count=1"
            )
        }
        S4  = @{
            Shell      = @(
                "go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1",
                "bash scripts/check-multi-agent-shared-contract.sh"
            )
            PowerShell = @(
                "go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1",
                "pwsh -File scripts/check-multi-agent-shared-contract.ps1"
            )
        }
        S5  = @{
            Shell      = @("go test ./skill/loader ./runtime/config -count=1")
            PowerShell = @("go test ./skill/loader ./runtime/config -count=1")
        }
        S6  = @{
            Shell      = @(
                "bash scripts/check-memory-contract-conformance.sh",
                "bash scripts/check-memory-scope-and-search-contract.sh"
            )
            PowerShell = @(
                "pwsh -File scripts/check-memory-contract-conformance.ps1",
                "pwsh -File scripts/check-memory-scope-and-search-contract.ps1"
            )
        }
        S7  = @{
            Shell      = @(
                "bash scripts/check-security-policy-contract.sh",
                "bash scripts/check-security-event-contract.sh",
                "bash scripts/check-security-delivery-contract.sh",
                "bash scripts/check-security-sandbox-contract.sh"
            )
            PowerShell = @(
                "pwsh -File scripts/check-security-policy-contract.ps1",
                "pwsh -File scripts/check-security-event-contract.ps1",
                "pwsh -File scripts/check-security-delivery-contract.ps1",
                "pwsh -File scripts/check-security-sandbox-contract.ps1"
            )
        }
        S8  = @{
            Shell      = @("bash scripts/check-react-contract.sh")
            PowerShell = @("pwsh -File scripts/check-react-contract.ps1")
        }
        S9  = @{
            Shell      = @(
                "bash scripts/check-policy-precedence-contract.sh",
                "bash scripts/check-runtime-budget-admission-contract.sh",
                "bash scripts/check-sandbox-rollout-governance-contract.sh"
            )
            PowerShell = @(
                "pwsh -File scripts/check-policy-precedence-contract.ps1",
                "pwsh -File scripts/check-runtime-budget-admission-contract.ps1",
                "pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1"
            )
        }
        S10 = @{
            Shell      = @(
                "bash scripts/check-observability-export-and-bundle-contract.sh",
                "bash scripts/check-diagnostics-replay-contract.sh"
            )
            PowerShell = @(
                "pwsh -File scripts/check-observability-export-and-bundle-contract.ps1",
                "pwsh -File scripts/check-diagnostics-replay-contract.ps1"
            )
        }
    }
}

function New-A64PathMatchers {
    return @{
        S1  = @("^context/assembler/", "^context/provider/", "^context/journal/")
        S2  = @("^runtime/diagnostics/", "^observability/event/runtime_recorder")
        S3  = @("^orchestration/scheduler/", "^orchestration/mailbox/", "^orchestration/composer/")
        S4  = @("^mcp/http/", "^mcp/stdio/", "^mcp/retry/", "^mcp/diag/")
        S5  = @("^skill/loader/")
        S6  = @("^memory/")
        S7  = @("^core/runner/", "^tool/local/", "^orchestration/teams/", "^orchestration/workflow/")
        S8  = @("^model/openai/", "^model/anthropic/", "^model/gemini/")
        S9  = @("^runtime/config/")
        S10 = @("^observability/event/dispatcher", "^observability/event/logger", "^observability/event/runtime_exporter")
    }
}

function Is-CrossCuttingPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    $normalized = $Path.Trim()
    if ($normalized -match "^scripts/check-a64-") { return $true }
    if ($normalized -match "^openspec/changes/introduce-engineering-and-performance-optimization-contract-a64/") { return $true }
    if ($normalized -match "^docs/development-roadmap\.md$") { return $true }
    if ($normalized -match "^docs/mainline-contract-test-index\.md$") { return $true }
    if ($normalized -match "^docs/runtime-config-diagnostics\.md$") { return $true }
    return $false
}

function Resolve-ImpactedSItems {
    param(
        [Parameter(Mandatory = $true)][string]$Mode,
        [Parameter(Mandatory = $true)][string[]]$ChangedFiles,
        [Parameter(Mandatory = $true)][hashtable]$PathMatchers,
        [Parameter(Mandatory = $true)][string[]]$AllS
    )
    if ($Mode -eq "full") {
        return $AllS
    }
    $set = New-Object 'System.Collections.Generic.HashSet[string]'
    foreach ($file in $ChangedFiles) {
        $path = $file.Trim()
        if ($path -eq "") { continue }
        if (Is-CrossCuttingPath -Path $path) {
            foreach ($item in $AllS) {
                [void]$set.Add($item)
            }
            continue
        }
        foreach ($item in $AllS) {
            foreach ($pattern in $PathMatchers[$item]) {
                if ($path -match $pattern) {
                    [void]$set.Add($item)
                    break
                }
            }
        }
    }
    return @($set | Sort-Object)
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$mode = Parse-Mode
$suites = New-A64SuitesMap
$pathMatchers = New-A64PathMatchers
$allS = @("S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8", "S9", "S10")

$changedFiles = @(Parse-ChangedFilesFromEnv)
if ($changedFiles.Count -eq 0) {
    $changedFiles = @(Get-ChangedFilesFromGit)
}

$impacted = Resolve-ImpactedSItems -Mode $mode -ChangedFiles $changedFiles -PathMatchers $pathMatchers -AllS $allS
if ($mode -eq "fast" -and $changedFiles.Count -gt 0 -and $impacted.Count -eq 0) {
    throw "[a64-impacted-gate-selection] fast mode selected but no impacted S-items resolved; mapping must be updated"
}

foreach ($item in $impacted) {
    if (-not $suites.ContainsKey($item)) {
        throw "[a64-impacted-gate-selection] missing suite mapping for $item"
    }
    $entry = $suites[$item]
    $shell = @($entry.Shell)
    $pwsh = @($entry.PowerShell)
    if ($shell.Count -eq 0 -or $pwsh.Count -eq 0) {
        throw "[a64-impacted-gate-selection] incomplete suite mapping for $item (shell/powershell must both be non-empty)"
    }
}

$report = [ordered]@{
    mode                = $mode
    changed_file_total  = $changedFiles.Count
    impacted_s_items    = $impacted
    mandatory_suite_map = @{}
}
foreach ($item in $impacted) {
    $entry = $suites[$item]
    $report.mandatory_suite_map[$item] = [ordered]@{
        shell      = @($entry.Shell)
        powershell = @($entry.PowerShell)
    }
}

$json = $report | ConvertTo-Json -Depth 6
Write-Host "[a64-impacted-gate-selection] report:"
Write-Host $json

$outputPath = (Get-EnvOrDefault -Name "BAYMAX_A64_IMPACTED_REPORT_PATH" -Default "").Trim()
if ($outputPath -ne "") {
    $parent = Split-Path -Parent $outputPath
    if ($parent -and -not (Test-Path -LiteralPath $parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    Set-Content -LiteralPath $outputPath -Value $json -NoNewline
    Write-Host "[a64-impacted-gate-selection] report written to $outputPath"
}

Write-Host "[a64-impacted-gate-selection] passed"
