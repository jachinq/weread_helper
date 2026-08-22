param(
  [Parameter(Position = 0)]
  [ValidateSet("patch", "minor", "major", "xiao", "zhong", "da")]
  [string]$Part = "patch",

  [string]$Version,

  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$DesktopRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$TauriConf = Join-Path $DesktopRoot "src-tauri\tauri.conf.json"
$CargoToml = Join-Path $DesktopRoot "src-tauri\Cargo.toml"
$PkgJson = Join-Path $DesktopRoot "package.json"

function Get-SemVer([string]$raw) {
  if ($raw -notmatch '^\s*(\d+)\.(\d+)\.(\d+)\s*$') {
    throw "version must be x.y.z, got: $raw"
  }
  return [pscustomobject]@{
    Major = [int]$Matches[1]
    Minor = [int]$Matches[2]
    Patch = [int]$Matches[3]
    Text  = "$($Matches[1]).$($Matches[2]).$($Matches[3])"
  }
}

function Read-JsonVersion([string]$path) {
  $json = Get-Content -Raw -Encoding UTF8 $path | ConvertFrom-Json
  return [string]$json.version
}

function Read-CargoVersion([string]$path) {
  $text = Get-Content -Raw -Encoding UTF8 $path
  if ($text -notmatch '(?m)^version\s*=\s*"([^"]+)"') {
    throw "cannot read version from $path"
  }
  return $Matches[1]
}

function Set-TextVersion([string]$path, [string]$pattern, [string]$replacement) {
  $text = Get-Content -Raw -Encoding UTF8 $path
  $updated = [regex]::Replace($text, $pattern, $replacement, 1)
  if ($updated -eq $text) {
    throw "failed to update version in $path"
  }
  if (-not $DryRun) {
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($path, $updated, $utf8NoBom)
  }
}

$fromTauri = Get-SemVer (Read-JsonVersion $TauriConf)
$fromCargo = Get-SemVer (Read-CargoVersion $CargoToml)
$fromPkg = Get-SemVer (Read-JsonVersion $PkgJson)

if ($fromCargo.Text -ne $fromTauri.Text -or $fromPkg.Text -ne $fromTauri.Text) {
  Write-Warning "version mismatch; using tauri.conf.json ($($fromTauri.Text))"
  Write-Warning "  tauri.conf.json: $($fromTauri.Text)"
  Write-Warning "  Cargo.toml:      $($fromCargo.Text)"
  Write-Warning "  package.json:    $($fromPkg.Text)"
}

$current = $fromTauri
if ($Version) {
  $next = Get-SemVer $Version
} else {
  $kind = switch ($Part) {
    { $_ -in @("patch", "xiao") } { "patch" }
    { $_ -in @("minor", "zhong") } { "minor" }
    { $_ -in @("major", "da") } { "major" }
  }
  $next = switch ($kind) {
    "major" { Get-SemVer "$($current.Major + 1).0.0" }
    "minor" { Get-SemVer "$($current.Major).$($current.Minor + 1).0" }
    default { Get-SemVer "$($current.Major).$($current.Minor).$($current.Patch + 1)" }
  }
}

if ($next.Text -eq $current.Text) {
  Write-Host "unchanged: $($current.Text)"
  exit 0
}

$preview = ""
if ($DryRun) { $preview = "  (dry-run)" }
Write-Host "version: $($current.Text) -> $($next.Text)$preview"

Set-TextVersion $TauriConf '(?m)^(\s*"version"\s*:\s*")[^"]+(")' "`${1}$($next.Text)`${2}"
Set-TextVersion $CargoToml '(?m)^(version\s*=\s*")[^"]+(")' "`${1}$($next.Text)`${2}"
Set-TextVersion $PkgJson '(?m)^(\s*"version"\s*:\s*")[^"]+(")' "`${1}$($next.Text)`${2}"

Write-Host "updated:"
Write-Host "  desktop/src-tauri/tauri.conf.json"
Write-Host "  desktop/src-tauri/Cargo.toml"
Write-Host "  desktop/package.json"