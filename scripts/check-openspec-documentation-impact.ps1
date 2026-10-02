Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Set-Location (Resolve-Path (Join-Path $PSScriptRoot '..'))
$areas = @('architecture','components','configuration','contract/API','diagnostics','examples','CLI/integration','best practices','roadmap')
$issues = New-Object 'System.Collections.Generic.List[string]'
function Add-Issue([string]$Code, [string]$Detail) { $issues.Add("${Code}: $Detail") | Out-Null }
$active = @(Get-ChildItem -LiteralPath 'openspec/changes' -Directory | Where-Object { $_.Name -ne 'archive' -and (Test-Path (Join-Path $_.FullName 'proposal.md')) } | Sort-Object Name)
if ($active.Count -eq 0) { Write-Output 'No active OpenSpec changes; documentation impact gate has no proposal assessments to inspect.' }
foreach ($change in $active) {
    foreach ($name in @('proposal.md','design.md','tasks.md')) {
        $file = Join-Path $change.FullName $name
        if (-not (Test-Path $file)) { Add-Issue 'missing-artifact' "$($change.Name)/$name"; continue }
        $content = Get-Content -Raw $file
        if ($content -notmatch '(?im)^##\s+Documentation Impact Assessment\s*$') { Add-Issue 'missing-assessment' "$($change.Name)/$name"; continue }
        foreach ($area in $areas) {
            if ($content -notmatch "(?im)^\|\s*$([regex]::Escape($area))\s*\|\s*(新增文档|修改文档|无需文档变更（附理由）)\s*\|") { Add-Issue 'missing-area' "$($change.Name)/${name}:$area" }
        }
        foreach ($row in [regex]::Matches($content, '(?im)^\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|')) {
            $outcome = $row.Groups[2].Value.Trim(); $evidence = $row.Groups[5].Value.Trim()
            if ($outcome -eq '无需文档变更（附理由）' -and ($evidence -eq '—' -or $evidence.Length -lt 8)) { Add-Issue 'missing-reason' "$($change.Name)/${name}:$($row.Groups[1].Value.Trim())" }
            if ($outcome -in @('新增文档','修改文档') -and ($row.Groups[3].Value.Trim() -eq '—' -or $row.Groups[4].Value.Trim() -eq '—' -or $evidence -eq '—')) { Add-Issue 'missing-evidence' "$($change.Name)/${name}:$($row.Groups[1].Value.Trim())" }
        }
    }
}
$changed = @(git diff --name-only HEAD; git ls-files --others --exclude-standard) | ForEach-Object { $_.Trim().Replace('\','/') } | Where-Object { $_ } | Sort-Object -Unique
if (@($changed | Where-Object { $_ -match '(^|/).*\.go$|^examples/' }).Count -gt 0) {
    foreach ($change in $active) {
        $proposal = Get-Content -Raw (Join-Path $change.FullName 'proposal.md')
        foreach ($area in @('configuration','contract/API','diagnostics','examples')) {
            if ($proposal -notmatch "(?im)^\|\s*$([regex]::Escape($area))\s*\|\s*(新增文档|修改文档)\s*\|") { Add-Issue 'surface-without-doc-task' "$($change.Name):$area" }
        }
    }
}
$readme = Get-Content -Raw README.md
foreach ($ref in [regex]::Matches($readme, '(?i)(?:docs|openspec)/[A-Za-z0-9_./-]+\.md')) { if (-not (Test-Path $ref.Value)) { Add-Issue 'stale-link' $ref.Value } }
if ($issues.Count -gt 0) { $issues | Sort-Object; exit 1 }
Write-Output 'Documentation impact gate passed.'
