param(
  [switch]$SkipWeb,
  [switch]$SkipTauri
)

$ErrorActionPreference = "Stop"

function Add-DirToPath([string]$dir) {
  if ($dir -and (Test-Path $dir) -and ($env:Path -notlike "*$dir*")) {
    $env:Path = "$dir;" + $env:Path
  }
}

$sys32 = Join-Path $env:WINDIR "System32"
Add-DirToPath $sys32
Add-DirToPath (Join-Path ${env:ProgramFiles} "Go\bin")
Add-DirToPath (Join-Path $env:USERPROFILE ".cargo\bin")
Add-DirToPath (Join-Path ${env:ProgramFiles} "nodejs")

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "未找到 go，请先安装 Go 并加入 PATH"
}
if (-not (Get-Command rustc -ErrorAction SilentlyContinue)) {
  throw "未找到 rustc，请先安装 Rust"
}
if (-not (Get-Command pnpm -ErrorAction SilentlyContinue)) {
  throw "未找到 pnpm，请在 desktop/ 执行 pnpm install"
}

$DesktopRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$RepoRoot = Resolve-Path (Join-Path $DesktopRoot "..")
$SrcTauri = Join-Path $DesktopRoot "src-tauri"
$ConfPath = Join-Path $SrcTauri "tauri.conf.json"
$Conf = Get-Content -Raw -Encoding UTF8 $ConfPath | ConvertFrom-Json
$ProductName = [string]$Conf.productName
$Version = [string]$Conf.version

$triple = $null
try {
  $triple = (rustc --print host-tuple 2>$null | Out-String).Trim()
} catch {}
if (-not $triple) {
  $triple = (rustc --print host-target | Out-String).Trim()
}
$Arch = "x64"
if ($triple -match "aarch64") { $Arch = "arm64" }
elseif ($triple -match "i686") { $Arch = "x86" }

$DistRoot = Join-Path $DesktopRoot "dist"
$Stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$PortableName = "weread-helper-$Version-windows-$Arch-portable"
$SetupName = "weread-helper-$Version-windows-$Arch-setup.exe"
$PortableDir = Join-Path $DistRoot $PortableName
$ZipPath = Join-Path $DistRoot "$PortableName.zip"
$SetupDest = Join-Path $DistRoot $SetupName

Write-Host "==> 微信读书助手桌面发布  v$Version  ($triple)"
Write-Host "    产物目录: $DistRoot"

if (-not $SkipTauri) {
  $sidecar = Join-Path $PSScriptRoot "build-sidecar.ps1"
  if ($SkipWeb) {
    & $sidecar -SkipWeb
  } else {
    & $sidecar
  }

  Push-Location $DesktopRoot
  try {
    pnpm exec tauri build
  } finally {
    Pop-Location
  }
}

$ReleaseDir = Join-Path $SrcTauri "target\release"
if (-not (Test-Path $ReleaseDir)) {
  throw "未找到 $ReleaseDir，请先完成 tauri build"
}

$MainExe = $null
Get-ChildItem $ReleaseDir -File -Filter "*.exe" | ForEach-Object {
  if (-not $MainExe -and ($_.Name -eq "$ProductName.exe" -or $_.Name -eq "weread-helper-desktop.exe")) {
    $MainExe = $_
  }
}
if (-not $MainExe) {
  throw "未找到主程序 exe（期望 $ProductName.exe）"
}

$SidecarSrc = $null
$sidecarGuess = @(
  (Join-Path $ReleaseDir "weread-helper.exe"),
  (Join-Path $ReleaseDir ("weread-helper-" + $triple + ".exe")),
  (Join-Path $SrcTauri ("binaries\weread-helper-" + $triple + ".exe"))
)
foreach ($p in $sidecarGuess) {
  if (Test-Path -LiteralPath $p) {
    $SidecarSrc = $p
    break
  }
}
if (-not $SidecarSrc) {
  throw "未找到 sidecar weread-helper.exe"
}

$WebSrc = $null
$webGuess = @(
  (Join-Path $SrcTauri "resources\web"),
  (Join-Path $RepoRoot "web\dist")
)
foreach ($p in $webGuess) {
  if (Test-Path -LiteralPath (Join-Path $p "index.html")) {
    $WebSrc = $p
    break
  }
}
if (-not $WebSrc) {
  throw "未找到前端静态资源（resources/web 或 web/dist）"
}

if (Test-Path $DistRoot) {
  Remove-Item -Recurse -Force $DistRoot
}
New-Item -ItemType Directory -Force -Path $PortableDir | Out-Null

Copy-Item $MainExe.FullName (Join-Path $PortableDir ($ProductName + ".exe"))
Copy-Item $SidecarSrc (Join-Path $PortableDir "weread-helper.exe")
Copy-Item -Recurse $WebSrc (Join-Path $PortableDir "web")
Copy-Item -Recurse $WebSrc (Join-Path $PortableDir "resources\web")

Get-ChildItem $ReleaseDir -File -Filter "*.dll" -ErrorAction SilentlyContinue | ForEach-Object {
  Copy-Item $_.FullName $PortableDir
}

$Readme = @"
微信读书助手 $Version 便携版
解压后直接运行「$ProductName.exe」。
数据（SQLite、密钥、WebView 缓存）写在本目录 data/，可整夹拷走。
需要本机已安装 Microsoft Edge WebView2 Runtime。
请勿把 data/ 放进压缩包再分发（含个人笔记与 API Key）。
构建时间：$Stamp
"@
Set-Content -Path (Join-Path $PortableDir "使用说明.txt") -Value $Readme -Encoding UTF8

if (Test-Path $ZipPath) { Remove-Item -Force $ZipPath }
Compress-Archive -Path $PortableDir -DestinationPath $ZipPath -CompressionLevel Optimal

$NsisDir = Join-Path $ReleaseDir "bundle\nsis"
$NsisSetup = $null
if (Test-Path $NsisDir) {
  $NsisSetup = Get-ChildItem $NsisDir -Recurse -File -Filter "*.exe" |
    Where-Object { $_.Name -match "setup" -or $_.Extension -eq ".exe" } |
    Sort-Object Length -Descending |
    Select-Object -First 1
}
if (-not $NsisSetup) {
  throw "未找到 NSIS 安装包（$NsisDir）。请确认 tauri.conf.json bundle.targets 包含 nsis。"
}
Copy-Item $NsisSetup.FullName $SetupDest

Write-Host ""
Write-Host "完成。"
Write-Host "  便携压缩包: $ZipPath"
Write-Host "  安装程序:   $SetupDest"
Write-Host "  便携目录:   $PortableDir"
Write-Host "  原始 NSIS:  $($NsisSetup.FullName)"