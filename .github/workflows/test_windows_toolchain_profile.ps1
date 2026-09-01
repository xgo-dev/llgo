param(
  [ValidateSet("msvc", "mingw")]
  [string]$Profile = "msvc",
  [Parameter(Mandatory = $true)]
  [ValidateSet("386", "amd64", "arm64")]
  [string]$GoArch
)

$ErrorActionPreference = "Stop"

function Assert-Success([string]$Operation, [int]$ExitCode = $LASTEXITCODE) {
  if ($ExitCode -ne 0) {
    throw "$Operation failed with exit code $ExitCode"
  }
}

function Invoke-NativeCapture([string]$Executable, [object[]]$ArgumentList = @()) {
  # Windows PowerShell 5.1 promotes redirected native stderr to ErrorRecord.
  # Capture each record as its original text under Continue so LLGo's normal
  # -x trace and println stderr remain data; pwsh 7 follows the same path.
  $savedErrorActionPreference = $ErrorActionPreference
  try {
    $ErrorActionPreference = "Continue"
    $output = (& $Executable @ArgumentList 2>&1 |
      ForEach-Object { $_.ToString() }) -join [Environment]::NewLine
    $exitCode = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $savedErrorActionPreference
  }
  return [PSCustomObject]@{ Output = $output; ExitCode = $exitCode }
}

function Assert-NativeOutput([string]$Executable) {
  # Go's implementation-defined println builtin may write to stderr. Capture
  # both streams because this smoke test validates execution, not stream choice.
  $result = Invoke-NativeCapture $Executable
  Assert-Success "Running $Executable" $result.ExitCode
  $output = $result.Output.Trim()
  $want = "windows-$Profile-$GoArch-profile"
  if ($output -ne $want) {
    throw "$Executable printed '$output', want '$want'"
  }
  $dependents = (& llvm-readobj.exe --coff-imports $Executable | Out-String)
  Assert-Success "Inspecting $Executable"
  $forbidden = if ($Profile -eq "msvc") {
    '(?i)(msys-2\.0|cygwin1|libwinpthread)\.dll'
  } else {
    '(?i)(msys-2\.0|cygwin1)\.dll'
  }
  if ($dependents -match $forbidden) {
    throw "$Executable has an unsupported POSIX-emulation dependency:`n$dependents"
  }
}

$sourceDir = Join-Path $env:RUNNER_TEMP "llgo-windows-toolchain-profile"
New-Item -ItemType Directory -Force $sourceDir | Out-Null
@'
module example.com/llgo-windows-toolchain-profile

go 1.27
'@ | Set-Content -Encoding ascii (Join-Path $sourceDir "go.mod")
$mainSource = @'
package main

func main() {
	println("windows-$Profile-$GoArch-profile")
}
'@
$mainSource.Replace('$Profile', $Profile).Replace('$GoArch', $GoArch) |
  Set-Content -Encoding utf8 (Join-Path $sourceDir "main.go")

$llgo = (Get-Command llgo.exe).Source
$profileClangDir = Split-Path (Get-Command clang.exe).Source
foreach ($tool in @("clang", "clang++", "llvm-config")) {
  $path = (Get-Command $tool).Source
  if ($Profile -eq "msvc" -and $path -match '(?i)[\\/](msys64|cygwin)[\\/]') {
    throw "$tool unexpectedly resolves through a POSIX environment: $path"
  }
  if ($Profile -eq "mingw") {
    $profilePattern = switch ($GoArch) {
      "386" {
        if ($tool -eq "llvm-config") { '(?i)[\\/]clang64[\\/]' } else { '(?i)[\\/]llgo-mingw-target-tools[\\/]' }
      }
      "amd64" { '(?i)[\\/]clang64[\\/]' }
      "arm64" { '(?i)[\\/]clangarm64[\\/]' }
    }
    if ($path -notmatch $profilePattern) {
      throw "$tool does not belong to the windows/$GoArch MinGW profile: $path"
    }
  }
}
$pkgConfig = (Get-Command pkg-config).Source
if ($pkgConfig -notmatch '(?i)[\\/]llgo-(msvc|mingw)(-target)?-tools[\\/]pkg-config\.cmd$') {
  throw "pkg-config does not resolve to the profile-local command wrapper: $pkgConfig"
}
$pkgConfigShell = Join-Path (Split-Path $pkgConfig) "pkg-config"
if (-not (Test-Path $pkgConfigShell)) {
  throw "The shell-compatible pkg-config wrapper was not found: $pkgConfigShell"
}
if ($env:PKG_CONFIG -or $env:PKG_CONFIG_PATH) {
  throw "The $Profile profile unexpectedly requires PKG_CONFIG or PKG_CONFIG_PATH"
}
if ($Profile -eq "msvc" -and $env:LLGO_MSYS2_LOCATION) {
  throw "The MSVC lane still exports LLGO_MSYS2_LOCATION"
}
& pkg-config --modversion llvm-21 | Out-Null
Assert-Success "Reading LLVM metadata through the profile-local pkg-config"

