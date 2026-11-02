$ErrorActionPreference = 'Stop'
$env:LLGO_INSTALLER_LIBRARY_ONLY = '1'
. (Join-Path $PSScriptRoot '..\..\install.ps1')

function Assert-Equal {
  param($Actual, $Expected)
  if ($Actual -ne $Expected) {
    throw "Got '$Actual', want '$Expected'"
  }
}

Assert-Equal (ConvertTo-LLGoArchitecture 'X64') 'amd64'
Assert-Equal (ConvertTo-LLGoArchitecture 'AMD64') 'amd64'
Assert-Equal (ConvertTo-LLGoArchitecture 'Arm64') 'arm64'
Assert-Equal (ConvertTo-LLGoArchitecture 'x86') '386'
$savedInstallArch = $env:LLGO_ARCH
try {
  foreach ($arch in @('386', 'amd64', 'arm64')) {
    $env:LLGO_ARCH = $arch
    Assert-Equal (Resolve-LLGoInstallArchitecture) $arch
  }
  $env:LLGO_ARCH = $null
  Assert-Equal (Resolve-LLGoInstallArchitecture) (Get-LLGoNativeArchitecture)
  $env:LLGO_ARCH = 'mips'
  $invalidRejected = $false
  try { $null = Resolve-LLGoInstallArchitecture } catch { $invalidRejected = $true }
  if (-not $invalidRejected) { throw 'An unsupported installer host architecture was accepted' }
} finally { $env:LLGO_ARCH = $savedInstallArch }

Assert-Equal (ConvertTo-LLGoVersion 'v1.2.3-rc.1') '1.2.3-rc.1'

# Unsupported dependency setup must reject before even creating the root.
$savedX86Environment = @{}
foreach ($name in @('OS', 'LLGO_ARCH', 'LLGO_ABI', 'LLGO_VERSION', 'LLGO_INSTALL_ROOT', 'LLGO_INSTALL_DEPS', 'LLGO_UPDATE_PATH')) {
  $savedX86Environment[$name] = [Environment]::GetEnvironmentVariable($name)
}
$x86Root = Join-Path ([IO.Path]::GetTempPath()) "llgo-rejected-x86-$([Guid]::NewGuid())"
try {
  $env:OS = 'Windows_NT'
  $env:LLGO_ARCH = '386'
  $env:LLGO_ABI = 'msvc'
  $env:LLGO_VERSION = '1.2.3'
  $env:LLGO_INSTALL_ROOT = $x86Root
  $env:LLGO_INSTALL_DEPS = '1'
  $env:LLGO_UPDATE_PATH = '0'
  $rejected = $false
  try { Install-LLGo } catch {
    if ($_.Exception.Message -notmatch 'Automatic x86 development dependency installation is unavailable') { throw }
    $rejected = $true
  }
  if (-not $rejected -or (Test-Path -LiteralPath $x86Root)) {
    throw 'Unsupported x86 dependency setup did not reject without creating its root'
  }
} finally {
  foreach ($name in $savedX86Environment.Keys) {
    [Environment]::SetEnvironmentVariable($name, $savedX86Environment[$name])
  }
}

Assert-Equal (ConvertTo-LLGoGoVersion 'go1.27rc1').ToString() '1.27.0'
Assert-Equal (ConvertTo-LLGoGoVersion 'devel go1.28-abcdef').ToString() '1.28.0'
$latestVersion = Get-LLGoLatestVersion
if ($latestVersion -notmatch '^\d+\.\d+') {
  throw "Unexpected latest LLGo version: $latestVersion"
}
$amd64Components = @(Get-LLGoVisualStudioComponents 'amd64')
$arm64Components = @(Get-LLGoVisualStudioComponents 'arm64')
Assert-Equal $amd64Components.Count 1
Assert-Equal $arm64Components.Count 2
Assert-Equal $arm64Components[1] 'Microsoft.VisualStudio.Component.VC.Tools.ARM64'

