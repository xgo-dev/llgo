param(
  [ValidateSet('amd64', 'arm64')][string]$GoArch,
  [ValidateSet('MSVC', 'MinGW')][string]$Profile
)
$ErrorActionPreference = 'Stop'
$identifier = "XGo.LLGo.$Profile"
$manifest = Join-Path $PSScriptRoot "winget-candidates/manifests/x/XGo/LLGo/$Profile/1.0.3"
$winget = (Get-Command winget.exe).Source
function Invoke-WinGet([string[]]$Arguments) {
  & $winget @Arguments
  if ($LASTEXITCODE -ne 0) { throw "winget $Arguments failed: $LASTEXITCODE" }
}
Invoke-WinGet @('--info')
Invoke-WinGet @('settings', '--enable', 'LocalManifestFiles')
Invoke-WinGet @('validate', '--manifest', $manifest)
Remove-Item Env:LLGO_ROOT -ErrorAction SilentlyContinue
Invoke-WinGet @('install', '--manifest', $manifest, '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity')
try {
  Invoke-WinGet @('list', '--name', "LLGo ($Profile)", '--exact', '--accept-source-agreements', '--disable-interactivity')
  # Read the updated persistent PATH rather than retaining the pre-install one.
  $env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('PATH', 'User')
  $llgo = (Get-Command llgo.exe).Source
  $output = (& $llgo version | Out-String).Trim()
  if ($LASTEXITCODE -ne 0 -or $output -ne "llgo v1.0.3 windows/$GoArch") {
    throw "Installed compiler returned: $output"
  }
  Write-Host "Installed command: $llgo"
  Write-Host $output
} finally {
  Invoke-WinGet @('uninstall', '--name', "LLGo ($Profile)", '--exact', '--silent', '--disable-interactivity')
}
$env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('PATH', 'User')
if (Get-Command llgo.exe -ErrorAction SilentlyContinue) { throw 'The llgo command remains after uninstall' }
Write-Host "WINGET_LIFECYCLE_OK $identifier windows/$GoArch"