$compilerArgs = @()
if ($Profile -eq "msvc") {
  # Official LLVM is an x64 host compiler in every MSVC lane. Apply the
  # activated target explicitly before checking cross-architecture output.
  $compilerArgs = @("--target=$env:LLGO_WINDOWS_TARGET_TRIPLE")
}
$compilerTarget = (& clang @compilerArgs -dumpmachine).Trim()
Assert-Success "Reading the Clang target"
$targetPattern = if ($Profile -eq "msvc") { '-windows-msvc$' } else { '-(windows-gnu|mingw32)$' }
if ($compilerTarget -notmatch $targetPattern) {
  throw "$Profile Clang reports incompatible target $compilerTarget"
}
$targetArch = switch ($GoArch) {
  "386" { "i686" }
  "amd64" { "x86_64|amd64" }
  "arm64" { "aarch64" }
}
if ($compilerTarget -notmatch "^($targetArch)-") {
  throw "$Profile Clang reports $compilerTarget for windows/$GoArch"
}

$savedCC = $env:CC
$savedCXX = $env:CXX
try {
  Remove-Item Env:CC -ErrorAction SilentlyContinue
  Remove-Item Env:CXX -ErrorAction SilentlyContinue

  $powershellExe = Join-Path $sourceDir "powershell.exe"
  Push-Location $sourceDir
  try {
    $result = Invoke-NativeCapture $llgo @("build", "-x", "-o", $powershellExe, ".")
    Assert-Success "Building with unset CC/CXX from PowerShell" $result.ExitCode
    $trace = $result.Output
  } finally {
    Pop-Location
  }
  $canonicalArch = switch ($GoArch) {
    "386" { "i686" }
    "amd64" { "x86_64" }
    "arm64" { "aarch64" }
  }
  $canonicalTarget = if ($Profile -eq "msvc") {
    "$canonicalArch-pc-windows-msvc"
  } else {
    "$canonicalArch-w64-windows-gnu"
  }
  if ($trace -notmatch [regex]::Escape($canonicalTarget)) {
    throw "Unset CC/CXX did not select the canonical $Profile target:`n$trace"
  }
  Assert-NativeOutput $powershellExe

  $cmdExe = Join-Path $sourceDir "cmd.exe"
  $cmdLine = 'cd /d "' + $sourceDir + '" && "' + $llgo + '" build -o "' + $cmdExe + '" .'
  & $env:ComSpec /d /s /c $cmdLine
  Assert-Success "Building with unset CC/CXX from cmd.exe"
  Assert-NativeOutput $cmdExe

  if ($Profile -eq "msvc") {
    . (Join-Path $env:GITHUB_WORKSPACE ".github\windows\msvc-target.ps1")
    $target = Get-LLGoWindowsMSVCTarget -GoArch $GoArch
    $installPath = Find-LLGoVisualStudio2022 -GoArch $GoArch
    $vsDevCmd = Join-Path $installPath "Common7\Tools\VsDevCmd.bat"
    $vsDevExe = Join-Path $sourceDir "vsdev.exe"
    # VsDevCmd prepends Visual Studio's optional bundled Clang. That compiler
    # does not necessarily install compiler-rt for every selected target (the
    # ARM64 lane is one example). Keep the standalone full-target LLVM profile
    # selected by setup-deps ahead of it while retaining DevShell's SDK, CRT,
    # and linker environment.
    $vsDevLine = 'call "' + $vsDevCmd + '" -no_logo -arch=' + $target.VisualStudio +
      ' -host_arch=x64 && set "PATH=' + $profileClangDir + ';%PATH%" && set "CC=" && set "CXX=" && cd /d "' + $sourceDir +
      '" && "' + $llgo + '" build -o "' + $vsDevExe + '" .'
    & $env:ComSpec /d /s /c $vsDevLine
    Assert-Success "Building from a fresh Visual Studio Developer Shell"
    Assert-NativeOutput $vsDevExe
  }
} finally {
  if ($null -eq $savedCC) { Remove-Item Env:CC -ErrorAction SilentlyContinue } else { $env:CC = $savedCC }
  if ($null -eq $savedCXX) { Remove-Item Env:CXX -ErrorAction SilentlyContinue } else { $env:CXX = $savedCXX }
}
