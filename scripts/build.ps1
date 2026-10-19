<#
.SYNOPSIS
    Build a release binary for Gospan.
.EXAMPLE
    ./scripts/build.ps1 -Version 1.0.0
#>
param(
    [string]$Version = "dev",
    [string]$Output  = "dist/gospan.exe"
)

$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

Write-Host "==> Running tests" -ForegroundColor Cyan
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "==> Running go vet" -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "==> Building $Output" -ForegroundColor Cyan
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

New-Item -ItemType Directory -Force -Path (Split-Path $Output) | Out-Null
go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $Output ./cmd/gospan
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$hash = (Get-FileHash $Output -Algorithm SHA256).Hash
Write-Host "==> Built $Output" -ForegroundColor Green
Write-Host "    SHA-256: $hash"
