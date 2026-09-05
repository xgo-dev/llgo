param(
  [ValidateRange(5, 120)]
  [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"
$goArch = $env:LLGO_WINDOWS_ARCH
$targetABI = $env:LLGO_WINDOWS_ABI
if ($goArch -notin @("386", "amd64", "arm64") -or $targetABI -notin @("msvc", "mingw")) {
  throw "Activate a supported LLGO_WINDOWS_ARCH and LLGO_WINDOWS_ABI before this test"
}
if ($env:LLGO_WINDOWS_TARGET_ACTIVE -ne "1") {
  throw "The Windows target environment has not been activated"
}

$root = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).ProviderPath
$temporaryRoot = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [IO.Path]::GetTempPath() }
$out = Join-Path $temporaryRoot ("llgo-windows-cpuprof-" + [Guid]::NewGuid())
New-Item -ItemType Directory $out | Out-Null

function Invoke-BoundedNative([string]$Executable, [string[]]$NativeArgs, [string]$Name) {
  $stdout = Join-Path $out "$Name.stdout"
  $stderr = Join-Path $out "$Name.stderr"
  # Start-Process joins ArgumentList with spaces, including on PowerShell 7.
  # None of these arguments ends in a backslash or contains an embedded quote.
  $quotedArgs = @($NativeArgs | ForEach-Object { '"' + $_ + '"' })
  $startArgs = @{
    FilePath = $Executable
    PassThru = $true
    NoNewWindow = $true
    RedirectStandardOutput = $stdout
    RedirectStandardError = $stderr
  }
  if ($quotedArgs.Count -ne 0) { $startArgs.ArgumentList = $quotedArgs }
  $process = Start-Process @startArgs
  # Windows PowerShell 5.1 otherwise loses ExitCode when WaitForExit observes
  # a process that has already exited. Retain its native handle until Dispose.
  $null = $process.Handle
  try {
    if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
      # Kill only the process tree created above, including a stuck linker.
      & taskkill.exe /PID $process.Id /T /F | Out-Null
      $null = $process.WaitForExit(5000)
      throw "$Name exceeded ${TimeoutSeconds}s; diagnostics: $out"
    }
    $process.WaitForExit()
    $output = (Get-Content -Raw $stdout) + (Get-Content -Raw $stderr)
    Write-Host $output
    if ($process.ExitCode -ne 0) {
      throw "$Name failed with exit code $($process.ExitCode); diagnostics: $out"
    }
    return $output
  } finally {
    $process.Dispose()
  }
}

$compilerArgs = @("-O2", "-fno-omit-frame-pointer", "-mno-omit-leaf-frame-pointer", "-fno-optimize-sibling-calls")
if ($targetABI -eq "msvc") {
  if (-not $env:LLGO_WINDOWS_TARGET_TRIPLE) {
    throw "The MSVC target triple was not exported by target activation"
  }
  $clang = (Get-Command clang.exe).Source
  $compilerArgs += @("--target=$env:LLGO_WINDOWS_TARGET_TRIPLE", "-fuse-ld=lld", "-fms-runtime-lib=dll")
} elseif ($goArch -eq "386") {
  # Bypass the .cmd wrapper: Start-Process must launch a real native process,
  # and the unqualified clang.exe in the host toolchain targets x64.
  $clang = Join-Path $env:LLGO_MINGW_TARGET_BIN "i686-w64-mingw32-clang.exe"
  if (-not (Test-Path $clang)) { throw "The activated x86 MinGW compiler is missing: $clang" }
} else {
  $clang = (Get-Command clang.exe).Source
}

$binary = Join-Path $out "windows-cpuprof-context.exe"
$source = Join-Path $root "dev\tests\windows-cpuprof\capture.c"
$compilerArgs += @($source, "-lkernel32", "-o", $binary)
$null = Invoke-BoundedNative -Executable $clang -NativeArgs $compilerArgs -Name "compile"

$readObj = (Get-Command llvm-readobj.exe).Source
$headers = Invoke-BoundedNative -Executable $readObj -NativeArgs @("--file-headers", $binary) -Name "headers"
$machine = switch ($goArch) {
  "386" { "IMAGE_FILE_MACHINE_I386" }
  "amd64" { "IMAGE_FILE_MACHINE_AMD64" }
  "arm64" { "IMAGE_FILE_MACHINE_ARM64" }
}
if (-not $headers.Contains($machine)) {
  throw "The profiler regression binary is not windows/$goArch"
}
$result = Invoke-BoundedNative -Executable $binary -NativeArgs @() -Name "capture"
if (-not $result.Contains("Windows CPU profiler CONTEXT regression: PASS")) {
  throw "The profiler regression did not report success; diagnostics: $out"
}
Write-Host "Windows CPU profiler regression passed ($targetABI/$goArch); diagnostics: $out"
