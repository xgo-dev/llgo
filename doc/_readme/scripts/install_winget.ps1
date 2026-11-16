$ErrorActionPreference = 'Stop'
winget install --id XGo.LLGo.MSVC --exact --source winget --scope user
if ($LASTEXITCODE -ne 0) { throw 'WinGet installation failed' }
