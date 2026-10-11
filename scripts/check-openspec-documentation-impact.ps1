Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Set-Location (Resolve-Path (Join-Path $PSScriptRoot '..'))
$areas = @('architecture','components','configuration','contract/API','diagnostics','examples','CLI/integration','best practices','roadmap')
$issues = New-Object 'System.Collections.Generic.List[string]'
function Add-Issue([string]$Code, [string]$Detail) { $issues.Add("${Code}: $Detail") | Out-Null }
$changed = @(git diff --name-only HEAD; git ls-files --others --exclude-standard) | ForEach-Object { $_.Trim().Replace('\','/') } | Where-Object { $_ } | Sort-Object -Unique
$base = @(git merge-base HEAD master 2>$null | Select-Object -First 1)
if ($base.Count -gt 0 -and $base[0]) {
    $changed = @($changed; git diff --name-only --diff-filter=ACMRTUXB "$($base[0])..HEAD") | ForEach-Object { $_.Trim().Replace('\','/') } | Where-Object { $_ } | Sort-Object -Unique
}
$active = @(Get-ChildItem -LiteralPath 'openspec/changes' -Directory | Where-Object { $_.Name -ne 'archive' -and (Test-Path (Join-Path $_.FullName 'proposal.md')) } | Sort-Object Name)
if ($active.Count -eq 0) { Write-Output 'No active OpenSpec changes; documentation impact gate has no proposal assessments to inspect.' }
foreach ($change in $active) {
    $commitments = @{}
    foreach ($name in @('proposal.md','design.md','tasks.md')) {
        $file = Join-Path $change.FullName $name
        if (-not (Test-Path $file)) { Add-Issue 'missing-artifact' "$($change.Name)/$name"; continue }
        $content = Get-Content -Raw $file
        $section = [regex]::Match($content, '(?ms)^##\s+Public Semantic Documentation Commitment\s*$(.*?)(?=^##\s|\z)')
        if (-not $section.Success) { Add-Issue 'missing-public-semantic-documentation-commitment' "$($change.Name)/$name"; continue }
        $values = @{}
        foreach ($field in @('对外语义变化','文档同步状态','维护文档','维护 owner','验证证据')) {
            $fieldMatch = [regex]::Match($section.Groups[1].Value, "(?m)^\s*-\s*$([regex]::Escape($field))：\s*(.+?)\s*$")
            if (-not $fieldMatch.Success) { Add-Issue 'invalid-public-semantic-documentation-commitment' "$($change.Name)/${name}:$field" } else { $values[$field] = $fieldMatch.Groups[1].Value.Trim() }
        }
        if ($values.Count -eq 5) {
            $semantic = $values['对外语义变化']; $status = $values['文档同步状态']; $paths = $values['维护文档']; $owner = $values['维护 owner']; $evidence = $values['验证证据']
            if ($semantic -notin @('是','否')) { Add-Issue 'invalid-public-semantic-documentation-commitment' "$($change.Name)/${name}:对外语义变化" }
            elseif ($semantic -eq '是') {
                if ($status -ne '已同步') { Add-Issue 'public-semantic-doc-sync-missing' "$($change.Name)/${name}:文档同步状态" }
                $docPaths = @([regex]::Matches($paths, '`([^`]+\.md)`') | ForEach-Object { $_.Groups[1].Value })
                if ($docPaths.Count -eq 0 -or $owner -eq '—' -or $evidence -eq '—') { Add-Issue 'public-semantic-doc-sync-missing' "$($change.Name)/$name" }
                foreach ($docPath in $docPaths) {
                    if (-not (Test-Path -LiteralPath $docPath) -or $docPath -notin $changed) { Add-Issue 'public-semantic-doc-sync-missing' "$($change.Name)/${name}:$docPath" }
                }
            } else {
                if ($status -notmatch '^无需同步（附理由）[:：].{8,}$'.Replace('（', '\（').Replace('）', '\）')) { Add-Issue 'invalid-public-semantic-documentation-commitment' "$($change.Name)/${name}:文档同步状态" }
            }
            if ($owner -eq '—' -or $evidence -eq '—' -or $owner.Length -lt 2 -or $evidence.Length -lt 8) { Add-Issue 'invalid-public-semantic-documentation-commitment' "$($change.Name)/$name:owner/evidence" }
            $commitments[$name] = @($semantic, $status, $paths, $owner, $evidence)
        }
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
    if ($commitments.Count -eq 3) {
        $baseline = ($commitments['proposal.md'] -join "`n")
        foreach ($name in @('design.md','tasks.md')) { if (($commitments[$name] -join "`n") -ne $baseline) { Add-Issue 'public-semantic-documentation-drift' "$($change.Name):$name" } }
    }
}
if (@($changed | Where-Object { ($_ -match '(^|/).*\.go$' -and $_ -notmatch '_test\.go$') -or $_ -match '^examples/' }).Count -gt 0) {
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
