$ErrorActionPreference = 'Stop'

if (-not $env:LLGO_ROOT) {
  throw "Enter the development environment with 'mise -C dev en' or 'pixi shell --manifest-path dev/pixi.toml'."
}
if ($env:GOFLAGS -ne '-tags=byollvm') { throw 'GOFLAGS must enable byollvm' }
$goVersion = (go env GOVERSION) -replace '\.\d+$', ''
$requiredGoVersion = ('go' + (Get-Content (Join-Path $env:LLGO_ROOT '.go-version')).Trim()) -replace '\.\d+$', ''
if ($goVersion -ne $requiredGoVersion) { throw 'Go must match the major/minor version in .go-version' }
if (-not ((llvm-config --version) -match '^22\.')) { throw 'LLVM 22 is required' }
if (-not (Test-Path (Join-Path $env:LLGO_ROOT 'runtime/go.mod'))) { throw 'LLGO_ROOT is incorrect' }

Set-Location $env:LLGO_ROOT
go run ./cmd/llgo version
if ($LASTEXITCODE -ne 0) { throw 'LLGo go run failed' }
go build -o llgo.exe ./cmd/llgo
if ($LASTEXITCODE -ne 0) { throw 'LLGo build failed' }
& .\llgo.exe version
if ($LASTEXITCODE -ne 0) { throw 'LLGo version failed' }

$smokeDir = Join-Path $env:TEMP ("llgo-devenv-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $smokeDir | Out-Null
try {
  @'
package main

import "fmt"

func main() { fmt.Println("LLGo dev shell works") }
'@ | Set-Content -Encoding utf8 (Join-Path $smokeDir 'main.go')
  $output = & .\llgo.exe run (Join-Path $smokeDir 'main.go')
  if ($LASTEXITCODE -ne 0 -or $output -ne 'LLGo dev shell works') {
    throw 'LLGo smoke program failed'
  }
  $output = & .\llgo.exe run ./dev/_devenv_smoke
  if ($LASTEXITCODE -ne 0 -or $output -ne 'LLGo native dependencies work') {
    throw 'LLGo native dependency check failed'
  }
  Write-Host 'LLGo development environment checks passed'
} finally {
  Remove-Item -Recurse -Force $smokeDir
}
