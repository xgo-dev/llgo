# Use the same real WinGet client on Windows Server 2025 and native ARM64 CI.
param([string]$Version = '1.29.380')
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+$') { throw 'Invalid WinGet version' }
Install-Module -Name Microsoft.WinGet.Client -Force -AcceptLicense -Repository PSGallery -Scope CurrentUser
Repair-WinGetPackageManager -Version $Version -AllUsers
$actual = (& winget.exe --version | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $actual -ne "v$Version") { throw "Expected WinGet v$Version, found $actual" }
