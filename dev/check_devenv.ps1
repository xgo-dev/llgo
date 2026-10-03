$ErrorActionPreference = 'Stop'

if ($env:GOFLAGS -ne '-tags=byollvm') { throw 'GOFLAGS must enable byollvm' }
if (-not ((go env GOVERSION) -match '^go1\.27\.')) { throw 'Go 1.27 is required' }
if (-not ((llvm-config --version) -match '^22\.')) { throw 'LLVM 22 is required' }
if (-not (Test-Path (Join-Path $env:LLGO_ROOT 'runtime/go.mod'))) { throw 'LLGO_ROOT is incorrect' }

Set-Location $env:LLGO_ROOT
go build -o llgo.exe ./cmd/llgo
if ($LASTEXITCODE -ne 0) { throw 'LLGo build failed' }
& .\llgo.exe version
if ($LASTEXITCODE -ne 0) { throw 'LLGo version failed' }

$smokeDir = Join-Path $env:TEMP ("llgo-pixi-" + [guid]::NewGuid().ToString('N'))
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
} finally {
  Remove-Item -Recurse -Force $smokeDir
}
