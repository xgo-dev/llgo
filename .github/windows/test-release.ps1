param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('msvc', 'mingw')][string]$Profile,
  [Parameter(Mandatory = $true)]
  [ValidateSet('amd64', 'arm64', '386')][string]$GoArch,
  [switch]$CheckOnly,
  [string]$ReadObj
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-lib.ps1')
$archives = @(Get-ChildItem '.windows-dist' -Filter "*.windows-$GoArch-$Profile.zip")
if ($archives.Count -ne 1) { throw "Expected exactly one windows/$GoArch/$Profile ZIP" }
$archive = $archives[0]
$checksumLine = (Get-Content -Raw ($archive.FullName + '.sha256')).Trim()
$actual = (Get-FileHash -Algorithm SHA256 $archive.FullName).Hash.ToLowerInvariant()
if ($checksumLine -ne "$actual  $($archive.Name)") { throw 'The Windows ZIP checksum does not match' }

# A different location, including Unicode and spaces, makes build-tree dependencies
# and accidentally unquoted paths observable before an archive can be published.
$releaseRoot = Join-Path $env:RUNNER_TEMP "extracted ZIP 空格 llgo-$GoArch-$Profile"
if (Test-Path $releaseRoot) { Remove-Item -LiteralPath $releaseRoot -Recurse -Force }
Expand-Archive -LiteralPath $archive.FullName -DestinationPath $releaseRoot
& python (Join-Path $PSScriptRoot 'release_zip.py') verify $archive.FullName $releaseRoot
if ($LASTEXITCODE -ne 0) { throw 'The extracted Windows ZIP is incomplete or has invalid directory records' }
foreach ($entry in @('bin/llgo.exe', 'runtime/go.mod', 'targets', 'LICENSES', 'WINDOWS.md', 'THIRD_PARTY_NOTICES.md', 'release.json')) {
  if (-not (Test-Path (Join-Path $releaseRoot $entry))) { throw "Release archive is missing $entry" }
}
$metadata = Get-Content -Raw (Join-Path $releaseRoot 'release.json') | ConvertFrom-Json
if ($metadata.goos -ne 'windows' -or $metadata.goarch -ne $GoArch -or $metadata.abi -ne $Profile -or
    $archive.Name -ne "llgo$($metadata.version).windows-$GoArch-$Profile.zip") {
  throw 'The archive name, host architecture, and ABI metadata disagree'
}
$commit = (& git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $metadata.commit -ne $commit) { throw 'The release belongs to a different commit' }

$pins = Get-Content -Raw (Join-Path $PSScriptRoot '../../internal/crosscompile/espressif.json') | ConvertFrom-Json
$target = @{ amd64 = 'x86_64-w64-mingw32'; arm64 = 'aarch64-w64-mingw32'; '386' = 'i686-w64-mingw32' }[$GoArch]
$payload = $pins.payloads.$target
$bundle = Join-Path $releaseRoot "crosscompile/$($payload.asset)"
if ($metadata.distribution -ne 'full' -or $metadata.esp_clang_host -ne $target -or
    $metadata.esp_clang_version -ne $pins.release -or $metadata.esp_clang_asset -ne $payload.asset -or
    $metadata.esp_clang_sha256 -ne $payload.sha256 -or -not (Test-Path -LiteralPath $bundle)) {
  throw 'The Windows archive must contain the pinned ESP runtime archive and metadata'
}
if ((Get-FileHash -Algorithm SHA256 $bundle).Hash.ToLowerInvariant() -ne $payload.sha256) {
  throw 'The bundled ESP runtime checksum does not match its pin'
}
if (-not $ReadObj -or -not (Test-Path $ReadObj)) { throw 'ZIP checks require the CI inspection tool -ReadObj' }

$llgo = Join-Path $releaseRoot 'bin/llgo.exe'
if ($metadata.compiler_sha256 -notmatch '^[0-9a-f]{64}$' -or
    (Get-FileHash -Algorithm SHA256 $llgo).Hash.ToLowerInvariant() -ne $metadata.compiler_sha256) {
  throw 'The release compiler checksum does not match its shared host artifact'
}
Copy-ReleaseDLLs -ReadObj $readObj -Executable $llgo -GoArch $GoArch -Profile $Profile -CheckOnly
$go = (Get-Command go).Source
$buildInfo = Invoke-ReleaseCapture $go @('version', '-m', $llgo)
foreach ($setting in @("GOOS=windows", "GOARCH=$GoArch", "CGO_ENABLED=1", "vcs.revision=$commit", 'vcs.modified=false')) {
  if (-not $buildInfo.Contains($setting)) { throw "Release Go build information is missing $setting" }
}

# Exercise the portable alias outside the package tree for both ABIs.
$links = Join-Path $env:RUNNER_TEMP "portable links 空格 llgo-$GoArch-$Profile"
New-Item -ItemType Directory -Force $links | Out-Null
$alias = Join-Path $links 'llgo.exe'
if (Test-Path $alias) { Remove-Item -LiteralPath $alias -Force }
New-Item -ItemType SymbolicLink -Path $alias -Target $llgo | Out-Null
$llgo = $alias

