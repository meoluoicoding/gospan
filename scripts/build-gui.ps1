<#
.SYNOPSIS
    Build the Fyne desktop GUI for Gospan.

.DESCRIPTION
    The GUI lives in its own module (gui/fyne) and requires CGO plus a C
    toolchain (gcc). The core engine remains CGO-free and is built separately by
    scripts/build.ps1.

    By default the binary is built as a Windows GUI application (-H=windowsgui)
    so it opens without a console window. Pass -Console to keep the console for
    debugging.

.EXAMPLE
    ./scripts/build-gui.ps1
    ./scripts/build-gui.ps1 -Console
#>
param(
    [string]$Output = "dist/gospan-gui.exe",
    [switch]$Console
)

$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")
$root = (Get-Location).Path

if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    Write-Host "Fyne requires a C toolchain (gcc)." -ForegroundColor Red
    Write-Host "Install MinGW-w64, for example:  scoop install mingw"
    exit 1
}

$ldflags = "-s -w"
if (-not $Console) {
    $ldflags += " -H=windowsgui"
}

Write-Host "==> Building GUI (CGO enabled) -> $Output" -ForegroundColor Cyan
$env:CGO_ENABLED = "1"

Push-Location "gui\fyne"
try {
    New-Item -ItemType Directory -Force -Path (Join-Path $root "dist") | Out-Null
    go build -trimpath -ldflags "$ldflags" -o (Join-Path $root $Output) .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} finally {
    Pop-Location
}

$exe = Join-Path $root $Output
$hash = (Get-FileHash $exe -Algorithm SHA256).Hash
Write-Host "==> Built $exe" -ForegroundColor Green
Write-Host "    SHA-256: $hash"
if (-not $Console) {
    Write-Host "    (no console window; pass -Console for a debug build)"
}