$savedABI = $env:LLGO_ABI
$savedMSYSTEM = $env:MSYSTEM
try {
  $env:LLGO_ABI = 'mingw'
  Assert-Equal (Resolve-LLGoWindowsABI -Architecture amd64) 'mingw'
  $env:LLGO_ABI = 'msvc'
  Assert-Equal (Resolve-LLGoWindowsABI -Architecture arm64) 'msvc'
  $env:LLGO_ABI = $null
  $env:MSYSTEM = 'CLANG64'
  Assert-Equal (Resolve-LLGoWindowsABI -Architecture amd64) 'mingw'
} finally {
  $env:LLGO_ABI = $savedABI
  $env:MSYSTEM = $savedMSYSTEM
}

$dependencyTemporary = Join-Path ([IO.Path]::GetTempPath()) "llgo-dependency-test-$([Guid]::NewGuid())"
try {
  $script:dependencyCalls = 0
  $fakeDependencyPath = Join-Path $dependencyTemporary 'tools'
  New-Item -ItemType Directory -Force $fakeDependencyPath | Out-Null
  function Test-LLGoGoVersion { return $true }
  function Install-LLGoMSVCDependencies {
    param([string]$Root, [string]$Version, [string]$Architecture)
    $script:dependencyCalls++
    return @($fakeDependencyPath)
  }
  $dependencyRoot = Join-Path $dependencyTemporary 'root'
  $firstPaths = @(Install-LLGoWindowsDependencies `
    -Root $dependencyRoot -Version '1.2.3' -Architecture 'amd64' -ABI 'msvc')
  $secondPaths = @(Install-LLGoWindowsDependencies `
    -Root $dependencyRoot -Version '1.2.3' -Architecture 'amd64' -ABI 'msvc')
  Assert-Equal $script:dependencyCalls 1
  Assert-Equal $firstPaths.Count 1
  Assert-Equal $secondPaths.Count 1
} finally {
  if (Test-Path -LiteralPath $dependencyTemporary) {
    Remove-Item -Recurse -Force -LiteralPath $dependencyTemporary
  }
}

if ($IsWindows -or $env:OS -eq 'Windows_NT') {
  $temporary = Join-Path ([IO.Path]::GetTempPath()) "llgo-installer-test-$([Guid]::NewGuid())"
  try {
    $payload = Join-Path $temporary 'payload'
    New-Item -ItemType Directory -Force `
      (Join-Path $payload 'bin'), `
      (Join-Path $payload 'runtime') | Out-Null
    Set-Content -Encoding ascii -LiteralPath (Join-Path $payload 'bin\llgo.exe') -Value 'test'
    Set-Content -Encoding ascii -LiteralPath (Join-Path $payload 'runtime\go.mod') `
      -Value 'module github.com/xgo-dev/llgo/runtime'
    $archive = Join-Path $temporary 'llgo.zip'
    Compress-Archive -Path (Join-Path $payload '*') -DestinationPath $archive

    $root = Join-Path $temporary 'root'
    $null = Install-LLGoRelease `
      -Root $root `
      -Version '1.2.3' `
      -Architecture 'amd64' `
      -ABI 'msvc' `
      -LocalArchive $archive
    $current = Get-Item -Force -LiteralPath (Join-Path $root 'current')
    $bin = Get-Item -Force -LiteralPath (Join-Path $root 'bin')
    if (-not ($current.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
      throw 'current is not a junction'
    }
    if (-not ($bin.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
      throw 'bin is not a junction'
    }
    if (-not (Test-Path -LiteralPath (Join-Path $root 'bin\llgo.exe'))) {
      throw 'the stable executable path does not resolve'
    }
    $null = Install-LLGoRelease `
      -Root $root `
      -Version '1.2.4' `
      -Architecture 'amd64' `
      -ABI 'msvc' `
      -LocalArchive $archive
    if (-not (Test-Path -LiteralPath (Join-Path $root 'versions\1.2.3\windows-amd64-msvc\bin\llgo.exe'))) {
      throw 'installing a new version removed the old version'
    }
  } finally {
    if (Test-Path -LiteralPath $temporary) {
      foreach ($link in @('bin', 'current')) {
        $linkPath = Join-Path $temporary "root\$link"
        if (Get-Item -Force -LiteralPath $linkPath -ErrorAction SilentlyContinue) {
          Remove-Item -Force -LiteralPath $linkPath
        }
      }
      Remove-Item -Recurse -Force -LiteralPath $temporary
    }
  }
}

Write-Host 'install.ps1 tests passed'