$savedPath = $env:PATH
try {
  Remove-Item Env:LLGO_ROOT -ErrorAction SilentlyContinue
  $commandDirectory = Split-Path -Parent $llgo
  $env:PATH = $commandDirectory + ';' + (Join-Path $env:SystemRoot 'System32') + ';' + $env:SystemRoot
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
# LLGO_ROOT stays unset so runtime discovery uses the installed compiler alias.
$env:LLGO_BUILD_CACHE = 'off'
$work = Join-Path $env:RUNNER_TEMP ('llgo-release-smoke-' + [Guid]::NewGuid())
New-Item -ItemType Directory $work | Out-Null
Push-Location $work
try {
  $utf8 = [Text.UTF8Encoding]::new($false)
  $moduleSource = @'
module release-smoke

go 1.27

require github.com/goplus/lib v0.5.1
'@
  [IO.File]::WriteAllText((Join-Path $work 'go.mod'), $moduleSource, $utf8)
  $programSource = @'
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
'@
  [IO.File]::WriteAllText((Join-Path $work 'main.go'), $programSource, $utf8)
  & $go mod tidy
  if ($LASTEXITCODE -ne 0) { throw 'Preparing the native release smoke test failed' }
  $program = Join-Path $work 'hello.exe'
  $trace = Invoke-ReleaseCapture $llgo @('build', '-x', '-o', $program, '.')
  $arch = @{ amd64 = 'x86_64'; arm64 = 'aarch64'; '386' = 'i686' }[$GoArch]
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
  Write-Host "Passed native Go/C/C++ compilation and execution for windows/$GoArch/$Profile"

  New-Item -ItemType Directory 'embedded' | Out-Null
  [IO.File]::WriteAllText((Join-Path $work 'embedded/main.go'), 'package main; func main() {}', $utf8)
  $savedLocalAppData = $env:LOCALAPPDATA
  $env:LOCALAPPDATA = Join-Path $work 'fresh user cache'
  New-Item -ItemType Directory -Force $env:LOCALAPPDATA | Out-Null
  # Embedded builds must not rely on the native SDK. Archive extraction can
  # discover installed 7-Zip separately or use an xz-capable Windows tar.
  # Keep the native SDK out of both tool discovery and compiler flags.
  $embeddedEnvironment = @{}
  foreach ($name in @('PATH', 'LLVM_CONFIG', 'CC', 'CXX', 'CGO_CPPFLAGS', 'CGO_CFLAGS', 'CGO_CXXFLAGS', 'CGO_LDFLAGS')) {
    $embeddedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name)
    [Environment]::SetEnvironmentVariable($name, $null)
  }
  $env:PATH = (Split-Path $go) + ';' + (Join-Path $env:SystemRoot 'System32') + ';' + $env:SystemRoot
  $cached = Join-Path $env:LOCALAPPDATA "llgo/crosscompile/esp-clang-$($pins.release)-$target"
  $evidence = Join-Path $env:RUNNER_TEMP "llgo-esp-runtime-$GoArch-$Profile"
  New-Item -ItemType Directory -Force $evidence | Out-Null
  $results = @()
  Write-Host "Testing bundled ESP runtime $($pins.release) from $bundle"
  foreach ($board in @('esp32-coreboard-v2', 'esp32c3')) {
    foreach ($mode in @('default', 'full')) {
      $firmware = Join-Path $work "embedded/$board-$mode.out"
      $buildArguments = @('build', '-x')
      if ($mode -eq 'full') { $buildArguments += '-lto=full' }
      $trace = Invoke-ReleaseCapture $llgo ($buildArguments + @('-target', $board, '-o', $firmware, './embedded'))
      [IO.File]::WriteAllText((Join-Path $evidence "$board-$mode.log"), $trace, $utf8)
      if (-not $trace.Replace('\', '/').Contains($cached.Replace('\', '/'))) {
        throw "The embedded build did not use the pinned host toolchain`n$trace"
      }
      if ($trace.Contains('ESP Clang not found') -or $trace.Contains('Downloading ESP Clang')) {
        throw "A released package must not download the ESP toolchain`n$trace"
      }
      $unpacked = $trace.Contains('Using bundled ESP Clang archive:')
      if ($unpacked -ne ($results.Count -eq 0)) {
        throw "Expected local ESP extraction followed by cache reuse`n$trace"
      }
      if ($unpacked -and -not $trace.Replace('\', '/').Contains($bundle.Replace('\', '/'))) {
        throw "The embedded build did not use the bundled archive`n$trace"
      }
      $header = [IO.File]::ReadAllBytes("$firmware.elf")
      $machine = if ($board -eq 'esp32c3') { 243 } else { 94 }
      if ($header.Length -lt 20 -or [BitConverter]::ToUInt32($header, 0) -ne 0x464c457f -or
          $header[4] -ne 1 -or $header[5] -ne 1 -or [BitConverter]::ToUInt16($header, 18) -ne $machine) {
        throw "The bundled ESP tools produced an invalid ELF for $board/$mode"
      }
      $results += "$board-$mode"
      Write-Host "Passed ESP firmware $board/$mode for windows/$GoArch/$Profile"
    }
  }
  Assert-ReleaseESPPayload -Root $cached -GoArch $GoArch -Version $pins.release
  $manifest = Get-Content -LiteralPath (Join-Path $cached 'LLGO-LLVM-MANIFEST.txt')
  if ($manifest -notcontains 'windows_llvm_link_mode=shared' -or $manifest -notcontains 'llvm_expected_version=22.1.4') {
    throw 'The bundled ESP tools must use shared LLVM 22.1.4'
  }
  if (-not (Test-Path (Join-Path $cached 'LICENSE-LLVM.txt'))) { throw 'Bundled ESP tools are missing their license' }
  @{
    repository = $pins.repository; release = $pins.release; target = $target
    source = 'bundled-archive'; firmware = $results; status = 'passed'
  } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $evidence 'results.json') -Encoding utf8
  Write-Host "Passed native Go/C/C++ and bundled ESP builds for windows/$GoArch/$Profile"

} finally {
  if (Test-Path variable:embeddedEnvironment) {
    foreach ($name in $embeddedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $embeddedEnvironment[$name]) }
  }
  if (Test-Path variable:savedLocalAppData) { $env:LOCALAPPDATA = $savedLocalAppData }
  Pop-Location
  Remove-Item -LiteralPath $work -Recurse -Force
}
