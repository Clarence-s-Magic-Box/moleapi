$ErrorActionPreference = "Stop"

Write-Host "[1/3] Build web (bun)" -ForegroundColor Cyan
Push-Location (Join-Path $PSScriptRoot "..\\web")
bun install
bun run build
Pop-Location

Write-Host "[2/3] Run backend tests" -ForegroundColor Cyan
Push-Location (Join-Path $PSScriptRoot "..")
go test ./...

Write-Host "[3/3] Build backend binary with VERSION" -ForegroundColor Cyan
$ver = (Get-Content (Join-Path $PSScriptRoot "..\\VERSION") -Raw).Trim()
$commit = "unknown"
try {
  $commit = (git rev-parse --short HEAD).Trim()
} catch {}
go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$ver' -X 'github.com/QuantumNous/new-api/common.Commit=$commit'" -o new-api.exe .
Write-Host ("Built new-api.exe version: " + (& .\\new-api.exe --version)) -ForegroundColor Green

Write-Host ""
Write-Host "Run: .\\new-api.exe" -ForegroundColor Yellow
Write-Host "Default SQLite path: one-api.db in current working directory (set SQLITE_PATH to change)." -ForegroundColor Yellow
Pop-Location

