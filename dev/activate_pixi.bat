@echo off
set "GOFLAGS=-tags=byollvm"
set "CGO_ENABLED=1"
set "CC=clang -fuse-ld=lld -fms-runtime-lib=dll"
set "CXX=clang++ -fuse-ld=lld -fms-runtime-lib=dll"
set "LLVM_CONFIG=llvm-config"
set "CGO_CXXFLAGS=-std=c++17"
set "LLGO_ROOT=%~dp0.."
for /f "delims=" %%F in ('llvm-config --cflags') do set "CGO_CPPFLAGS=%%F"
for /f "usebackq delims=" %%F in (`powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%PIXI_PROJECT_ROOT%\pixi_windows_ldflags.ps1"`) do set "CGO_LDFLAGS=%%F"
if not defined CGO_LDFLAGS exit /b 1
