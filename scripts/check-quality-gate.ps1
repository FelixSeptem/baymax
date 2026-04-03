Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Test-WritableDirectory {
    param(
        [Parameter(Mandatory = $false)][string]$Path
    )
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return $false
    }
    try {
        if (-not (Test-Path -LiteralPath $Path)) {
            New-Item -ItemType Directory -Path $Path -Force | Out-Null
        }
        $probe = Join-Path $Path ("._write_probe_" + [Guid]::NewGuid().ToString("N"))
        [System.IO.File]::WriteAllText($probe, "ok")
        Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
        return $true
    }
    catch {
        return $false
    }
}

function Ensure-WritableCacheEnv {
    param(
        [Parameter(Mandatory = $true)][string]$EnvName,
        [Parameter(Mandatory = $true)][string]$FallbackPath
    )
    $current = [Environment]::GetEnvironmentVariable($EnvName)
    if (Test-WritableDirectory -Path $current) {
        return
    }
    if (-not (Test-WritableDirectory -Path $FallbackPath)) {
        throw "[quality-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
    }
    Set-Item -Path ("Env:" + $EnvName) -Value $FallbackPath
}

Ensure-WritableCacheEnv -EnvName "GOCACHE" -FallbackPath (Join-Path $repoRoot ".gocache")
Ensure-WritableCacheEnv -EnvName "GOLANGCI_LINT_CACHE" -FallbackPath (Join-Path $repoRoot ".gocache/golangci-lint")

