# Run in Windows PowerShell: Appx deployment needs its native module.
# Install the same signed, checksummed release on Server 2025 and ARM64.
$ErrorActionPreference = 'Stop'
$Version = '1.29.380'
$assets = @{
  'Microsoft.DesktopAppInstaller_8wekyb3d8bbwe.msixbundle' = '65dea9c01ce08ee7b763366b27c0e651f97db857c11ca9b9c301826c10092f2e'
  'DesktopAppInstaller_Dependencies.zip' = 'ba875afe9d190f61218985ac0292a99d1db710bf93e13c68944ca9d89f0d82d1'
}
$temp = Join-Path $env:RUNNER_TEMP ('winget-setup-' + [Guid]::NewGuid())
New-Item -ItemType Directory $temp | Out-Null
try {
  foreach ($name in $assets.Keys) {
    $path = Join-Path $temp $name
    & curl.exe --fail --location --retry 5 --retry-all-errors --output $path "https://github.com/microsoft/winget-cli/releases/download/v$Version/$name"
    if ($LASTEXITCODE -ne 0) { throw "Downloading $name failed" }
    if ((Get-FileHash -Algorithm SHA256 $path).Hash -ne $assets[$name]) { throw "Invalid SHA-256 for $name" }
  }
  $dependencies = Join-Path $temp 'dependencies'
  Expand-Archive -LiteralPath (Join-Path $temp 'DesktopAppInstaller_Dependencies.zip') -DestinationPath $dependencies
  $arch = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  $packages = @(Get-ChildItem (Join-Path $dependencies "$arch/*.appx"))
  if ($packages.Count -ne 3) { throw "Missing WinGet dependencies for $arch" }
  # Runner images can contain an unregistered Store stub with a newer package
  # version. Deploy the pinned bundle rather than registering that broken stub.
  Add-AppxPackage -Path (Join-Path $temp 'Microsoft.DesktopAppInstaller_8wekyb3d8bbwe.msixbundle') `
    -DependencyPath $packages.FullName -ForceUpdateFromAnyVersion -ForceApplicationShutdown
} finally {
  Remove-Item -LiteralPath $temp -Recurse -Force
}
$package = Get-AppxPackage -Name Microsoft.DesktopAppInstaller
if (@($package).Count -ne 1) { throw 'Expected one registered WinGet package' }
$env:PATH = "$($package.InstallLocation);$env:PATH"
$package.InstallLocation | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append
$actual = (& winget.exe --version | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $actual -ne "v$Version") { throw "Expected WinGet v$Version, found $actual" }
