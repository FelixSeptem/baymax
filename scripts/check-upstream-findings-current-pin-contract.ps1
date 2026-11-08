Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$findingsPath = "docs/upstream/baymax-findings.md"
$evidencePath = "docs/baymax-contract-evidence.md"
$currentPin = "v0.0.0-20261103091945-d2e08cd6dc8c"
$legacyPin = "v0.0.0-20261022053159-3d0ab5949790"

foreach ($path in @($findingsPath, $evidencePath)) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "[upstream-findings-current-pin] missing evidence document: $path"
    }
}

$findings = Get-Content -LiteralPath $findingsPath -Raw
$evidence = Get-Content -LiteralPath $evidencePath -Raw
$combined = "$findings`n$evidence"

foreach ($document in @(@{ Path = $findingsPath; Text = $findings }, @{ Path = $evidencePath; Text = $evidence })) {
    if ($document.Text -notmatch [regex]::Escape($currentPin)) {
        throw "[upstream-findings-current-pin] $($document.Path) missing current pin $currentPin"
    }
}

if ($findings -notmatch [regex]::Escape($legacyPin) -or $findings -notmatch "(?i)historical/legacy") {
    throw "[upstream-findings-current-pin] legacy pin must be present and marked historical/legacy in $findingsPath"
}
if ($evidence -notmatch [regex]::Escape($legacyPin) -or $evidence -notmatch "(?i)historical/legacy") {
    throw "[upstream-findings-current-pin] legacy pin must be present and marked historical/legacy in $evidencePath"
}

foreach ($marker in @(
        "Child/composer continuation",
        "Parallel/vote team continuation",
        "Source history and projection recovery",
        "Mailbox lifecycle",
        "Native cancellation and timeout",
        "Native causal retry",
        "Provider live-smoke"
    )) {
    if ($findings -notmatch [regex]::Escape($marker)) {
        throw "[upstream-findings-current-pin] findings missing required row: $marker"
    }
}

foreach ($marker in @(
        '| child / composer |',
        '| `team_parallel` |',
        '| `team_vote` |',
        '| workflow |',
        '| mailbox lifecycle |',
        '| native cancellation / timeout |',
        '| native causal retry |',
        '| provider live paths |'
    )) {
    if ($evidence -notmatch [regex]::Escape($marker)) {
        throw "[upstream-findings-current-pin] matrix missing required row: $marker"
    }
}

foreach ($marker in @(
        "same-host",
        "cross-machine",
        "source/projection",
        "non-atomic",
        "recovery-required",
        "skipped-unconfigured",
        "unsupported"
    )) {
    if ($combined -notmatch [regex]::Escape($marker)) {
        throw "[upstream-findings-current-pin] missing limitation marker: $marker"
    }
}

if ($combined -match "(?i)all execution forms.{0,80}(support|supported).{0,80}durable replay") {
    throw "[upstream-findings-current-pin] evidence overclaims durable replay for all execution forms"
}
if ($combined -match "(?i)cross-machine coordination (is|remains) (supported|verified)" -or
    $combined -match "(?i)source/projection (atomic commit|atomic transaction|atomicity is supported)") {
    throw "[upstream-findings-current-pin] evidence overclaims cross-machine or atomic support"
}
if ($combined -notmatch "(?i)not (a|an) (Baymax )?(upstream )?bug") {
    throw "[upstream-findings-current-pin] skipped-unconfigured must not be classified as a bug"
}

Write-Host "[upstream-findings-current-pin] passed"
