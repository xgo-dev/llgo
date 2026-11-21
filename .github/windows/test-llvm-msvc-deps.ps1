$ErrorActionPreference = 'Stop'
$testRoot = Join-Path ([IO.Path]::GetTempPath()) "llgo-msvc-deps-$([Guid]::NewGuid())"
$savedTemp = $env:RUNNER_TEMP
$savedCache = $env:RUNNER_TOOL_CACHE
try {
  New-Item -ItemType Directory -Force $testRoot | Out-Null
  Copy-Item "$PSScriptRoot/llvm-msvc-deps.ps1", "$PSScriptRoot/release-lib.ps1" $testRoot
  $env:RUNNER_TEMP = Join-Path $testRoot 'temp with spaces'
  $env:RUNNER_TOOL_CACHE = Join-Path $testRoot 'cache with spaces'
  $source = Join-Path $testRoot 'package'
  foreach ($directory in @('lib', 'licenses')) {
    New-Item -ItemType Directory -Force (Join-Path $source $directory) | Out-Null
  }
  $files = @{}
  foreach ($relative in @('lib/zs.lib', 'lib/zstd_static.lib', 'licenses/zlib-LICENSE', 'licenses/zstd-LICENSE')) {
    $file = Join-Path $source $relative
    "fixture $relative" | Set-Content $file
    $files[$relative] = (Get-FileHash $file -Algorithm SHA256).Hash.ToLowerInvariant()
  }
  $revision = '1234567890123456789012345678901234567890'
  @{
    schema = 1; llvm_version = '23.1.6'; version = '23.1.6-r1'
    producer_revision = $revision; architecture = 'x86_64'
    runtime = 'MT'; build_type = 'Release'; files = $files
  } | ConvertTo-Json -Depth 5 | Set-Content (Join-Path $source 'build-manifest.json')
  $zip = Join-Path $testRoot 'deps.zip'
  Compress-Archive -Path "$source/*" -DestinationPath $zip
  $sha = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLowerInvariant()
  $cache = Join-Path $env:RUNNER_TOOL_CACHE 'llgo-llvm-msvc-deps'
  New-Item -ItemType Directory -Force $cache | Out-Null
  $cachedZip = Join-Path $cache "$sha.zip"
  Copy-Item $zip $cachedZip
  $asset = @{ name = 'deps.zip'; sha256 = $sha }
  @{
    repository = 'invalid/no-network'; release = 'fixture'; version = '23.1.6-r1'
    llvm_version = '23.1.6'; producer_revision = $revision
    assets = @{ amd64 = $asset; arm64 = $asset }
  } | ConvertTo-Json -Depth 5 | Set-Content (Join-Path $testRoot 'llvm-msvc-deps.json')
  . (Join-Path $testRoot 'llvm-msvc-deps.ps1')
  $installed = Get-LLGoLLVMMSVCDependencies -GoArch amd64 -LLVMVersion '23.1.6'
  if (-not (Test-Path "$installed/lib/zs.lib")) { throw 'Valid package was not installed' }
  # An extracted directory is disposable; restore it from checked bytes on every call.
  'corrupt extracted file' | Set-Content "$installed/lib/zs.lib"
  $installed = Get-LLGoLLVMMSVCDependencies -GoArch amd64 -LLVMVersion '23.1.6'
  if ((Get-FileHash "$installed/lib/zs.lib" -Algorithm SHA256).Hash -ne $files['lib/zs.lib']) {
    throw 'Corrupt extracted files were reused'
  }
  foreach ($case in @(@{ arch = 'arm64'; version = '23.1.6' },
                     @{ arch = 'amd64'; version = '23.1.5' })) {
    $rejected = $false
    try { $null = Get-LLGoLLVMMSVCDependencies -GoArch $case.arch -LLVMVersion $case.version }
    catch { $rejected = $true }
    if (-not $rejected) { throw 'Mismatched SDK/architecture was accepted' }
  }
  'corrupt cache' | Set-Content $cachedZip
  $rejected = $false
  try { $null = Get-LLGoLLVMMSVCDependencies -GoArch amd64 -LLVMVersion '23.1.6' }
  catch { $rejected = $true }
  if (-not $rejected) { throw 'Corrupt cached bytes were accepted' }
  $flags = Get-ReleaseMSVCLinkFlags -LibraryDirectory 'C:/LLVM SDK/lib' `
    -AdditionalLibraryDirectories @('C:/dependency package/lib') -LibraryFlags @('-lzs')
  if ($flags -ne '"-LC:/LLVM SDK/lib" "-LC:/dependency package/lib" -lzs') {
    throw "Library directory quoting changed: $flags"
  }
  Write-Host 'LLVM MSVC dependency integrity, identity and relocation checks passed'
} finally {
  $env:RUNNER_TEMP = $savedTemp
  $env:RUNNER_TOOL_CACHE = $savedCache
  if (Test-Path $testRoot) { Remove-Item -LiteralPath $testRoot -Recurse -Force }
}