if (-not $env:GOPROXY) {
    $env:GOPROXY = "https://proxy.golang.org,direct"
}
if (-not $env:GOSUMDB) {
    $env:GOSUMDB = "sum.golang.org"
}
if (-not $env:GOVULNDB) {
    $env:GOVULNDB = "https://vuln.go.dev"
}
if (-not $env:CGO_ENABLED) {
    $env:CGO_ENABLED = "1"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

function Get-PositiveIntEnvOrDefault {
    param(
        [Parameter(Mandatory = $true)][string]$EnvName,
        [Parameter(Mandatory = $true)][int]$DefaultValue
    )
    $raw = [Environment]::GetEnvironmentVariable($EnvName)
    if ([string]::IsNullOrWhiteSpace($raw)) {
        return $DefaultValue
    }
    $parsed = 0
    if (-not [int]::TryParse($raw, [ref]$parsed) -or $parsed -le 0) {
        throw "[quality-gate] $EnvName must be a positive integer, got: $raw"
    }
    return $parsed
}

function Stop-ProcessTree {
    param(
        [Parameter(Mandatory = $true)][int]$ProcessId,
        [Parameter(Mandatory = $false)][System.Diagnostics.Process]$Process
    )
    if ($ProcessId -le 0) {
        return $false
    }
    Write-Warning "[quality-gate] killing timed out process tree pid=$ProcessId"
    if ($null -ne $Process) {
        try {
            if (-not $Process.HasExited) {
                $Process.Kill($true)
                [void]$Process.WaitForExit(5000)
                return $true
            }
        }
        catch {
            Write-Warning "[quality-gate] managed process kill failed for pid=${ProcessId}: $($_.Exception.Message)"
        }
    }

    $output = @(& taskkill /PID $ProcessId /T /F 2>&1)
    $killed = $false
    foreach ($line in $output) {
        if ($null -eq $line) {
            continue
        }
        $text = if ($line -is [System.Management.Automation.ErrorRecord]) { $line.ToString() } else { [string]$line }
        if ($text -match "SUCCESS") {
            $killed = $true
        }
        Write-Host $text
    }
    return $killed
}

function Invoke-StepCommandWithTimeout {
    param(
        [Parameter(Mandatory = $true)][string]$StepLabel,
        [Parameter(Mandatory = $true)][scriptblock]$Command,
        [Parameter(Mandatory = $true)][int]$TimeoutSeconds
    )
    $pwshPath = (Get-Command pwsh -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Source)
    if ([string]::IsNullOrWhiteSpace($pwshPath)) {
        throw "[quality-gate] unable to locate pwsh executable for step execution"
    }

    $commandText = $Command.ToString()
    $repoRootLiteral = $repoRoot.Path.Replace("'", "''")
    $payload = @"
Set-StrictMode -Version Latest
`$ErrorActionPreference = "Stop"
Set-Location -LiteralPath '$repoRootLiteral'
& {
$commandText
}
`$stepExit = `$LASTEXITCODE
if (`$null -eq `$stepExit) { `$stepExit = 0 }
exit `$stepExit
"@
    $encodedPayload = [Convert]::ToBase64String([System.Text.Encoding]::Unicode.GetBytes($payload))

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $pwshPath
    $startInfo.Arguments = "-NoLogo -NoProfile -EncodedCommand $encodedPayload"
    $startInfo.WorkingDirectory = $repoRoot.Path
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $false
    $startInfo.RedirectStandardError = $false

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo

    try {
        if (-not $process.Start()) {
            throw "[quality-gate] failed to start step process for $StepLabel"
        }
        Register-ActiveProcessId -ProcessId $process.Id

        $finished = $process.WaitForExit($TimeoutSeconds * 1000)
        if (-not $finished) {
            $killed = Stop-ProcessTree -ProcessId $process.Id -Process $process
            if (-not $killed) {
                Write-Warning "[quality-gate] failed to confirm process tree termination for pid=$($process.Id)"
            }
            throw "[quality-gate] step timeout: $StepLabel exceeded ${TimeoutSeconds}s"
        }
        $process.WaitForExit()

        if ($process.ExitCode -ne 0) {
            throw "[quality-gate] command failed: $StepLabel (exit=$($process.ExitCode))"
        }
    }
    finally {
        Unregister-ActiveProcessId -ProcessId $process.Id
        $process.Dispose()
    }
}

$script:QualityGateStartedAt = Get-Date
$script:QualityGateStepIndex = 0
$script:QualityGateStepTimings = New-Object 'System.Collections.Generic.List[object]'
$script:QualityGateActiveProcessIds = New-Object 'System.Collections.Generic.HashSet[int]'
$script:QualityGateTotalTimeoutSeconds = Get-PositiveIntEnvOrDefault -EnvName "BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS" -DefaultValue 900
$script:QualityGateStepTimeoutSeconds = Get-PositiveIntEnvOrDefault -EnvName "BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS" -DefaultValue 600
$script:QualityGateParallelism = Get-PositiveIntEnvOrDefault -EnvName "BAYMAX_QUALITY_GATE_PARALLELISM" -DefaultValue 3
if ($script:QualityGateStepTimeoutSeconds -gt $script:QualityGateTotalTimeoutSeconds) {
    $script:QualityGateStepTimeoutSeconds = $script:QualityGateTotalTimeoutSeconds
}
Write-Host "[quality-gate] timeout budget: total=${script:QualityGateTotalTimeoutSeconds}s step=${script:QualityGateStepTimeoutSeconds}s parallelism=$script:QualityGateParallelism"

function Write-NativeOutputLines {
    param(
        [Parameter(Mandatory = $true)][AllowEmptyCollection()][object[]]$Lines
    )
    foreach ($line in $Lines) {
        if ($null -eq $line) {
            continue
        }
        if ($line -is [System.Management.Automation.ErrorRecord]) {
            Write-Host ($line.ToString())
            continue
        }
        Write-Host ([string]$line)
    }
}

function Add-StepTimingRecord {
    param(
        [Parameter(Mandatory = $true)][int]$StepNumber,
        [Parameter(Mandatory = $true)][string]$StepLabel,
        [Parameter(Mandatory = $true)][datetime]$StartedAt,
        [Parameter(Mandatory = $true)][string]$Status,
        [Parameter(Mandatory = $false)][string]$ErrorText
    )
    $elapsed = (Get-Date) - $StartedAt
    $totalElapsed = (Get-Date) - $script:QualityGateStartedAt
    $durationSeconds = [Math]::Round($elapsed.TotalSeconds, 2)
    $totalElapsedSeconds = [Math]::Round($totalElapsed.TotalSeconds, 2)
    $script:QualityGateStepTimings.Add([PSCustomObject]@{
            step                  = $StepNumber
            label                 = $StepLabel
            status                = $Status
            duration_seconds      = $durationSeconds
            total_elapsed_seconds = $totalElapsedSeconds
            error                 = $ErrorText
        }) | Out-Null
    if ($Status -eq "ok") {
        Write-Host "$StepLabel [done=${durationSeconds}s total=${totalElapsedSeconds}s]"
    }
    else {
        Write-Warning "$StepLabel [failed=${durationSeconds}s total=${totalElapsedSeconds}s reason=$ErrorText]"
    }
}

function Register-ActiveProcessId {
    param(
        [Parameter(Mandatory = $true)][int]$ProcessId
    )
    if ($ProcessId -gt 0) {
        [void]$script:QualityGateActiveProcessIds.Add($ProcessId)
    }
}

function Unregister-ActiveProcessId {
    param(
        [Parameter(Mandatory = $true)][int]$ProcessId
    )
    if ($ProcessId -gt 0) {
        [void]$script:QualityGateActiveProcessIds.Remove($ProcessId)
    }
}

function Stop-ActiveQualityGateProcesses {
    if ($script:QualityGateActiveProcessIds.Count -eq 0) {
        return
    }
    foreach ($pid in @($script:QualityGateActiveProcessIds)) {
        try {
            Stop-ProcessTree -ProcessId $pid
        }
        catch {
            Write-Warning "[quality-gate] failed to stop residual pid=${pid}: $($_.Exception.Message)"
        }
        finally {
            [void]$script:QualityGateActiveProcessIds.Remove($pid)
        }
    }
}

function Write-StepTimingSummary {
    $totalElapsed = (Get-Date) - $script:QualityGateStartedAt
    $completedCount = $script:QualityGateStepTimings.Count
    Write-Host "[quality-gate][timing] steps_completed=$completedCount total_elapsed=$([Math]::Round($totalElapsed.TotalSeconds, 2))s budget=${script:QualityGateTotalTimeoutSeconds}s"
    if ($completedCount -eq 0) {
        return
    }
    Write-Host "[quality-gate][timing] slowest steps:"
    foreach ($entry in ($script:QualityGateStepTimings | Sort-Object -Property duration_seconds -Descending | Select-Object -First 10)) {
        Write-Host ("[quality-gate][timing] step={0} duration={1}s elapsed={2}s status={3} label={4}" -f $entry.step, $entry.duration_seconds, $entry.total_elapsed_seconds, $entry.status, $entry.label)
    }
}

function Invoke-RequiredStep {
    param(
        [Parameter(Mandatory = $true)][string]$StepLabel,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )

    $script:QualityGateStepIndex++
    $stepNumber = $script:QualityGateStepIndex
    $elapsedBefore = (Get-Date) - $script:QualityGateStartedAt
    $remainingSeconds = $script:QualityGateTotalTimeoutSeconds - [Math]::Floor($elapsedBefore.TotalSeconds)
    if ($remainingSeconds -le 0) {
        throw "[quality-gate] total timeout exceeded before step $stepNumber ($StepLabel)"
    }
    $stepTimeout = [Math]::Min($script:QualityGateStepTimeoutSeconds, $remainingSeconds)
    $startedAt = Get-Date
    Write-Host "$StepLabel [step=$stepNumber start=$($startedAt.ToString('yyyy-MM-dd HH:mm:ss')) elapsed=$([Math]::Round($elapsedBefore.TotalSeconds, 2))s remaining=${remainingSeconds}s timeout=${stepTimeout}s]"

    $status = "ok"
    $errorText = $null
    try {
        Invoke-StepCommandWithTimeout -StepLabel $StepLabel -Command $Command -TimeoutSeconds $stepTimeout
    }
    catch {
        $status = "failed"
        $errorText = $_.Exception.Message
        throw
    }
    finally {
        Add-StepTimingRecord -StepNumber $stepNumber -StepLabel $StepLabel -StartedAt $startedAt -Status $status -ErrorText $errorText
    }
}

function Invoke-RequiredParallelSteps {
    param(
        [Parameter(Mandatory = $true)][AllowEmptyCollection()][object[]]$Steps
    )

    if ($null -eq $Steps -or $Steps.Count -eq 0) {
        return
    }
    if ($script:QualityGateParallelism -le 1 -or $Steps.Count -eq 1) {
        foreach ($step in $Steps) {
            Invoke-RequiredStep -StepLabel $step.StepLabel -Command $step.Command
        }
        return
    }

    Write-Host "[quality-gate] parallel batch start: size=$($Steps.Count) max_parallel=$script:QualityGateParallelism"

    $pending = New-Object 'System.Collections.Generic.Queue[object]'
    foreach ($step in $Steps) {
        $pending.Enqueue($step)
    }
    $running = @()

    while ($pending.Count -gt 0 -or $running.Count -gt 0) {
        $elapsedBeforeLaunch = (Get-Date) - $script:QualityGateStartedAt
        $remainingTotal = $script:QualityGateTotalTimeoutSeconds - [Math]::Floor($elapsedBeforeLaunch.TotalSeconds)
        if ($remainingTotal -le 0) {
            foreach ($item in $running) {
                Stop-Job -Job $item.Job -ErrorAction SilentlyContinue
                Remove-Job -Job $item.Job -Force -ErrorAction SilentlyContinue
            }
            throw "[quality-gate] total timeout exceeded while running parallel batch"
        }

        while ($pending.Count -gt 0 -and $running.Count -lt $script:QualityGateParallelism) {
            $next = $pending.Dequeue()
            $script:QualityGateStepIndex++
            $stepNumber = $script:QualityGateStepIndex
            $launchElapsed = (Get-Date) - $script:QualityGateStartedAt
            $launchRemaining = $script:QualityGateTotalTimeoutSeconds - [Math]::Floor($launchElapsed.TotalSeconds)
            if ($launchRemaining -le 0) {
                throw "[quality-gate] total timeout exceeded before step $stepNumber ($($next.StepLabel))"
            }
            $stepTimeout = [Math]::Min($script:QualityGateStepTimeoutSeconds, $launchRemaining)
            $startedAt = Get-Date
            Write-Host "$($next.StepLabel) [step=$stepNumber start=$($startedAt.ToString('yyyy-MM-dd HH:mm:ss')) elapsed=$([Math]::Round($launchElapsed.TotalSeconds, 2))s remaining=${launchRemaining}s timeout=${stepTimeout}s mode=parallel]"

            $commandText = $next.Command.ToString()
            $job = Start-Job -ScriptBlock {
                param($cmdText, $repoRootPath)
                Set-StrictMode -Version Latest
                $ErrorActionPreference = "Stop"
                Set-Location -LiteralPath $repoRootPath
                $stepOutput = @()
                $stepExit = 0
                try {
                    $stepOutput = @(& ([ScriptBlock]::Create($cmdText)) 2>&1)
                    $stepExit = $LASTEXITCODE
                    if ($null -eq $stepExit) {
                        $stepExit = 0
                    }
                }
                catch {
                    $stepOutput += $_
                    $stepExit = 1
                }
                [PSCustomObject]@{
                    ExitCode = [int]$stepExit
                    Output   = $stepOutput
                }
            } -ArgumentList $commandText, $repoRoot.Path

            $running += [PSCustomObject]@{
                StepNumber     = $stepNumber
                StepLabel      = [string]$next.StepLabel
                StartedAt      = $startedAt
                TimeoutSeconds = $stepTimeout
                Job            = $job
            }
        }

        Start-Sleep -Milliseconds 250
        $nextRunning = @()
        foreach ($item in $running) {
            $stepElapsed = (Get-Date) - $item.StartedAt
            $jobState = $item.Job.State
            $timedOut = ($stepElapsed.TotalSeconds -ge $item.TimeoutSeconds) -and ($jobState -eq "Running")
            $jobFinished = $jobState -in @("Completed", "Failed", "Stopped")

            if (-not $timedOut -and -not $jobFinished) {
                $nextRunning += $item
                continue
            }

            if ($timedOut) {
                Stop-Job -Job $item.Job -ErrorAction SilentlyContinue
                $jobState = $item.Job.State
            }

            $result = $null
            try {
                $result = Receive-Job -Job $item.Job -ErrorAction SilentlyContinue
            }
            catch {
                $result = $null
            }
            Remove-Job -Job $item.Job -Force -ErrorAction SilentlyContinue

            $payload = if ($null -eq $result) { $null } else { @($result) | Select-Object -Last 1 }
            $outputLines = @()
            $exitCode = 1
            if ($null -ne $payload) {
                if ($payload.PSObject.Properties.Name -contains "Output") {
                    $outputLines = @($payload.Output)
                }
                if ($payload.PSObject.Properties.Name -contains "ExitCode") {
                    $exitCode = [int]$payload.ExitCode
                }
            }
            Write-NativeOutputLines -Lines $outputLines

            $status = "ok"
            $errorText = $null
            if ($timedOut) {
                $status = "failed"
                $errorText = "[quality-gate] step timeout: $($item.StepLabel) exceeded $($item.TimeoutSeconds)s"
            }
            elseif (Test-GoStatCachePermissionWarning -Output $outputLines) {
                $status = "failed"
                $errorText = "[native-strict] command failed: $($item.StepLabel) (go stat cache write permission warning detected)"
            }
            elseif ($jobState -eq "Failed") {
                $status = "failed"
                $errorText = "[quality-gate] command failed: $($item.StepLabel) (job failed)"
            }
            elseif ($jobState -eq "Stopped" -and $exitCode -eq 0) {
                $status = "failed"
                $errorText = "[quality-gate] command stopped: $($item.StepLabel)"
            }
            elseif ($exitCode -ne 0) {
                $status = "failed"
                $errorText = "[quality-gate] command failed: $($item.StepLabel) (exit=$exitCode)"
            }

            Add-StepTimingRecord -StepNumber $item.StepNumber -StepLabel $item.StepLabel -StartedAt $item.StartedAt -Status $status -ErrorText $errorText

            if ($status -ne "ok") {
                foreach ($other in $nextRunning) {
                    Stop-Job -Job $other.Job -ErrorAction SilentlyContinue
                    Remove-Job -Job $other.Job -Force -ErrorAction SilentlyContinue
                }
                foreach ($other in $running) {
                    if ($other.Job.Id -eq $item.Job.Id) {
                        continue
                    }
                    Stop-Job -Job $other.Job -ErrorAction SilentlyContinue
                    Remove-Job -Job $other.Job -Force -ErrorAction SilentlyContinue
                }
                throw $errorText
            }
        }
        $running = $nextRunning
    }
}

try {
    $qualityGateScope = if ($env:BAYMAX_QUALITY_GATE_SCOPE) { $env:BAYMAX_QUALITY_GATE_SCOPE.Trim().ToLowerInvariant() } else { "full" }
    if ($qualityGateScope -ne "full" -and $qualityGateScope -ne "general") {
        throw "[quality-gate] BAYMAX_QUALITY_GATE_SCOPE must be one of: full, general"
    }
    Write-Host "[quality-gate] scope=$qualityGateScope"

    if ($qualityGateScope -eq "general") {
        Invoke-RequiredStep -StepLabel "[quality-gate] repo hygiene" -Command {
            pwsh -File scripts/check-repo-hygiene.ps1
        }

        Invoke-RequiredStep -StepLabel "[quality-gate] docs consistency" -Command {
            pwsh -File scripts/check-docs-consistency.ps1
        }

        Invoke-RequiredStep -StepLabel "[quality-gate] go test ./..." -Command {
            go test ./...
        }

        $generalCgoEnabled = ((Invoke-NativeCaptureStrict -Label "go env CGO_ENABLED" -Command {
                go env CGO_ENABLED
            }) | Select-Object -First 1).Trim()
        if ($generalCgoEnabled -ne "1") {
            throw "[quality-gate] go test -race requires CGO_ENABLED=1"
        }

        Invoke-RequiredStep -StepLabel "[quality-gate] go test -race ./..." -Command {
            go test -race ./...
        }

        Invoke-RequiredStep -StepLabel "[quality-gate] golangci-lint --config .golangci.yml" -Command {
            golangci-lint run --config .golangci.yml
        }

        Write-Host "[quality-gate] done (scope=general)"
        return
    }

Invoke-RequiredStep -StepLabel "[quality-gate] repo hygiene" -Command {
    pwsh -File scripts/check-repo-hygiene.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] docs consistency" -Command {
    pwsh -File scripts/check-docs-consistency.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] canonical mailbox entrypoints" -Command {
    pwsh -File scripts/check-canonical-mailbox-entrypoints.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] multi-agent shared contract suites" -Command {
    pwsh -File scripts/check-multi-agent-shared-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] state snapshot contract suites" -Command {
    pwsh -File scripts/check-state-snapshot-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] runtime readiness + explainability + version governance contract suites" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event ./orchestration/composer ./integration -run 'Test(RuntimeReadiness|ReadinessAdmission|ArbitrationVersionGovernanceContract|StoreRunReadiness|StoreRunArbitrationVersionGovernance|RuntimeRecorderA40ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA49ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA50ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderParsesA50ArbitrationVersionGovernanceFields|ComposerReadiness)' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] diagnostics cardinality contract suites" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(DiagnosticsCardinality|ManagerDiagnosticsCardinality|StoreRunCardinality|CardinalityListGovernance|RuntimeRecorderA45ParserCompatibilityAdditiveNullableDefault|DiagnosticsCardinalityContract)' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter-health contract suites" -Command {
    go test ./adapter/health ./runtime/config ./runtime/diagnostics ./observability/event ./integration/adapterconformance -run 'Test(RunnerProbe|AdapterHealthConfig|ManagerAdapterHealth|ManagerReadinessPreflightAdapterHealth|StoreRunReadinessAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderA14ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA46ParserCompatibilityAdditiveNullableDefault|AdapterConformanceHealth(Matrix|Governance))' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] mailbox runtime wiring regression" -Command {
    go test ./integration -run '^TestComposerContractMailboxRuntimeWiring' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] timeout resolution contract suites" -Command {
    go test ./integration -run '^TestTimeoutResolutionContract' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] readiness-timeout-health replay fixture suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReplayContractPrimaryReasonArbitrationFixture|ReadinessTimeoutHealthReplayContract|PrimaryReasonArbitrationReplayContract)' -count=1
}

Invoke-RequiredParallelSteps -Steps @(
    @{
        StepLabel = "[quality-gate] react contract suites"
        Command   = { pwsh -File scripts/check-react-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] react plan notebook contract suites"
        Command   = { pwsh -File scripts/check-react-plan-notebook-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] hooks + middleware contract suites"
        Command   = { pwsh -File scripts/check-hooks-middleware-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] security sandbox contract suites"
        Command   = { pwsh -File scripts/check-security-sandbox-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] sandbox rollout governance contract suites"
        Command   = { pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] sandbox egress + adapter allowlist contract suites"
        Command   = { pwsh -File scripts/check-sandbox-egress-allowlist-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] policy precedence contract suites"
        Command   = { pwsh -File scripts/check-policy-precedence-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] runtime budget admission contract suites"
        Command   = { pwsh -File scripts/check-runtime-budget-admission-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] agent eval and tracing interop contract suites"
        Command   = { pwsh -File scripts/check-agent-eval-and-tracing-interop-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] adapter conformance"
        Command   = { pwsh -File scripts/check-adapter-conformance.ps1 }
    },
    @{
        StepLabel = "[quality-gate] adapter manifest contract"
        Command   = { pwsh -File scripts/check-adapter-manifest-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] adapter capability negotiation contract"
        Command   = { pwsh -File scripts/check-adapter-capability-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] adapter contract replay"
        Command   = { pwsh -File scripts/check-adapter-contract-replay.ps1 }
    },
    @{
        StepLabel = "[quality-gate] sandbox adapter conformance contract"
        Command   = { pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] memory contract conformance"
        Command   = { pwsh -File scripts/check-memory-contract-conformance.ps1 }
    },
    @{
        StepLabel = "[quality-gate] memory scope and search governance contract"
        Command   = { pwsh -File scripts/check-memory-scope-and-search-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] observability export and diagnostics bundle contract"
        Command   = { pwsh -File scripts/check-observability-export-and-bundle-contract.ps1 }
    },
    @{
        StepLabel = "[quality-gate] adapter scaffold drift"
        Command   = { pwsh -File scripts/check-adapter-scaffold-drift.ps1 }
    }
)

Invoke-RequiredStep -StepLabel "[quality-gate] go test ./..." -Command {
    go test ./...
}

$cgoEnabled = ((Invoke-NativeCaptureStrict -Label "go env CGO_ENABLED" -Command {
            go env CGO_ENABLED
        }) | Select-Object -First 1).Trim()
if ($cgoEnabled -ne "1") {
    throw "[quality-gate] go test -race requires CGO_ENABLED=1"
}

Invoke-RequiredStep -StepLabel "[quality-gate] go test -race ./..." -Command {
    go test -race ./...
}

Invoke-RequiredStep -StepLabel "[quality-gate] golangci-lint --config .golangci.yml" -Command {
    golangci-lint run --config .golangci.yml
}

Invoke-RequiredStep -StepLabel "[quality-gate] CA4 benchmark regression" -Command {
    pwsh -File scripts/check-ca4-benchmark-regression.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] multi-agent mainline benchmark regression" -Command {
    pwsh -File scripts/check-multi-agent-performance-regression.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] diagnostics query benchmark regression" -Command {
    pwsh -File scripts/check-diagnostics-query-performance-regression.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] full-chain example smoke" -Command {
    pwsh -File scripts/check-full-chain-example-smoke.ps1
}

$scanMode = if ($env:BAYMAX_SECURITY_SCAN_MODE) { $env:BAYMAX_SECURITY_SCAN_MODE.Trim().ToLowerInvariant() } else { "strict" }
$govulncheckEnabled = if ($env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED) { $env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED.Trim().ToLowerInvariant() } else { "true" }
if ($govulncheckEnabled -eq "true") {
    Write-Host "[quality-gate] govulncheck (mode=$scanMode)"

    $proxyVars = @("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy", "GIT_HTTP_PROXY", "GIT_HTTPS_PROXY")
    $savedProxy = @{}
    $needsDirect = $false
    foreach ($name in $proxyVars) {
        $value = [Environment]::GetEnvironmentVariable($name)
        if ([string]::IsNullOrWhiteSpace($value)) {
            continue
        }
        $savedProxy[$name] = $value
        $lower = $value.Trim().ToLowerInvariant()
        if ($lower.Contains("127.0.0.1:9") -or $lower.Contains("localhost:9") -or $lower.Contains("[::1]:9")) {
            $needsDirect = $true
        }
    }
    if ($needsDirect) {
        Write-Host "[quality-gate] detected placeholder proxy for govulncheck; run with direct connection"
        foreach ($name in $proxyVars) {
            Remove-Item -Path ("Env:" + $name) -ErrorAction SilentlyContinue
        }
    }

    # The only governance exception path: warn mode allows vulnerability findings without unblocking other strict checks.
    $allowWarn = ($scanMode -eq "warn")
    try {
        if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
            $exitCode = Invoke-NativeStrict -Label "govulncheck ./..." -AllowFailure:$allowWarn -Command {
                govulncheck ./...
            }
        }
        else {
            $exitCode = Invoke-NativeStrict -Label "go run golang.org/x/vuln/cmd/govulncheck@latest ./..." -AllowFailure:$allowWarn -Command {
                go run golang.org/x/vuln/cmd/govulncheck@latest ./...
            }
        }

        if ($allowWarn -and $exitCode -ne 0) {
            Write-Warning "[quality-gate] govulncheck found issues but mode=warn; continue"
        }
    }
    finally {
        if ($needsDirect) {
            foreach ($name in $savedProxy.Keys) {
                Set-Item -Path ("Env:" + $name) -Value $savedProxy[$name]
            }
        }
    }
}
else {
    Write-Host "[quality-gate] govulncheck disabled by BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED"
}

    Write-Host "[quality-gate] done"
}
finally {
    Write-StepTimingSummary
    Stop-ActiveQualityGateProcesses
}
