param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('msvc', 'mingw')][string]$Profile,
  [Parameter(Mandatory = $true)]
  [ValidateSet('amd64', 'arm64')][string]$GoArch,
  [switch]$CheckOnly
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-lib.ps1')
$archives = @(Get-ChildItem '.windows-dist' -Filter "*.windows-$GoArch-$Profile.tar.gz")
if ($archives.Count -ne 1) { throw "Expected exactly one windows/$GoArch/$Profile archive" }
$archive = $archives[0]
$checksumLine = (Get-Content -Raw ($archive.FullName + '.sha256')).Trim()
$actual = (Get-FileHash -Algorithm SHA256 $archive.FullName).Hash.ToLowerInvariant()
if ($checksumLine -ne "$actual  $($archive.Name)") { throw 'The Windows archive checksum does not match' }

# A different location, including a space, makes build-tree dependencies and
# accidentally unquoted paths observable before an archive can be published.
$releaseRoot = Join-Path $env:RUNNER_TEMP "extracted llgo-$GoArch-$Profile"
if (Test-Path $releaseRoot) { Remove-Item -LiteralPath $releaseRoot -Recurse -Force }
New-Item -ItemType Directory $releaseRoot | Out-Null
& tar.exe -xzf $archive.FullName -C $releaseRoot
if ($LASTEXITCODE -ne 0) { throw 'Extracting the integrated Windows archive failed' }
foreach ($entry in @('bin/llgo.exe', 'runtime/go.mod', 'targets', 'LICENSES', 'THIRD_PARTY_NOTICES.md',
    'crosscompile/clang/bin/clang++.exe', 'crosscompile/clang/bin/llvm-readobj.exe',
    'crosscompile/clang/THIRD-PARTY-LICENSES.txt', 'crosscompile/clang/LICENSE-LLVM.txt', 'release.json')) {
  if (-not (Test-Path (Join-Path $releaseRoot $entry))) { throw "Release archive is missing $entry" }
}
$metadata = Get-Content -Raw (Join-Path $releaseRoot 'release.json') | ConvertFrom-Json
if ($metadata.goos -ne 'windows' -or $metadata.goarch -ne $GoArch -or $metadata.abi -ne $Profile -or
    $archive.Name -ne "llgo$($metadata.version).windows-$GoArch-$Profile.tar.gz") {
  throw 'The archive name, host architecture, and ABI metadata disagree'
}
$commit = (& git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $metadata.commit -ne $commit) { throw 'The release belongs to a different commit' }

$llgo = Join-Path $releaseRoot 'bin/llgo.exe'
$readObj = Join-Path $releaseRoot 'crosscompile/clang/bin/llvm-readobj.exe'
Copy-ReleaseDLLs -ReadObj $readObj -Executable $llgo -GoArch $GoArch -Profile $Profile -CheckOnly
$go = (Get-Command go).Source
$buildInfo = Invoke-ReleaseCapture $go @('version', '-m', $llgo)
foreach ($setting in @("GOOS=windows", "GOARCH=$GoArch", "CGO_ENABLED=1", "vcs.revision=$commit", 'vcs.modified=false')) {
  if (-not $buildInfo.Contains($setting)) { throw "Release Go build information is missing $setting" }
}

$savedPath = $env:PATH
try {
  Remove-Item Env:LLGO_ROOT -ErrorAction SilentlyContinue
  $env:PATH = (Join-Path $releaseRoot 'bin') + ';' + (Join-Path $env:SystemRoot 'System32') + ';' + $env:SystemRoot
  $version = (Invoke-ReleaseCapture $llgo @('version')).Trim()
  if ($version -ne "llgo v$($metadata.version) windows/$GoArch") {
    throw "Unexpected release compiler version: $version"
  }
} finally {
  $env:PATH = $savedPath
}
Write-Host "Validated standalone windows/$GoArch/$Profile compiler and DLL closure"
if ($CheckOnly) { return }

# Native SDK/CRT and gc/libffi/etc. remain host dependencies, as for the Unix
# integrated releases. setup-deps and activate-windows-target select that ABI.
# LLGO_ROOT stays unset so both runtime and ESP Clang must come from the archive.
$env:LLGO_BUILD_CACHE = 'off'
$work = Join-Path $env:RUNNER_TEMP ('llgo-release-smoke-' + [Guid]::NewGuid())
New-Item -ItemType Directory $work | Out-Null
Push-Location $work
try {
  @'
module release-smoke

go 1.27

require github.com/goplus/lib v0.5.1
'@ | Set-Content -Encoding utf8 'go.mod'
  @'
package main

import (
	"fmt"
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/cpp/std"
)

func main() {
	fmt.Println("Hello, LLGo!")
	println("Hello, LLGo!")
	c.Printf(c.Str("Hello, LLGo!\n"))
	c.Printf(std.Str("Hello LLGo by cpp/std.Str\n").CStr())
}
'@ | Set-Content -Encoding utf8 'main.go'
  & $go mod tidy
  if ($LASTEXITCODE -ne 0) { throw 'Preparing the native release smoke test failed' }
  $program = Join-Path $work 'hello.exe'
  $trace = Invoke-ReleaseCapture $llgo @('build', '-x', '-o', $program, '.')
  $arch = @{ amd64 = 'x86_64'; arm64 = 'aarch64' }[$GoArch]
  $triple = if ($Profile -eq 'msvc') { "$arch-pc-windows-msvc" } else { "$arch-w64-windows-gnu" }
  if (-not $trace.Contains($triple)) { throw "Release smoke test did not select $triple`n$trace" }
  $null = Assert-ReleasePE -ReadObj $readObj -Path $program -GoArch $GoArch -Profile $Profile
  $env:LLGO_STDIO_NOBUF = '1'
  $output = Invoke-ReleaseCapture $program @()
  Remove-Item Env:LLGO_STDIO_NOBUF
  $lines = $output -split "`r?`n"
  if (@($lines | Where-Object { $_ -eq 'Hello, LLGo!' }).Count -ne 3 -or
      $lines -notcontains 'Hello LLGo by cpp/std.Str') {
    throw "Unexpected native Go/C/C++ output:`n$output"
  }

  New-Item -ItemType Directory 'embedded' | Out-Null
  Set-Content -Encoding utf8 'embedded/main.go' 'package main; func main() {}'
  $firmware = Join-Path $work 'embedded/demo.out'
  $trace = Invoke-ReleaseCapture $llgo @('build', '-x', '-target', 'esp32-coreboard-v2', '-o', $firmware, './embedded')
  if (-not (Test-Path "$firmware.elf")) { throw 'The integrated ESP Clang did not produce firmware' }
  # Go traces can use either slash style on Windows.
  if (-not $trace.Replace('\', '/').Contains($releaseRoot.Replace('\', '/') + '/crosscompile/clang')) {
    throw "The embedded build did not use the extracted release toolchain`n$trace"
  }
  Write-Host "Passed native Go/C/C++ and integrated ESP builds for windows/$GoArch/$Profile"
} finally {
  Pop-Location
  Remove-Item -LiteralPath $work -Recurse -Force
}
