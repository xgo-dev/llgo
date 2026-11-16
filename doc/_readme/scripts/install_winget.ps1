$ErrorActionPreference = 'Stop'
winget install XGo.LLGo.MSVC
if ($LASTEXITCODE -ne 0) { throw 'WinGet installation failed' }
