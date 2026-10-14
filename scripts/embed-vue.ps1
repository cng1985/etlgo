# 把 frontend 构建到 web/dist，供 web 包的 //go:embed 打进 Go 二进制。
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Fail([string]$Message) {
  [Console]::Error.WriteLine($Message)
  exit 1
}

$root = Split-Path -Parent $PSScriptRoot
$frontend = Join-Path $root 'frontend'
$dist = Join-Path $root 'web\dist'

$npm = Get-Command npm.cmd -ErrorAction SilentlyContinue
if (-not $npm) {
  $npm = Get-Command npm -ErrorAction SilentlyContinue
}
if (-not $npm) {
  Fail '需要 Node.js 22+ 和 npm，才能生成要嵌入的 Vue 页面。'
}

$nodeModules = Join-Path $frontend 'node_modules'
if (-not (Test-Path -LiteralPath $nodeModules)) {
  & $npm.Source ci --prefix $frontend
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

& $npm.Source --prefix $frontend run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$index = Join-Path $dist 'index.html'
if (-not (Test-Path -LiteralPath $index)) {
  Fail "未生成 $index"
}

$content = Get-Content -LiteralPath $index -Raw -Encoding UTF8
if ($content -notmatch 'id="app"') {
  Fail 'web/dist/index.html 里没有 Vue 挂载点 #app'
}

$assets = Join-Path $dist 'assets'
$assetFiles = @()
if (Test-Path -LiteralPath $assets) {
  $assetFiles = @(Get-ChildItem -LiteralPath $assets -File)
}
if ($assetFiles.Count -eq 0) {
  Fail 'web/dist/assets 为空，Vue 构建没有产出可嵌入的脚本。'
}

Write-Host 'Vue 页面已写入 web/dist。执行 go build 或 make build 会把它嵌进二进制。'
