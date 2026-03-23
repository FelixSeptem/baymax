Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-NativeStrict {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command,
        [switch]$AllowFailure
    )

    & $Command
    $exitCode = $LASTEXITCODE
    if ($null -eq $exitCode) {
        $exitCode = 0
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

    $output = & $Command
    $exitCode = $LASTEXITCODE
    if ($null -eq $exitCode) {
        $exitCode = 0
    }

    if ($exitCode -ne 0 -and -not $AllowFailure) {
        throw ("[native-strict] command failed: " + $Label + " (exit=" + $exitCode + ")")
    }

    return $output
}
