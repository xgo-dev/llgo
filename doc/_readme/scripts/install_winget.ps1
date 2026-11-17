$ErrorActionPreference = 'Stop'
# Use the legacy ID until the unified XGo.LLGo package is published.
winget install XGo.LLGo.MSVC
if ($LASTEXITCODE -ne 0) { throw 'WinGet installation failed' }
