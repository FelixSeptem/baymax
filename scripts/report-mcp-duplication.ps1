param(
    [string]$HttpPath = "mcp/http/client.go",
    [string]$StdioPath = "mcp/stdio/client.go",
    [string]$BaselinePath = "docs/metrics/mcp-duplication-baseline.json",
    [double]$MinReductionPct = 0,
    [switch]$WriteBaseline = $false
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Normalize-Lines {
    param([string]$Path)
    $raw = Get-Content -Path $Path
    $out = @()
    foreach ($line in $raw) {
        $s = $line.Trim()
        if ($s -eq "") { continue }
        if ($s.StartsWith("//")) { continue }
        if ($s -match "^package\s+") { continue }
        if ($s -eq "import (" -or $s -eq ")" -or $s -eq "{") { continue }
        $out += $s
    }
    return $out
}

function Build-CountMap {
    param([string[]]$Lines)
    $map = @{}
    foreach ($line in $Lines) {
        if ($line.Length -lt 8) { continue }
        if ($map.ContainsKey($line)) {
            $map[$line] = [int]$map[$line] + 1
        } else {
            $map[$line] = 1
        }
    }
    return $map
}

$httpLines = Normalize-Lines -Path $HttpPath
$stdioLines = Normalize-Lines -Path $StdioPath

$httpMap = Build-CountMap -Lines $httpLines
$stdioMap = Build-CountMap -Lines $stdioLines

$duplicated = 0
foreach ($k in $httpMap.Keys) {
    if (-not $stdioMap.ContainsKey($k)) { continue }
    $duplicated += [Math]::Min([int]$httpMap[$k], [int]$stdioMap[$k])
}

$minTotal = [Math]::Min($httpLines.Count, $stdioLines.Count)
$duplicatePct = 0.0
if ($minTotal -gt 0) {
    $duplicatePct = [Math]::Round(($duplicated / $minTotal) * 100, 2)
}

$result = [ordered]@{
    generated_at = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
    http_path = $HttpPath
    stdio_path = $StdioPath
    http_lines = $httpLines.Count
    stdio_lines = $stdioLines.Count
    duplicated_lines = $duplicated
    duplicate_pct = $duplicatePct
}

if ($WriteBaseline) {
    $dir = Split-Path -Parent $BaselinePath
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir | Out-Null
    }
    $result | ConvertTo-Json -Depth 4 | Set-Content -Path $BaselinePath
    Write-Host ("Baseline written: " + $BaselinePath)
    $result | ConvertTo-Json -Depth 4
    exit 0
}

if (Test-Path $BaselinePath) {
    $baseline = Get-Content -Raw $BaselinePath | ConvertFrom-Json
    $baselinePct = [double]$baseline.duplicate_pct
    $reductionPct = [Math]::Round($baselinePct - $duplicatePct, 2)
    $result["baseline_duplicate_pct"] = $baselinePct
    $result["reduction_pct"] = $reductionPct
    if ($MinReductionPct -gt 0 -and $reductionPct -lt $MinReductionPct) {
        $result | ConvertTo-Json -Depth 4
        throw ("duplicate reduction " + $reductionPct + "% is below threshold " + $MinReductionPct + "%")
    }
}

$result | ConvertTo-Json -Depth 4
