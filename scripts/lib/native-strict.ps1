Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Test-GoStatCachePermissionWarning {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][AllowEmptyCollection()][object[]]$Output
    )

    if ($null -eq $Output -or $Output.Count -eq 0) {
        return $false
    }

    $text = ($Output | ForEach-Object {
            if ($null -eq $_) {
                return ""
            }
            if ($_ -is [System.Management.Automation.ErrorRecord]) {
                return $_.ToString()
            }
            return [string]$_
        }) -join "`n"

    if ($text -match "go:\s+writing stat cache:" -and $text -match "(Access is denied|permission denied|operation not permitted|read-only file system)") {
        return $true
    }
    return $false
}

function Invoke-NativeStrict {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command,
        [switch]$AllowFailure
    )

    $output = @(& $Command 2>&1)
    foreach ($line in $output) {
        if ($null -eq $line) {
            continue
        }
        if ($line -is [System.Management.Automation.ErrorRecord]) {
            Write-Host ($line.ToString())
            continue
        }
        Write-Host ([string]$line)
    }

    $exitCode = $LASTEXITCODE
    if ($null -eq $exitCode) {
        $exitCode = 0
    }

    if (Test-GoStatCachePermissionWarning -Output $output) {
        throw ("[native-strict] command failed: " + $Label + " (go stat cache write permission warning detected)")
    }

    if ($exitCode -ne 0 -and -not $AllowFailure) {
        throw ("[native-strict] command failed: " + $Label + " (exit=" + $exitCode + ")")
    }
    if ($AllowFailure) {
        return [int]$exitCode
    }
}

function Invoke-NativeCaptureStrict {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command,
        [switch]$AllowFailure
    )

    $output = @(& $Command 2>&1)
    $exitCode = $LASTEXITCODE
    if ($null -eq $exitCode) {
        $exitCode = 0
    }

    if (Test-GoStatCachePermissionWarning -Output $output) {
        throw ("[native-strict] command failed: " + $Label + " (go stat cache write permission warning detected)")
    }

    if ($exitCode -ne 0 -and -not $AllowFailure) {
        throw ("[native-strict] command failed: " + $Label + " (exit=" + $exitCode + ")")
    }

    return $output
}
