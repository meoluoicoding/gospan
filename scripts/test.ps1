<#
.SYNOPSIS
    Run the full Gospan validation suite.
#>
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

Write-Host "==> gofmt" -ForegroundColor Cyan
$unformatted = gofmt -l .
if ($unformatted) {
    Write-Host "Unformatted files:" -ForegroundColor Red
    $unformatted | ForEach-Object { Write-Host "  $_" }
    exit 1
}

Write-Host "==> go vet" -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "==> go test (race)" -ForegroundColor Cyan
go test -race ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "==> benchmarks" -ForegroundColor Cyan
go test "-bench=." "-benchmem" "./internal/engine/..."
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "All checks passed." -ForegroundColor Green
