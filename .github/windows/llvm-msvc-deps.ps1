# One verified host dependency installation for CI and release compiler links.
. (Join-Path $PSScriptRoot 'release-lib.ps1')

function Get-LLGoLLVMMSVCDependencies {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('amd64', 'arm64')][string]$GoArch,
    [Parameter(Mandatory = $true)][string]$LLVMVersion
  )
  $pins = Get-Content -Raw (Join-Path $PSScriptRoot 'llvm-msvc-deps.json') | ConvertFrom-Json
  if ($pins.llvm_version -ne $LLVMVersion) {
    throw "MSVC dependencies target LLVM $($pins.llvm_version), not $LLVMVersion"
  }
  $asset = $pins.assets.$GoArch
  if (-not $asset) { throw "No LLVM MSVC dependencies for $GoArch" }
  $cache = Join-Path $env:RUNNER_TOOL_CACHE 'llgo-llvm-msvc-deps'
  $archive = Get-ReleaseArchive `
    -URL "https://github.com/$($pins.repository)/releases/download/$($pins.release)/$($asset.name)" `
    -SHA256 $asset.sha256 -CacheDirectory $cache -Extension '.zip'
  # Always extract verified archive bytes rather than trusting a previous directory.
  $destination = Join-Path (Get-ReleaseTemporaryRoot) "llgo-llvm-msvc-deps/$($asset.sha256)"
  if (Test-Path -LiteralPath $destination) { Remove-Item -LiteralPath $destination -Recurse -Force }
  Expand-Archive -LiteralPath $archive -DestinationPath $destination
  $manifest = Get-Content -Raw (Join-Path $destination 'build-manifest.json') | ConvertFrom-Json
  $architecture = @{ amd64 = 'x86_64'; arm64 = 'aarch64' }[$GoArch]
  if ($manifest.schema -ne 1 -or $manifest.llvm_version -ne $LLVMVersion -or
      $manifest.producer_revision -ne $pins.producer_revision -or
      $manifest.version -ne $pins.version -or $manifest.architecture -ne $architecture -or
      $manifest.runtime -ne 'MT' -or $manifest.build_type -ne 'Release') {
    throw 'LLVM MSVC dependency manifest does not match the pinned host SDK contract'
  }
  foreach ($relative in @('lib/zs.lib', 'lib/zstd_static.lib', 'licenses/zlib-LICENSE', 'licenses/zstd-LICENSE')) {
    $file = Join-Path $destination $relative
    $expected = $manifest.files.$relative
    if ($expected -notmatch '^[0-9a-f]{64}$' -or -not (Test-Path -LiteralPath $file) -or
        (Get-FileHash -Algorithm SHA256 $file).Hash -ne $expected) {
      throw "Missing or corrupt LLVM MSVC dependency: $relative"
    }
  }
  return $destination
}
