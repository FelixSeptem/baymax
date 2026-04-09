Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$minChangeIdRaw = if ($env:BAYMAX_EXAMPLE_IMPACT_MIN_CHANGE_ID) { $env:BAYMAX_EXAMPLE_IMPACT_MIN_CHANGE_ID.Trim() } else { "70" }
$minChangeId = 0
if (-not [int]::TryParse($minChangeIdRaw, [ref]$minChangeId) -or $minChangeId -lt 0) {
    throw "[invalid-example-impact-value] BAYMAX_EXAMPLE_IMPACT_MIN_CHANGE_ID must be a non-negative integer, got: $minChangeIdRaw"
}

$allowedValues = @(
    "新增示例",
    "修改示例",
    "无需示例变更（附理由）"
)

function Get-ChangeNumericID {
    param(
        [Parameter(Mandatory = $true)][string]$ChangeName
    )
    $matches = [regex]::Matches($ChangeName.ToLowerInvariant(), "-a([0-9]+)(?:-|$)")
    if ($matches.Count -eq 0) {
        return $null
    }
    $raw = $matches[$matches.Count - 1].Groups[1].Value
    $parsed = 0
    if (-not [int]::TryParse($raw, [ref]$parsed)) {
        return $null
    }
    return $parsed
}

function Get-DeclarationValue {
    param(
        [Parameter(Mandatory = $true)][string]$ProposalPath
    )
    $lines = Get-Content -Path $ProposalPath
    $inSection = $false
    foreach ($line in $lines) {
        $trimmed = ([string]$line).Trim()
        if (-not $inSection) {
            if ($trimmed -match '(?i)^##\s*example impact assessment\s*$' -or $trimmed -match '^##\s*示例影响评估\s*$') {
                $inSection = $true
            }
            continue
        }
        if ($trimmed -match '^##\s+') {
            break
        }
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }
        $candidate = $trimmed
        $candidate = $candidate -replace '^[\-\*]+\s*', ''
        $candidate = $candidate -replace '^\[[xX ]\]\s*', ''
        $candidate = $candidate.Trim().Trim('`').Trim()
        if ([string]::IsNullOrWhiteSpace($candidate)) {
            continue
        }
        return $candidate
    }
    return ""
}

function Test-AllowedDeclarationValue {
    param(
        [Parameter(Mandatory = $true)][string]$Value
    )
    if ($allowedValues -contains $Value) {
        return $true
    }
    if ($Value.StartsWith("无需示例变更（附理由）：") -or $Value.StartsWith("无需示例变更（附理由）:")) {
        $reason = $Value.Substring("无需示例变更（附理由）".Length).TrimStart("：", ":").Trim()
        return -not [string]::IsNullOrWhiteSpace($reason)
    }
    return $false
}

$openspecOutput = Invoke-NativeCaptureStrict -Label "openspec list --json" -Command {
    openspec list --json
}
$openspecText = ($openspecOutput | ForEach-Object {
        if ($null -eq $_) { return "" }
        if ($_ -is [System.Management.Automation.ErrorRecord]) { return $_.ToString() }
        return [string]$_
    }) -join "`n"
$openspecPayload = $openspecText | ConvertFrom-Json
$activeChanges = @($openspecPayload.changes |
    Where-Object { ([string]$_.status).Trim().ToLowerInvariant() -eq "in-progress" } |
    ForEach-Object { ([string]$_.name).Trim() } |
    Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
    Sort-Object -Unique)

$issues = New-Object 'System.Collections.Generic.List[string]'

foreach ($change in $activeChanges) {
    $changeId = Get-ChangeNumericID -ChangeName $change
    if ($null -ne $changeId -and $changeId -lt $minChangeId) {
        continue
    }

    $proposalPath = "openspec/changes/$change/proposal.md"
    if (-not (Test-Path $proposalPath)) {
        $issues.Add("[missing-example-impact-declaration] ${change}: missing proposal file $proposalPath") | Out-Null
        continue
    }

    $declarationValue = Get-DeclarationValue -ProposalPath $proposalPath
    if ([string]::IsNullOrWhiteSpace($declarationValue)) {
        $issues.Add("[missing-example-impact-declaration] ${change}: missing Example Impact Assessment declaration in $proposalPath") | Out-Null
        continue
    }
    if (-not (Test-AllowedDeclarationValue -Value $declarationValue)) {
        $issues.Add("[invalid-example-impact-value] ${change}: unsupported declaration `"$declarationValue`" in $proposalPath") | Out-Null
    }
}

if ($issues.Count -gt 0) {
    foreach ($issue in $issues) {
        Write-Host $issue
    }
    Write-Host "hint: add section '## Example Impact Assessment' in proposal.md and use one of: 新增示例 | 修改示例 | 无需示例变更（附理由）"
    Write-Host "hint: this gate only enforces changes with numeric suffix >= a$minChangeId."
    throw "[missing-example-impact-declaration] openspec example impact declaration gate failed"
}

Write-Host "[openspec-example-impact-declaration] passed"
