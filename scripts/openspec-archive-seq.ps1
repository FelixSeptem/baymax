param(
    [string]$ChangeName = "",
    [switch]$MigrateExisting = $false
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$changesDir = Join-Path $repoRoot "openspec/changes"
$archiveDir = Join-Path $changesDir "archive"
$indexPath = Join-Path $archiveDir "INDEX.md"

# Existing historical order requested by the team.
$preferredOrder = @(
    "build-go-agent-loop-framework",
    "upgrade-openai-native-stream-mapping",
    "optimize-runtime-concurrency-and-async-io"
)

function Get-SlugFromArchiveDirName {
    param([string]$Name)
    if ($Name -match "^\d{3}-(.+)$") {
        return $Matches[1]
    }
    if ($Name -match "^\d{4}-\d{2}-\d{2}-(.+)$") {
        return $Matches[1]
    }
    return $Name
}

function Get-SeqFromArchiveDirName {
    param([string]$Name)
    if ($Name -match "^(\d{3})-.+$") {
        return [int]$Matches[1]
    }
    return $null
}

function Get-NextSequence {
    param([string]$ArchiveRoot)
    $maxSeq = 0
    if (Test-Path $ArchiveRoot) {
        Get-ChildItem -Path $ArchiveRoot -Directory | ForEach-Object {
            $seq = Get-SeqFromArchiveDirName -Name $_.Name
            if ($null -ne $seq -and $seq -gt $maxSeq) {
                $maxSeq = $seq
            }
        }
    }
    return ($maxSeq + 1)
}

function Write-ArchiveIndex {
    param([string]$ArchiveRoot, [string]$OutputPath)
    $lines = @(
        "# Archive Index",
        "",
        "Updated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')",
        ""
    )
    $dirs = Get-ChildItem -Path $ArchiveRoot -Directory |
        Sort-Object {
            $seq = Get-SeqFromArchiveDirName -Name $_.Name
            if ($null -eq $seq) { return 999999 }
            return $seq
        }, Name

    foreach ($d in $dirs) {
        $seq = Get-SeqFromArchiveDirName -Name $d.Name
        $slug = Get-SlugFromArchiveDirName -Name $d.Name
        if ($null -ne $seq) {
            $lines += ("- {0:D3} -> {1}" -f $seq, $slug)
        } else {
            $lines += ("- n/a -> {0}" -f $slug)
        }
    }
    $lines | Set-Content -Path $OutputPath
}

function Rename-ArchiveDirsByPlan {
    param(
        [string]$ArchiveRoot,
        [array]$Plan
    )

    # Two-phase rename to avoid collisions.
    $tmpPlan = New-Object System.Collections.ArrayList
    foreach ($item in $Plan) {
        $src = Join-Path $ArchiveRoot $item.Source
        $tmp = Join-Path $ArchiveRoot ("__tmp__" + [guid]::NewGuid().ToString("N"))
        Rename-Item -Path $src -NewName (Split-Path -Leaf $tmp)
        [void]$tmpPlan.Add([pscustomobject]@{
                Temp   = Split-Path -Leaf $tmp
                Target = $item.Target
            })
    }
    foreach ($item in $tmpPlan) {
        $tmp = Join-Path $ArchiveRoot $item.Temp
        Rename-Item -Path $tmp -NewName $item.Target
    }
}

function Migrate-ExistingArchive {
    param([string]$ArchiveRoot, [string[]]$Preferred)

    if (-not (Test-Path $ArchiveRoot)) {
        New-Item -ItemType Directory -Path $ArchiveRoot | Out-Null
    }

    $dirs = Get-ChildItem -Path $ArchiveRoot -Directory
    if ($dirs.Count -eq 0) {
        return
    }

    $used = @{}
    $plan = New-Object System.Collections.ArrayList

    # First, enforce explicit historical order.
    for ($i = 0; $i -lt $Preferred.Count; $i++) {
        $slug = $Preferred[$i]
        $target = ("{0:D3}-{1}" -f ($i + 1), $slug)
        $match = $dirs | Where-Object { (Get-SlugFromArchiveDirName -Name $_.Name) -eq $slug } | Select-Object -First 1
        if ($null -eq $match) {
            continue
        }
        $used[$match.FullName] = $true
        if ($match.Name -ne $target) {
            [void]$plan.Add([pscustomobject]@{
                    Source = $match.Name
                    Target = $target
                })
        }
    }

    # Then assign sequence numbers to remaining directories.
    $next = $Preferred.Count + 1
    $left = $dirs | Where-Object { -not $used.ContainsKey($_.FullName) } | Sort-Object LastWriteTime, Name
    foreach ($d in $left) {
        $slug = Get-SlugFromArchiveDirName -Name $d.Name
        $target = ("{0:D3}-{1}" -f $next, $slug)
        if ($d.Name -ne $target) {
            [void]$plan.Add([pscustomobject]@{
                    Source = $d.Name
                    Target = $target
                })
        }
        $next++
    }

    if ($plan.Count -gt 0) {
        Rename-ArchiveDirsByPlan -ArchiveRoot $ArchiveRoot -Plan $plan
    }
}

function Archive-OneChange {
    param(
        [string]$Change,
        [string]$ChangesRoot,
        [string]$ArchiveRoot
    )

    if ([string]::IsNullOrWhiteSpace($Change)) {
        return
    }

    $activeDir = Join-Path $ChangesRoot $Change

    Push-Location $repoRoot
    try {
        & openspec archive $Change -y
    } catch {
        # Keep going because OpenSpec may already have copied specs and archive folder
        # but fail during unlink on Windows.
        Write-Warning ("openspec archive returned error: " + $_.Exception.Message)
    } finally {
        Pop-Location
    }

    $escaped = [regex]::Escape($Change)
    $archivedDir = Get-ChildItem -Path $ArchiveRoot -Directory |
        Where-Object { $_.Name -match ("^(\d{3}|\d{4}-\d{2}-\d{2})-" + $escaped + "$") } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1

    if ($null -eq $archivedDir) {
        Write-Warning ("archive output for change '" + $Change + "' was not found; skip cleanup.")
        return
    }

    $required = @(".openspec.yaml", "proposal.md", "tasks.md")
    if (Test-Path (Join-Path $activeDir "design.md")) {
        $required += "design.md"
    }
    foreach ($rel in $required) {
        if (-not (Test-Path (Join-Path $archivedDir.FullName $rel))) {
            Write-Warning ("archive incomplete for '" + $Change + "': missing " + $rel + "; skip cleanup.")
            return
        }
    }

    $activeSpecsDir = Join-Path $activeDir "specs"
    if (Test-Path $activeSpecsDir) {
        $activeSpecFiles = Get-ChildItem -Path $activeSpecsDir -Recurse -File -Filter "spec.md"
        foreach ($spec in $activeSpecFiles) {
            $relative = $spec.FullName.Substring($activeDir.Length).TrimStart('\', '/')
            if (-not (Test-Path (Join-Path $archivedDir.FullName $relative))) {
                Write-Warning ("archive incomplete for '" + $Change + "': missing " + $relative + "; skip cleanup.")
                return
            }
        }
    }

    $archivedSpecCount = (Get-ChildItem -Path $archivedDir.FullName -Recurse -File -Filter "spec.md" | Measure-Object).Count
    if ($archivedSpecCount -eq 0) {
        Write-Warning ("archive incomplete for '" + $Change + "': no spec.md found; skip cleanup.")
        return
    }

    # Cleanup active change dir if still exists.
    if (Test-Path $activeDir) {
        try {
            & attrib -R $activeDir /S /D | Out-Null
            Remove-Item -Recurse -Force $activeDir
        } catch {
            Write-Warning ("failed to remove active change dir '" + $activeDir + "': " + $_.Exception.Message)
        }
    }

    # If new archive is still date-based, rename to next sequence.
    if ($archivedDir.Name -match "^\d{3}-$escaped$") {
        return
    }

    if ($archivedDir.Name -notmatch "^\d{4}-\d{2}-\d{2}-$escaped$") {
        return
    }

    $nextSeq = Get-NextSequence -ArchiveRoot $ArchiveRoot
    $target = "{0:D3}-{1}" -f $nextSeq, $Change
    Rename-Item -Path $archivedDir.FullName -NewName $target
}

if (-not (Test-Path $archiveDir)) {
    New-Item -ItemType Directory -Path $archiveDir | Out-Null
}

if ($MigrateExisting) {
    Migrate-ExistingArchive -ArchiveRoot $archiveDir -Preferred $preferredOrder
}

if (-not [string]::IsNullOrWhiteSpace($ChangeName)) {
    Archive-OneChange -Change $ChangeName -ChangesRoot $changesDir -ArchiveRoot $archiveDir
    # Re-run migration to keep historical order fixed and append unknowns after it.
    Migrate-ExistingArchive -ArchiveRoot $archiveDir -Preferred $preferredOrder
}

Write-ArchiveIndex -ArchiveRoot $archiveDir -OutputPath $indexPath
Write-Host "Done. Archive naming is normalized and INDEX.md is updated."
