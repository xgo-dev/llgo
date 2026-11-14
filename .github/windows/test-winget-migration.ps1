# Exercise the portable package IDs with the real WinGet client and tested ZIP.
param(
  [Parameter(Mandatory = $true)][string]$Archive,
  [Parameter(Mandatory = $true)][ValidateSet('amd64', 'arm64', '386')][string]$GoArch
)
$ErrorActionPreference = 'Stop'
$archivePath = (Resolve-Path -LiteralPath $Archive).Path
$winget = (Get-Command winget.exe).Source
$installerArch = @{ amd64 = 'x64'; arm64 = 'arm64'; '386' = 'x86' }[$GoArch]
$temp = Join-Path $env:RUNNER_TEMP ('llgo-winget-' + [Guid]::NewGuid())
New-Item -ItemType Directory $temp | Out-Null
$ids = @('XGo.LLGo.MSVC', 'XGo.LLGo.MinGW', 'XGo.LLGo')

function Invoke-WinGet([string[]]$Arguments) {
  & $winget @Arguments
  if ($LASTEXITCODE -ne 0) { throw "WinGet failed ($LASTEXITCODE): $Arguments" }
}

function Get-PortableRegistration([string]$Identifier) {
  # Read the public uninstall registration; never rewrite WinGet's state.
  $roots = @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall',
             'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall',
             'HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
  foreach ($root in $roots) {
    Get-ChildItem $root -ErrorAction SilentlyContinue | ForEach-Object {
      $entry = Get-ItemProperty $_.PSPath
      if ($entry.WinGetPackageIdentifier -eq $Identifier) { $entry }
    }
  }
}

function Assert-InstalledCompiler([string]$Alias) {
  if (-not (Test-Path $Alias)) { throw 'The unified portable alias is missing' }
  $compilerVersion = (& $Alias version | Out-String).Trim()
  if ($LASTEXITCODE -ne 0 -or $compilerVersion -notmatch " windows/$GoArch$") {
    throw "Unexpected migrated compiler: $compilerVersion"
  }
  Write-Host $compilerVersion
}

foreach ($id in $ids) {
  if (@(Get-PortableRegistration $id).Count) { throw "Refusing to replace an existing $id installation" }
}

# Local manifests use a loopback HTTP server; every download retains the real
# release ZIP hash and the normal archive/security checks.
Copy-Item $archivePath (Join-Path $temp 'llgo.zip')
$checksum = (Get-FileHash -Algorithm SHA256 $archivePath).Hash.ToLowerInvariant()
$serverScript = Join-Path $temp 'server.py'
@'
import functools, http.server, pathlib, sys
root = pathlib.Path(sys.argv[1])
handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=str(root))
with http.server.ThreadingHTTPServer(('127.0.0.1', 0), handler) as server:
    (root / 'port').write_text(str(server.server_address[1]))
    server.serve_forever()
