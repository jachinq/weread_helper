param(
  [switch]$SkipWeb
)

$ErrorActionPreference = "Stop"

$sys32 = Join-Path $env:WINDIR "System32"
if ($env:Path -notlike "*$sys32*") {
  $env:Path = "$sys32;" + $env:Path
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  $goGuess = Join-Path ${env:ProgramFiles} "Go\bin"
  if (Test-Path (Join-Path $goGuess "go.exe")) {
    $env:Path = "$goGuess;" + $env:Path
  }
}

$DesktopRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$RepoRoot = Resolve-Path (Join-Path $DesktopRoot "..")
$SrcTauri = Join-Path $DesktopRoot "src-tauri"
$BinDir = Join-Path $SrcTauri "binaries"
$WebDest = Join-Path $SrcTauri "resources\web"

if (-not $SkipWeb) {
  Push-Location (Join-Path $RepoRoot "web")
  try {
    pnpm run build
  } finally {
    Pop-Location
  }
}

New-Item -ItemType Directory -Force -Path (Join-Path $SrcTauri "resources") | Out-Null
if (Test-Path $WebDest) {
  Remove-Item -Recurse -Force $WebDest
}
Copy-Item -Recurse (Join-Path $RepoRoot "web\dist") $WebDest

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

$triple = $null
try {
  $triple = (rustc --print host-tuple 2>$null | Out-String).Trim()
} catch {}
if (-not $triple) {
  $triple = (rustc --print host-target | Out-String).Trim()
}
if (-not $triple) {
  throw "无法通过 rustc 获取 target triple"
}

$out = Join-Path $BinDir "weread-helper-$triple.exe"
$env:CGO_ENABLED = "0"
Push-Location (Join-Path $RepoRoot "server")
try {
  go build -trimpath -ldflags="-s -w" -o $out ./cmd/api
} finally {
  Pop-Location
}

Write-Host "sidecar -> $out"
Write-Host "web -> $WebDest"
