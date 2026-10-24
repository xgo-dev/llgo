# Cross-platform regression tests using real, minimal COFF images. No Windows
# executable is run, so this also works with LLVM and PowerShell on macOS/Linux.
param(
  [string]$Clang = 'clang',
  [string]$Linker = 'lld-link',
  [string]$ReadObj = 'llvm-readobj'
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-lib.ps1')
$temporary = Join-Path ([IO.Path]::GetTempPath()) ('llgo-release-pe-' + [Guid]::NewGuid())
New-Item -ItemType Directory $temporary | Out-Null

function Invoke-TestTool([string]$Tool, [string[]]$ToolArguments) {
  & $Tool @ToolArguments
  if ($LASTEXITCODE -ne 0) { throw "$Tool failed: $ToolArguments" }
}

function Expect-Failure([scriptblock]$Operation, [string]$Pattern) {
  try { & $Operation } catch {
    if ($_.Exception.Message -notmatch $Pattern) { throw }
    return
  }
  throw "Expected failure matching $Pattern"
}

try {
  $systemDirectory = Join-Path $temporary 'system'
  New-Item -ItemType Directory $systemDirectory | Out-Null
  foreach ($goArch in @('amd64', 'arm64', '386')) {
    $source = Join-Path $temporary $goArch
    $stage = Join-Path $temporary "$goArch stage"
    New-Item -ItemType Directory $source, $stage | Out-Null
    $arch = @{ amd64 = 'x86_64'; arm64 = 'aarch64'; '386' = 'i686' }[$goArch]
    $machine = @{ amd64 = 'x64'; arm64 = 'arm64'; '386' = 'x86' }[$goArch]
    Set-Content (Join-Path $source 'leaf.c') '__declspec(dllexport) int leaf(void) { return 42; }'
    Set-Content (Join-Path $source 'middle.c') '__declspec(dllimport) int leaf(void); __declspec(dllexport) int middle(void) { return leaf(); }'
    Set-Content (Join-Path $source 'main.c') '__declspec(dllimport) int middle(void); int entry(void) { return middle(); }'
    foreach ($name in @('leaf', 'middle', 'main')) {
      Invoke-TestTool $Clang @("--target=$arch-pc-windows-msvc", '-ffreestanding', '-fno-stack-protector', '-c',
        (Join-Path $source "$name.c"), '-o', (Join-Path $source "$name.obj"))
    }
    Invoke-TestTool $Linker @('/dll', '/noentry', '/nodefaultlib', "/machine:$machine",
      "/out:$source/leaf.dll", "/implib:$source/leaf.lib", "$source/leaf.obj")
    Invoke-TestTool $Linker @('/dll', '/noentry', '/nodefaultlib', "/machine:$machine",
      "/out:$source/middle.dll", "/implib:$source/middle.lib", "$source/middle.obj", "$source/leaf.lib")
    $executable = Join-Path $stage 'llgo.exe'
    Invoke-TestTool $Linker @('/entry:entry', '/subsystem:console', '/nodefaultlib', "/machine:$machine",
      "/out:$executable", "$source/main.obj", "$source/middle.lib")
    $arguments = @{
      ReadObj = $ReadObj; Executable = $executable; GoArch = $goArch; Profile = 'mingw'
      SourceDirectories = @($source); SystemDirectory = $systemDirectory
    }
    Copy-ReleaseDLLs @arguments
    foreach ($dll in @('leaf.dll', 'middle.dll')) {
      if (-not (Test-Path (Join-Path $stage $dll))) { throw "Transitive DLL $dll was not copied" }
    }
    Copy-ReleaseDLLs @arguments -CheckOnly

    # An import must not be satisfied by the build SDK during artifact checks.
    Remove-Item (Join-Path $stage 'leaf.dll')
    Expect-Failure { Copy-ReleaseDLLs @arguments -CheckOnly } 'leaf.dll.*absent'
    Copy-ReleaseDLLs @arguments
    if ($goArch -eq '386' -and [Environment]::Is64BitOperatingSystem) {
      # A DLL present only in the 64-bit OS directory must not satisfy a
      # 32-bit release's dependency audit on a 64-bit build machine.
      $savedSystemRoot = $env:SystemRoot
      try {
        $env:SystemRoot = Join-Path $temporary 'Windows'
        $system32 = Join-Path $env:SystemRoot 'System32'
        $systemX86 = Join-Path $env:SystemRoot 'SysWOW64'
        New-Item -ItemType Directory -Force $system32, $systemX86 | Out-Null
        Remove-Item (Join-Path $stage 'leaf.dll')
        Copy-Item (Join-Path $source 'leaf.dll') $system32
        $audit = $arguments.Clone()
        $audit.Remove('SystemDirectory')
        Expect-Failure { Copy-ReleaseDLLs @audit -CheckOnly } 'leaf.dll.*absent'
        Copy-Item (Join-Path $source 'leaf.dll') $systemX86
        Copy-ReleaseDLLs @audit -CheckOnly
      } finally {
        $env:SystemRoot = $savedSystemRoot
      }
      Copy-ReleaseDLLs @arguments
    }
    $otherArch = if ($goArch -eq 'amd64') { 'arm64' } else { 'amd64' }
    Expect-Failure {
      Assert-ReleasePE -ReadObj $ReadObj -Path $executable -GoArch $otherArch -Profile 'mingw'
    } 'PE machine'

    # A DLL with the right name but the wrong architecture must also fail.
    if ($goArch -eq 'arm64') {
      Copy-Item (Join-Path $temporary 'amd64/leaf.dll') (Join-Path $stage 'leaf.dll') -Force
      Expect-Failure { Copy-ReleaseDLLs @arguments -CheckOnly } 'PE machine'
    }

    # Real PE import tables, rather than a mocked parser, exercise the runtime
    # boundary shared by the builder and clean-runner artifact checks.
    foreach ($dll in @('msys-2.0.dll', 'libc++.dll')) {
      Invoke-TestTool $Linker @('/dll', '/noentry', '/nodefaultlib', "/machine:$machine",
        "/out:$source/$dll", "/implib:$source/forbidden.lib", "$source/middle.obj", "$source/leaf.lib")
      Invoke-TestTool $Linker @('/entry:entry', '/subsystem:console', '/nodefaultlib', "/machine:$machine",
        "/out:$stage/forbidden.exe", "$source/main.obj", "$source/forbidden.lib")
      Expect-Failure {
        Assert-ReleasePE -ReadObj $ReadObj -Path "$stage/forbidden.exe" -GoArch $goArch -Profile 'msvc'
      } 'runtime outside'
      if ($dll -eq 'libc++.dll') {
        $null = Assert-ReleasePE -ReadObj $ReadObj -Path "$stage/forbidden.exe" -GoArch $goArch -Profile 'mingw'
      } else {
        Expect-Failure {
          Assert-ReleasePE -ReadObj $ReadObj -Path "$stage/forbidden.exe" -GoArch $goArch -Profile 'mingw'
        } 'runtime outside'
      }
    }
    Write-Host "Passed $goArch DLL closure, relocation, missing dependency, architecture, and ABI checks"
  }
} finally {
  Remove-Item -LiteralPath $temporary -Recurse -Force
}
