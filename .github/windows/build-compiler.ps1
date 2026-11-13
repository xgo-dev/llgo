param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('amd64', 'arm64', '386')][string]$GoArch,
  [Parameter(Mandatory = $true)][string]$OutputDirectory
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-lib.ps1')
$root = (Get-Location).Path
$metadata = Get-Content -Raw '.windows-dist/metadata.json' | ConvertFrom-Json
$commit = (& git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $metadata.commit -ne $commit) {
  throw 'The GoReleaser metadata does not describe this checkout'
}
if ($metadata.version -notmatch '^[0-9A-Za-z][0-9A-Za-z.+-]*$') {
  throw "Invalid release version: $($metadata.version)"
}
$stage = $OutputDirectory
if (Test-Path $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
$bin = Join-Path $stage 'bin'
New-Item -ItemType Directory -Force $bin | Out-Null

$llvmVersion = '22.1.8'
$readObj = (Get-Command llvm-readobj.exe).Source
$llvmRoot = Split-Path (Split-Path (Get-Command llvm-config.exe).Source)
if ($GoArch -eq 'arm64') {
  # The ordinary MSVC CI profile builds an x64 compiler for all targets.
  # A release labelled arm64 must link the actual ARM64 LLVM libraries.
  $sdkParent = Join-Path $env:RUNNER_TEMP "llgo-release-llvm/$llvmVersion/arm64"
  $sdkName = "clang+llvm-$llvmVersion-aarch64-pc-windows-msvc"
  $llvmRoot = Join-Path $sdkParent $sdkName
  if (Test-Path $sdkParent) { Remove-Item -LiteralPath $sdkParent -Recurse -Force }
  $asset = $sdkName.Replace('+', '%2B') + '.tar.xz'
  Expand-ReleaseXz `
    -URL "https://github.com/llvm/llvm-project/releases/download/llvmorg-$llvmVersion/$asset" `
    -SHA256 'de718c58ebbc5f61d58c17b90457fcf42983bc2c4a4aba3e010d108713bfd7f1' `
    -CacheDirectory (Join-Path $env:RUNNER_TOOL_CACHE 'llgo-release-downloads/llvm-arm64') `
    -Destination $sdkParent `
    -Entries @("$sdkName/include/*", "$sdkName/lib/*", "$sdkName/bin/llvm-config.exe", "$sdkName/bin/*.dll")
  if (-not (Test-Path (Join-Path $llvmRoot 'include/llvm-c/Core.h')) -or
      -not (Test-Path (Join-Path $llvmRoot 'lib/LLVMCore.lib'))) {
    throw 'The ARM64 LLVM development SDK is incomplete'
  }
}

if ($GoArch -eq '386') {
  $toolchain = Get-Content -Raw (Join-Path $PSScriptRoot '../../internal/crosscompile/espressif.json') | ConvertFrom-Json
  if ($toolchain.sdk.version -ne $llvmVersion) { throw 'The x86 SDK pin does not match the host LLVM version' }
  $parent = Join-Path $env:RUNNER_TEMP 'llgo-release-x86-sdk'
  if (Test-Path $parent) { Remove-Item -Recurse -Force $parent }
  Expand-ReleaseXz `
    -URL "https://github.com/$($toolchain.sdk.repository)/releases/download/$($toolchain.sdk.release)/$($toolchain.sdk.asset)" `
    -SHA256 $toolchain.sdk.sha256 `
    -CacheDirectory (Join-Path $env:RUNNER_TOOL_CACHE 'llgo-release-downloads/llvm-x86') `
    -Destination $parent
  $llvmRoot = Join-Path $parent 'llvm-sdk'
  $manifest = Get-Content (Join-Path $llvmRoot 'LLGO-LLVM-MANIFEST.txt')
  if ($manifest -notcontains 'host_target=i686-w64-mingw32' -or $manifest -notcontains 'llvm_version=22.1.8' -or
      $manifest -notcontains 'source_revision=ca7933e47d3a3451d81e72ac174dcb5aa28b59d1' -or
      $manifest -notcontains 'link_mode=static') {
    throw 'The LLVM development SDK is not Windows x86/LLVM 22.1.8'
  }
  $null = Assert-ReleasePE -ReadObj $readObj -Path (Join-Path $llvmRoot 'bin/llvm-config.exe') -GoArch '386' -Profile 'mingw'
}
$llvmConfig = Join-Path $llvmRoot 'bin/llvm-config.exe'
if ((Invoke-ReleaseCapture $llvmConfig @('--version')).Trim() -ne $llvmVersion) {
  throw "The release compiler requires LLVM $llvmVersion"
}
$env:CGO_ENABLED = '1'
$env:GOOS = 'windows'
$env:GOARCH = $GoArch
$env:CGO_CPPFLAGS = (Invoke-ReleaseCapture $llvmConfig @('--cflags')).Trim().Replace('\', '/').Replace("`n", ' ')
$env:CGO_CXXFLAGS = '-std=c++17'
if ($GoArch -eq '386') {
  $bootstrap = Get-ReleaseX86Bootstrap
  $env:CC = '"' + (Join-Path $bootstrap 'bin/i686-w64-mingw32-clang.exe').Replace('\', '/') + '"'
  $env:CXX = '"' + (Join-Path $bootstrap 'bin/i686-w64-mingw32-clang++.exe').Replace('\', '/') + '"'
  $flags = (Invoke-ReleaseCapture $llvmConfig @('--link-static', '--ldflags', '--libs', 'all', '--system-libs')).Trim()
  # The host compiler is independent of the C ABI selected for user programs.
  # Static C++/unwind keeps both profiles' x86 llgo.exe free of toolchain DLLs.
  $env:CGO_LDFLAGS = $flags.Replace('\', '/').Replace("`n", ' ') + ' -static -lc++ -lunwind'
} else {
  . (Join-Path $PSScriptRoot 'msvc-target.ps1')
  $target = Enter-LLGoVisualStudio -GoArch $GoArch
  # Keep the bootstrap driver from setup-deps; the developer shell can put a
  # different Visual Studio Clang first on PATH.
  $bootstrapBin = Split-Path $readObj
  $env:CC = '"' + (Join-Path $bootstrapBin 'clang.exe').Replace('\', '/') +
    '" --target=' + $target.Triple + ' -fuse-ld=lld -fms-runtime-lib=static'
  $env:CXX = '"' + (Join-Path $bootstrapBin 'clang++.exe').Replace('\', '/') +
    '" --target=' + $target.Triple + ' -fuse-ld=lld -fms-runtime-lib=static'
  $libraryNames = (Invoke-ReleaseCapture $llvmConfig @('--link-static', '--libnames', 'all')).Trim() -split '\s+'
  $systemNames = (Invoke-ReleaseCapture $llvmConfig @('--link-static', '--system-libs')) -split '\s+' |
    Where-Object { $_ -and $_ -notin @('libxml2s.lib', 'xml2s.lib') }
  # Match setup-deps: the official SDK advertises optional XML support but
  # omits its library. LLGo does not link that unused manifest component.
  $libraryFlags = @($libraryNames) + @($systemNames) | ForEach-Object {
    if (-not $_.EndsWith('.lib', [StringComparison]::OrdinalIgnoreCase)) {
      throw "Unexpected MSVC LLVM library: $_"
    }
    '-l' + [IO.Path]::GetFileNameWithoutExtension($_)
  }
  $env:CGO_LDFLAGS = Get-ReleaseMSVCLinkFlags -LibraryDirectory (Join-Path $llvmRoot 'lib') -LibraryFlags $libraryFlags
}

$executable = Join-Path $bin 'llgo.exe'
$buildTime = ([DateTimeOffset]$metadata.date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$linkFlags = "-X github.com/xgo-dev/llgo/internal/env.buildVersion=v$($metadata.version) " +
  "-X github.com/xgo-dev/llgo/internal/env.buildTime=$buildTime"
& go build -tags=byollvm -ldflags $linkFlags -o $executable ./cmd/llgo
if ($LASTEXITCODE -ne 0) { throw 'Building the Windows release compiler failed' }
Copy-ReleaseDLLs -ReadObj $readObj -Executable $executable -GoArch $GoArch -Profile 'msvc' `
  -SourceDirectories @((Join-Path $llvmRoot 'bin'))

# CI-only inspection tools travel with the host artifact, never in release ZIPs.
# This permits DLL/startup checks before any SDK is installed on a fresh runner.
$inspection = Join-Path $stage 'inspection'
New-Item -ItemType Directory -Force $inspection | Out-Null
$inspector = Join-Path $inspection 'llvm-readobj.exe'
Copy-Item -LiteralPath $readObj -Destination $inspector
$inspectionMachine = (Get-ReleasePE -ReadObj $readObj -Path $readObj).Machine
$inspectionArch = @{ AMD64 = 'amd64'; ARM64 = 'arm64'; I386 = '386' }[$inspectionMachine]
Copy-ReleaseDLLs -ReadObj $readObj -Executable $inspector -GoArch $inspectionArch -Profile 'mingw' `
  -SourceDirectories @((Split-Path $readObj))

# Each architecture has one host compiler, shared by both native ABI packages.
# LLVM/C++ remains static; native C ABI selection belongs to the user's tools.
$licenses = Join-Path $stage 'licenses'
New-Item -ItemType Directory -Force $licenses | Out-Null
Copy-Item -LiteralPath (Join-Path $root 'LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt') `
  -Destination (Join-Path $licenses 'LLGO-LLVM-LICENSE.txt')
if ($GoArch -eq '386') {
  Copy-Item -Path (Join-Path $llvmRoot 'third-party-licenses/*') -Destination $licenses -Recurse
}
@{
  version = $metadata.version
  commit = $commit
  date = $buildTime
  goarch = $GoArch
  llvm_version = $llvmVersion
  compiler_sha256 = (Get-FileHash -Algorithm SHA256 $executable).Hash.ToLowerInvariant()
} | ConvertTo-Json | Set-Content -Encoding utf8 (Join-Path $stage 'compiler.json')
Write-Host "Built the shared windows/$GoArch compiler in $stage"
