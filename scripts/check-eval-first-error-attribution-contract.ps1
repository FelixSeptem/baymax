$ErrorActionPreference = "Stop"
$env:GOCACHE = if ($env:BAYMAX_FIRST_ERROR_GOCACHE) { $env:BAYMAX_FIRST_ERROR_GOCACHE } else { Join-Path (Get-Location) ".gocache-eval-first-error" }
go test ./runtime/evalcontract ./tool/diagnosticsreplay -run 'Test(NormalizeFirstErrorAttribution|CompareFirstErrorAttribution|FirstErrorAttributionFixture)' -count=1
if ($LASTEXITCODE -ne 0) { throw "first-error attribution contract tests failed" }
if (rg -n 'os/exec|os/Chdir|net/http|git |workspace mutation|raw_reasoning|transcript_body|provider_response|tool_output|memory_body|workspace_body|credential' runtime/evalcontract/first_error_attribution.go tool/diagnosticsreplay/first_error_attribution.go) { throw "first-error attribution contains forbidden side effects or body persistence" }
Write-Host "[eval-first-error-attribution-contract] passed"
