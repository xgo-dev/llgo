# Exercise the real Go/cgo parser and the archive cache's integrity boundary.
param([string]$Go = 'go')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-lib.ps1')
$temporary = Join-Path ([IO.Path]::GetTempPath()) ('llgo-release-build-' + [Guid]::NewGuid())
New-Item -ItemType Directory $temporary | Out-Null
$savedEnvironment = @{}
foreach ($name in @('CGO_ENABLED', 'CGO_LDFLAGS', 'CGO_CPPFLAGS', 'CGO_CFLAGS', 'CGO_CXXFLAGS', 'GOOS', 'GOARCH', 'GOWORK')) {
  $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name)
}

try {
  # A cache hit must still match the pin. The invalid URL makes accidental
  # network access on this path fail rather than hide a cache regression.
  $seed = Join-Path $temporary 'seed'
  [IO.File]::WriteAllText($seed, 'verified release payload')
  $sha256 = (Get-FileHash -Algorithm SHA256 $seed).Hash.ToLowerInvariant()
  $cache = Join-Path $temporary 'cache'
  New-Item -ItemType Directory $cache | Out-Null
  $archive = Join-Path $cache "$sha256.tar.xz"
  Copy-Item $seed $archive
  $arguments = @{ URL = 'https://invalid.invalid/payload'; SHA256 = $sha256; CacheDirectory = $cache }
  if ((Get-ReleaseArchive @arguments) -ne $archive) { throw 'The verified cache was not reused' }
  [IO.File]::WriteAllText($archive, 'tampered release payload')
  $rejected = $false
  try { $null = Get-ReleaseArchive @arguments } catch {
    if ($_.Exception.Message -notmatch 'SHA-256.*expected') { throw }
    $rejected = $true
  }
  if (-not $rejected) { throw 'A corrupted cached archive was accepted' }

  foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $null) }
  $env:CGO_ENABLED = '1'
  $env:GOWORK = 'off'
  $env:GOOS = (& $Go env GOHOSTOS).Trim()
  $env:GOARCH = (& $Go env GOHOSTARCH).Trim()
  Set-Content (Join-Path $temporary 'go.mod') "module release-cgo-test`n`ngo 1.20`n"
  @'
package main

import "C"
import (
    "fmt"
    "runtime"
    "runtime/debug"
    "strings"
)

func main() {
    var pcs [1]uintptr
    if runtime.Callers(1, pcs[:]) != 1 {
        panic("missing runtime frame")
    }
    frame, _ := runtime.CallersFrames(pcs[:]).Next()
    if frame.Function != "main.main" {
        panic("runtime function name was lost")
    }
    info, ok := debug.ReadBuildInfo()
    if !ok || info.Path != "release-cgo-test" {
        panic("build info was lost")
    }
    for _, setting := range info.Settings {
        if setting.Key == "-ldflags" &&
            strings.Contains(setting.Value, "buildVersion=v1.2.3") &&
            strings.Contains(setting.Value, "buildTime=2026-11-01T00:00:00Z") {
            fmt.Println("release runtime metadata: ok")
            return
        }
    }
    panic("release version or time metadata was lost")
}
'@ | Set-Content (Join-Path $temporary 'main.go')
  Push-Location $temporary
  try {
    # Match the running Go tool's architecture, including an x64 Go tool on
    # ARM64 Windows. Cover both the CI path and a spaced SDK path.
    foreach ($directory in @('lib', 'LLVM SDK/lib')) {
      $libraryDirectory = Join-Path $temporary $directory
      New-Item -ItemType Directory -Force $libraryDirectory | Out-Null
      $env:CGO_LDFLAGS = Get-ReleaseMSVCLinkFlags -LibraryDirectory $libraryDirectory
      $executable = Join-Path $temporary 'cgo-test.exe'
      $linkFlags = Get-ReleaseCompilerLinkFlags -Version '1.2.3' -BuildTime '2026-11-01T00:00:00Z'
      & $Go build -ldflags $linkFlags -o $executable .
      if ($LASTEXITCODE -ne 0) { throw 'The release CGO_LDFLAGS failed a real Go build' }
      $sections = & llvm-readobj --sections $executable
      if ($LASTEXITCODE -ne 0) { throw 'Inspecting release debug sections failed' }
      if ($sections -match 'Name: \.z?debug_') {
        throw 'Release compiler must omit DWARF to preserve the compact package budget'
      }
      # MSVC external linking need not emit a COFF symbol table. The runtime
      # must still recover function names and build info without DWARF.
      $runtimeMetadata = & $executable
      if ($LASTEXITCODE -ne 0 -or ($runtimeMetadata -join "`n").Trim() -ne 'release runtime metadata: ok') {
        throw 'Release compiler must retain executable Go runtime metadata'
      }
      Write-Host "Verified release runtime metadata: $directory ($env:GOARCH)"
    }
  } finally {
    Pop-Location
  }
  # A shared compiler is bound to one source/version/architecture. Reject
  # stale or modified artifacts before either ABI archive can copy its bytes.
  $hostRoot = Join-Path $temporary 'host-compiler'
  New-Item -ItemType Directory -Force "$hostRoot/bin", "$hostRoot/licenses" | Out-Null
  $hostExecutable = Join-Path $hostRoot 'bin/llgo.exe'
  [IO.File]::WriteAllText($hostExecutable, 'host compiler bytes')
  $metadata = @{ version = '1.2.3'; commit = ('a' * 40); date = '2026-10-19T00:00:00Z' }
  $manifest = @{
    version = $metadata.version; commit = $metadata.commit; date = $metadata.date
    goarch = '386'; llvm_version = '22.1.8'
    compiler_sha256 = (Get-FileHash -Algorithm SHA256 $hostExecutable).Hash.ToLowerInvariant()
  }
  $manifestPath = Join-Path $hostRoot 'compiler.json'
  $manifest | ConvertTo-Json | Set-Content -Encoding utf8 $manifestPath
  Assert-ReleaseCompilerArtifact -Root $hostRoot -Metadata $metadata -GoArch '386' -LLVMVersion '22.1.8'
  foreach ($key in @('version', 'commit', 'date', 'goarch', 'llvm_version', 'compiler_sha256')) {
    $invalid = $manifest.Clone()
    $invalid[$key] = 'stale'
    $invalid | ConvertTo-Json | Set-Content -Encoding utf8 $manifestPath
    $rejected = $false
    try { Assert-ReleaseCompilerArtifact -Root $hostRoot -Metadata $metadata -GoArch '386' -LLVMVersion '22.1.8' } catch { $rejected = $true }
    if (-not $rejected) { throw "A compiler with stale $key was accepted" }
  }
  $manifest | ConvertTo-Json | Set-Content -Encoding utf8 $manifestPath
  [IO.File]::WriteAllText($hostExecutable, 'tampered compiler bytes')
  $rejected = $false
  try { Assert-ReleaseCompilerArtifact -Root $hostRoot -Metadata $metadata -GoArch '386' -LLVMVersion '22.1.8' } catch { $rejected = $true }
  if (-not $rejected) { throw 'A tampered shared compiler was accepted' }
  Write-Host 'Passed cached archive integrity and CGO linker argument checks'
} finally {
  foreach ($name in $savedEnvironment.Keys) {
    [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name])
  }
  Remove-Item -LiteralPath $temporary -Recurse -Force
}
