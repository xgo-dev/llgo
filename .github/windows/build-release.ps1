param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('msvc', 'mingw')][string]$Profile,
  [Parameter(Mandatory = $true)]
  [ValidateSet('amd64', 'arm64')][string]$GoArch
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
$stage = Join-Path $env:RUNNER_TEMP "llgo-release-$GoArch-$Profile"
if (Test-Path $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
$bin = Join-Path $stage 'bin'
New-Item -ItemType Directory -Force $bin | Out-Null

$llvmVersion = '22.1.8'
$readObj = (Get-Command llvm-readobj.exe).Source
$llvmRoot = if ($Profile -eq 'mingw') {
  $env:LLGO_MINGW_HOST_ROOT
} else {
  Split-Path (Split-Path (Get-Command llvm-config.exe).Source)
}
if ($Profile -eq 'msvc' -and $GoArch -eq 'arm64') {
  # The ordinary MSVC CI profile builds an x64 compiler for all targets.
  # A release labelled arm64 must link the actual ARM64 LLVM libraries.
  $sdkParent = Join-Path $env:RUNNER_TOOL_CACHE "llgo-release-llvm/$llvmVersion/arm64"
  $sdkName = "clang+llvm-$llvmVersion-aarch64-pc-windows-msvc"
  $llvmRoot = Join-Path $sdkParent $sdkName
  if (-not (Test-Path (Join-Path $llvmRoot '.complete'))) {
    if (Test-Path $sdkParent) { Remove-Item -LiteralPath $sdkParent -Recurse -Force }
    $asset = $sdkName.Replace('+', '%2B') + '.tar.xz'
    Expand-ReleaseXz `
      -URL "https://github.com/llvm/llvm-project/releases/download/llvmorg-$llvmVersion/$asset" `
      -SHA256 'de718c58ebbc5f61d58c17b90457fcf42983bc2c4a4aba3e010d108713bfd7f1' `
      -Destination $sdkParent `
      -Entries @("$sdkName/include/*", "$sdkName/lib/*", "$sdkName/bin/llvm-config.exe", "$sdkName/bin/*.dll")
    if (-not (Test-Path (Join-Path $llvmRoot 'include/llvm-c/Core.h')) -or
        -not (Test-Path (Join-Path $llvmRoot 'lib/LLVMCore.lib'))) {
      throw 'The ARM64 LLVM development SDK is incomplete'
    }
    New-Item -ItemType File (Join-Path $llvmRoot '.complete') | Out-Null
  }
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
if ($Profile -eq 'msvc') {
  . (Join-Path $PSScriptRoot 'msvc-target.ps1')
  $target = Enter-LLGoVisualStudio2022 -GoArch $GoArch
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
  $env:CGO_LDFLAGS = '-L"' + (Join-Path $llvmRoot 'lib').Replace('\', '/') + '" ' + ($libraryFlags -join ' ')
} else {
  $env:CGO_LDFLAGS = (Invoke-ReleaseCapture $llvmConfig @('--ldflags', '--libs', 'all', '--system-libs')).Trim().Replace('\', '/').Replace("`n", ' ')
}

$executable = Join-Path $bin 'llgo.exe'
$buildTime = ([DateTimeOffset]$metadata.date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$linkFlags = "-X github.com/xgo-dev/llgo/internal/env.buildVersion=v$($metadata.version) " +
  "-X github.com/xgo-dev/llgo/internal/env.buildTime=$buildTime"
& go build -tags=byollvm -ldflags $linkFlags -o $executable ./cmd/llgo
if ($LASTEXITCODE -ne 0) { throw 'Building the Windows release compiler failed' }
Copy-ReleaseDLLs -ReadObj $readObj -Executable $executable -GoArch $GoArch -Profile $Profile `
  -SourceDirectories @((Join-Path $llvmRoot 'bin'))

foreach ($entry in @('LICENSE', 'LICENSES', 'README.md', 'THIRD_PARTY_NOTICES.md', 'runtime', 'targets')) {
  Copy-Item -LiteralPath (Join-Path $root $entry) -Destination $stage -Recurse
}
if ($Profile -eq 'mingw') {
  # Preserve the MSYS2 component licenses alongside redistributed LLVM/C++
  # and transitive DLLs. Nothing from MSYS2's POSIX /usr/bin is packaged.
  Copy-Item -LiteralPath (Join-Path $llvmRoot 'share/licenses') `
    -Destination (Join-Path $stage 'LICENSES/windows-runtime') -Recurse
}

# Keep the existing integrated layout. ESP's Windows payload is x64 for both
# host architectures, as in LLGo's existing Windows ARM64 download path.
$espVersion = '22.1.4_20260905'
$espAsset = "clang-esp-$espVersion-x86_64-w64-mingw32.tar.xz"
$espParent = Join-Path $env:RUNNER_TEMP ('llgo-release-esp-' + [Guid]::NewGuid())
try {
  Expand-ReleaseXz `
    -URL "https://github.com/goplus/espressif-llvm-project-prebuilt/releases/download/$espVersion/$espAsset" `
    -SHA256 '3d32533daec8be08e608496eff817798eb7d3c25f07a02de1f1c94c0a0bbb8b3' `
    -Destination $espParent
  $crosscompile = Join-Path $stage 'crosscompile'
  New-Item -ItemType Directory $crosscompile | Out-Null
  Move-Item -LiteralPath (Join-Path $espParent 'esp-clang') -Destination (Join-Path $crosscompile 'clang')
} finally {
  if (Test-Path $espParent) { Remove-Item -LiteralPath $espParent -Recurse -Force }
}
Copy-Item -LiteralPath (Join-Path $root 'LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt') `
  -Destination (Join-Path $stage 'crosscompile/clang/LICENSE-LLVM.txt')
@{
  version = $metadata.version
  commit = $commit
  date = $buildTime
  goos = 'windows'
  goarch = $GoArch
  abi = $Profile
  llvm_version = $llvmVersion
  esp_clang_version = $espVersion
  esp_clang_host = 'windows/amd64'
} | ConvertTo-Json | Set-Content -Encoding utf8 (Join-Path $stage 'release.json')

$archiveName = "llgo$($metadata.version).windows-$GoArch-$Profile.tar.gz"
$archive = Join-Path $root ".windows-dist/$archiveName"
& tar.exe -czf $archive -C $stage .
if ($LASTEXITCODE -ne 0) { throw 'Creating the integrated Windows archive failed' }
$checksum = (Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant()
"$checksum  $archiveName" | Set-Content -Encoding ascii "$archive.sha256"
Write-Host "Created $archive"