'@ | Set-Content -Encoding utf8 $serverScript
$python = (Get-Command python).Source
$server = Start-Process -FilePath $python -ArgumentList @('-u', ('"' + $serverScript + '"'), ('"' + $temp + '"')) -PassThru -NoNewWindow
try {
  $deadline = [DateTime]::UtcNow.AddSeconds(15)
  $port = ''
  while ($port -notmatch '^[0-9]+$') {
    if ($server.HasExited -or [DateTime]::UtcNow -gt $deadline) { throw 'WinGet fixture server did not start' }
    if (Test-Path "$temp/port") { $port = ([string](Get-Content "$temp/port" -Raw)).Trim() }
    if ($port -notmatch '^[0-9]+$') { Start-Sleep -Milliseconds 100 }
  }
  function Write-Candidate([string]$Identifier, [string]$Version, [string]$Name, [string]$ProductCode = '') {
    $directory = Join-Path $temp $Name
    New-Item -ItemType Directory -Force $directory | Out-Null
    $productCodeLine = if ($ProductCode) { 'ProductCode: ' + (ConvertTo-Json -InputObject $ProductCode -Compress) } else { '' }
    @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.singleton.1.12.0.schema.json
PackageIdentifier: $Identifier
PackageVersion: '$Version'
PackageLocale: en-US
Publisher: XGo
PackageName: LLGo
License: Apache-2.0
ShortDescription: LLGo portable migration fixture
InstallerType: zip
NestedInstallerType: portable
NestedInstallerFiles:
- RelativeFilePath: bin/llgo.exe
  PortableCommandAlias: llgo
UpgradeBehavior: uninstallPrevious
$productCodeLine
Installers:
- Architecture: $installerArch
  InstallerUrl: http://127.0.0.1:$port/llgo.zip
  InstallerSha256: $checksum
ManifestType: singleton
ManifestVersion: 1.12.0
"@ | Set-Content -Encoding utf8 "$directory/$Identifier.yaml"
    return $directory
  }
  Invoke-WinGet -Arguments @('settings', '--enable', 'LocalManifestFiles')
  $new = Write-Candidate 'XGo.LLGo' '2.0.0' 'unified'
  Invoke-WinGet -Arguments @('validate', '--manifest', $new)
  $next = $null
  $unifiedProductCode = ''
  foreach ($legacy in @('XGo.LLGo.MSVC', 'XGo.LLGo.MinGW')) {
    $old = Write-Candidate $legacy '1.0.0' $legacy
    Invoke-WinGet -Arguments @('validate', '--manifest', $old)
    Invoke-WinGet -Arguments @('install', '--manifest', $old, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity')
    $registration = @(Get-PortableRegistration $legacy)
    if ($registration.Count -ne 1) { throw "$legacy did not register" }

    # Local manifests have no public-catalog correlation. Select the actual
    # public ARP ProductCode; published installs use the documented --id form.
    Invoke-WinGet -Arguments @('uninstall', '--product-code', $registration[0].PSChildName, '--exact', '--accept-source-agreements', '--disable-interactivity')
    Invoke-WinGet -Arguments @('install', '--manifest', $new, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity')
    $registration = @(Get-PortableRegistration 'XGo.LLGo')
    if (@(Get-PortableRegistration $legacy).Count -ne 0 -or $registration.Count -ne 1) {
      throw "Migration from $legacy did not leave exactly one unified registration"
    }
    if (-not $next) {
      $unifiedProductCode = $registration[0].PSChildName
      $next = Write-Candidate 'XGo.LLGo' '2.0.1' 'next' $unifiedProductCode
      Invoke-WinGet -Arguments @('validate', '--manifest', $next)
    } elseif ($registration[0].PSChildName -ne $unifiedProductCode) {
      throw 'The unified package registration changed between migrations'
    }
    $alias = Join-Path $env:LOCALAPPDATA 'Microsoft/WinGet/Links/llgo.exe'
    Assert-InstalledCompiler $alias

    Invoke-WinGet -Arguments @('upgrade', '--manifest', $next, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity')
    $registration = @(Get-PortableRegistration 'XGo.LLGo')
    if ($registration.Count -ne 1 -or $registration[0].DisplayVersion -ne '2.0.1') {
      throw 'Subsequent upgrades must use only the unified ID'
    }
    Assert-InstalledCompiler $alias
    Invoke-WinGet -Arguments @('uninstall', '--product-code', $registration[0].PSChildName, '--exact', '--accept-source-agreements', '--disable-interactivity')
    if (Test-Path $alias) { throw 'Uninstall left a stale llgo alias' }
    Write-Host "Passed windows/$GoArch $legacy -> XGo.LLGo -> next unified version -> uninstall"
  }
} finally {
  foreach ($id in $ids) {
    foreach ($entry in @(Get-PortableRegistration $id)) {
      & $winget uninstall --product-code $entry.PSChildName --exact --accept-source-agreements --disable-interactivity
    }
  }
  if (-not $server.HasExited) { Stop-Process -Id $server.Id -Force }
  Remove-Item -LiteralPath $temp -Recurse -Force
}
