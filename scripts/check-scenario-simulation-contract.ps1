$ErrorActionPreference = "Stop"
$env:GOCACHE = Join-Path (Get-Location).Path ".gocache-scenario-contract"

Write-Host "[scenario-simulation] offline contract tests"
go test ./integration/scenariosimulation -count=1
if ($LASTEXITCODE -ne 0) { throw "[scenario-simulation] contract tests failed" }

$forbidden = Select-String -Path "integration/scenariosimulation/*.go" -Pattern '"github.com/FelixSeptem/baymax/(runtime|model|context)/'
if ($forbidden) { throw "[scenario-simulation] forbidden production dependency detected: $($forbidden -join ', ')" }

$productionImports = Select-String -Path "runtime/**/*.go", "context/**/*.go", "model/**/*.go", "core/**/*.go", "tool/**/*.go" -Pattern 'github.com/FelixSeptem/baymax/integration/scenariosimulation'
if ($productionImports) { throw "[scenario-simulation] production package imports test-support detected: $($productionImports -join ', ')" }

Write-Host "[scenario-simulation] contract gate passed"
