param(
  [ValidateSet('amd64', 'arm64')][string]$GoArch,
  [ValidateSet('path', 'symlink')][string]$Mode,
  [ValidateSet('install', 'build', 'uninstall')][string]$Phase
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-lib.ps1')
$evidence = Join-Path $env:RUNNER_TEMP 'winget-ab'
New-Item -ItemType Directory -Force $evidence | Out-Null
$state = Join-Path $evidence 'installed.json'
$winget = (Get-Command winget.exe).Source
function Invoke-WinGet([string[]]$Arguments) {
  & $winget @Arguments
  if ($LASTEXITCODE -ne 0) { throw "winget $Arguments failed: $LASTEXITCODE" }
}
Remove-Item Env:LLGO_ROOT -ErrorAction SilentlyContinue
if ($Phase -eq 'uninstall') {
  # Even a failed version probe must uninstall its successfully installed package.
  if (-not (Test-Path (Join-Path $evidence 'install-started'))) { return }
  Invoke-WinGet @('uninstall', '--name', 'LLGo MinGW', '--exact', '--silent', '--disable-interactivity')
  $env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('PATH', 'User')
  if (Get-Command llgo.exe -ErrorAction SilentlyContinue) { throw 'llgo remains after uninstall' }
  Write-Host "WINGET_UNINSTALL_OK $GoArch $Mode"
  return
}
if ($Phase -eq 'install') {
  $manifest = Join-Path $evidence 'manifest'
  Copy-Item (Join-Path $PSScriptRoot 'winget-candidates/manifests/x/XGo/LLGo/MinGW/1.0.9') $manifest -Recurse
  if ($Mode -eq 'symlink') {
    $installer = Join-Path $manifest 'XGo.LLGo.MinGW.installer.yaml'
    ((Get-Content -Raw $installer) -replace '(?m)^ArchiveBinariesDependOnPath: true\r?\n', '') | Set-Content -NoNewline $installer
    if ((Get-Content -Raw $installer) -match 'ArchiveBinariesDependOnPath') { throw 'Failed to remove the PATH setting' }
  }
  Invoke-WinGet @('--info')
  Invoke-WinGet @('settings', '--enable', 'LocalManifestFiles')
  Invoke-WinGet @('validate', '--manifest', $manifest)
  Set-Content (Join-Path $evidence 'install-started') 'true'
  Invoke-WinGet @('install', '--manifest', $manifest, '--architecture', @{amd64='x64';arm64='arm64'}[$GoArch], '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity')
  Invoke-WinGet @('list', '--name', 'LLGo MinGW', '--exact', '--accept-source-agreements', '--disable-interactivity')
  Write-Host 'Command discovery before process PATH refresh:'
  Get-Command llgo -All -ErrorAction SilentlyContinue | Format-List CommandType,Source,Definition
  $env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('PATH', 'User')
  $llgo = (Get-Command llgo.exe).Source
  $item = Get-Item -LiteralPath $llgo
  $roots = @(Get-ChildItem (Join-Path $env:LOCALAPPDATA 'Microsoft/WinGet/Packages') -Filter release.json -File -Recurse | Where-Object { $_.FullName -like '*XGo.LLGo.MinGW*' })
  if ($roots.Count -ne 1) { throw "Expected one installed release.json, got $($roots.Count)" }
  $releaseRoot = $roots[0].DirectoryName
  $metadata = Get-Content -Raw $roots[0].FullName | ConvertFrom-Json
  if ($metadata.version -ne '1.0.9' -or $metadata.goarch -ne $GoArch -or $metadata.abi -ne 'mingw') { throw 'Incorrect installed metadata' }
  @{ command=$llgo; root=$releaseRoot; linkType=$item.LinkType; target=$item.Target; path=$env:PATH; mode=$Mode; arch=$GoArch } | ConvertTo-Json | Tee-Object -FilePath $state
  if ($Mode -eq 'symlink' -and $item.LinkType -ne 'SymbolicLink') { throw 'Candidate did not create a symbolic link; no A/B comparison' }
  if ($Mode -eq 'path' -and $item.LinkType -eq 'SymbolicLink') { throw 'Baseline unexpectedly created a symbolic link' }
  # Exclude MSYS2/LLVM and even the real package bin in symlink mode.
  $env:PATH = (Split-Path $llgo) + ';' + (Join-Path $env:SystemRoot 'System32') + ';' + $env:SystemRoot
  $version = (Invoke-ReleaseCapture $llgo @('version')).Trim()
  if ($version -ne "llgo v1.0.9 windows/$GoArch") { throw "Unexpected version: $version" }

  $real = Join-Path $releaseRoot 'bin/llgo.exe'
  $systemPath = (Join-Path $env:SystemRoot 'System32') + ';' + $env:SystemRoot
  $linkDir = Join-Path $evidence 'probe-link'
  New-Item -ItemType Directory -Force $linkDir | Out-Null
  $link = Join-Path $linkDir 'llgo.exe'
  New-Item -ItemType SymbolicLink -Path $link -Target $real | Out-Null
  $probes = @()
  foreach ($case in @(
    @{name='direct-system-path'; exe=$real; path=$systemPath},
    @{name='command-installed-bin'; exe='llgo'; path=(Split-Path $real)+';'+$systemPath},
    @{name='symlink-system-path'; exe=$link; path=$systemPath},
    @{name='symlink-with-real-bin'; exe=$link; path=(Split-Path $real)+';'+$systemPath}
  )) {
    foreach ($kind in @('help','version','flag-version')) {
      $env:PATH = $case.path
      $argsForProbe = switch($kind) { 'help' { @() }; 'version' { @('version') }; 'flag-version' { @('--version') } }
      # ProcessStartInfo captures nonzero exits and stderr without hiding them.
      $start = [Diagnostics.ProcessStartInfo]::new()
      $start.FileName = if ($case.exe -eq 'llgo') { (Get-Command llgo.exe).Source } else { $case.exe }
      $start.Arguments = $argsForProbe -join ' '
      $start.UseShellExecute = $false
      $start.RedirectStandardOutput = $true
      $start.RedirectStandardError = $true
      $proc = [Diagnostics.Process]::Start($start)
      $stdout = $proc.StandardOutput.ReadToEnd(); $stderr = $proc.StandardError.ReadToEnd()
      $proc.WaitForExit()
      $result = @{case=$case.name; command=$start.FileName; argument=$kind; exit=$proc.ExitCode; stdout=$stdout; stderr=$stderr}
      $probes += $result
      Write-Host ($result | ConvertTo-Json -Compress)
      if ($kind -ne 'flag-version' -and $case.name -ne 'symlink-system-path' -and $proc.ExitCode -ne 0) { throw "Required startup probe failed: $($case.name) $kind" }
    }
  }
  $probes | ConvertTo-Json -Depth 5 | Set-Content (Join-Path $evidence 'startup-probes.json')
  Remove-Item -LiteralPath $link -Force
  Write-Host "WINGET_CLEAN_START_OK $GoArch $Mode $version"
  return
}
$installed = Get-Content -Raw $state | ConvertFrom-Json
$llgo = $installed.command
$releaseRoot = $installed.root
$readObj = Join-Path $releaseRoot 'crosscompile/clang/bin/llvm-readobj.exe'
$Profile = 'mingw'
$go = (Get-Command go).Source
$env:LLGO_BUILD_CACHE = 'off'
$work = Join-Path $env:RUNNER_TEMP ('llgo-release-smoke-' + [Guid]::NewGuid())
New-Item -ItemType Directory $work | Out-Null
Push-Location $work
try {
  @'
module release-smoke

go 1.27

require github.com/goplus/lib v0.5.1
'@ | Set-Content -Encoding utf8 'go.mod'
  @'
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
'@ | Set-Content -Encoding utf8 'main.go'
  & $go mod tidy
  if ($LASTEXITCODE -ne 0) { throw 'Preparing the native release smoke test failed' }
  $program = Join-Path $work 'hello.exe'
  $trace = Invoke-ReleaseCapture $llgo @('build', '-x', '-o', $program, '.')
  $trace | Set-Content (Join-Path $evidence 'native-build.log')
  $arch = @{ amd64 = 'x86_64'; arm64 = 'aarch64' }[$GoArch]
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

  New-Item -ItemType Directory 'embedded' | Out-Null
  Set-Content -Encoding utf8 'embedded/main.go' 'package main; func main() {}'
  $firmware = Join-Path $work 'embedded/demo.out'
  $trace = Invoke-ReleaseCapture $llgo @('build', '-x', '-target', 'esp32-coreboard-v2', '-o', $firmware, './embedded')
  $trace | Set-Content (Join-Path $evidence 'esp-build.log')
  if (-not (Test-Path "$firmware.elf")) { throw 'The integrated ESP Clang did not produce firmware' }
  # Go traces can use either slash style on Windows.
  if (-not $trace.Replace('\', '/').Contains($releaseRoot.Replace('\', '/') + '/crosscompile/clang')) {
    throw "The embedded build did not use the extracted release toolchain`n$trace"
  }
  Write-Host "Passed native Go/C/C++ and integrated ESP builds for windows/$GoArch/$Profile"
} finally {
  Pop-Location
  Remove-Item -LiteralPath $work -Recurse -Force
}
