$script:LLGoRepository = 'xgo-dev/llgo'
$script:LLGoGoVersion = '1.27.0'
$script:LLGoLLVMVersion = '22.1.8'

function ConvertTo-LLGoArchitecture {
  param([Parameter(Mandatory = $true)][string]$Architecture)

  switch ($Architecture.ToLowerInvariant()) {
    { $_ -in @('x64', 'amd64', 'x86_64') } { return 'amd64' }
    { $_ -in @('arm64', 'aarch64') } { return 'arm64' }
    { $_ -in @('x86', 'i386', 'i486', 'i586', 'i686', '386') } { return '386' }
    default { throw "Unsupported Windows architecture: $Architecture" }
  }
}

function Get-LLGoNativeArchitecture {
  try {
    return (ConvertTo-LLGoArchitecture `
      ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()))
  } catch {
    $architecture = if ($env:PROCESSOR_ARCHITEW6432) {
      $env:PROCESSOR_ARCHITEW6432
    } else {
      $env:PROCESSOR_ARCHITECTURE
    }
    return (ConvertTo-LLGoArchitecture $architecture)
  }
}

function Resolve-LLGoInstallArchitecture {
  if ($env:LLGO_ARCH) { return ConvertTo-LLGoArchitecture $env:LLGO_ARCH }
  return Get-LLGoNativeArchitecture
}

function ConvertTo-LLGoVersion {
  param([Parameter(Mandatory = $true)][string]$Version)

  $normalized = $Version -replace '^v', ''
  if ($normalized -notmatch '^[0-9A-Za-z][0-9A-Za-z.+-]*$') {
    throw "Invalid LLGo version: $Version"
  }
  return $normalized
}

function Invoke-LLGoDownload {
  param(
    [Parameter(Mandatory = $true)][string]$Uri,
    [Parameter(Mandatory = $true)][string]$OutFile
  )

  for ($attempt = 1; $attempt -le 5; $attempt++) {
    try {
      Invoke-WebRequest -UseBasicParsing -Uri $Uri -OutFile $OutFile
      return
    } catch {
      if ($attempt -eq 5) { throw }
      Start-Sleep -Seconds (2 * $attempt)
    }
  }
}

function Get-LLGoLatestVersion {
  $response = Invoke-WebRequest `
    -UseBasicParsing `
    -Method Head `
    -Uri "https://github.com/$script:LLGoRepository/releases/latest"
  $releaseUri = if ($response.BaseResponse.ResponseUri) {
    $response.BaseResponse.ResponseUri
  } elseif ($response.BaseResponse.RequestMessage.RequestUri) {
    $response.BaseResponse.RequestMessage.RequestUri
  } else {
    throw 'The latest GitHub release redirect did not expose its destination'
  }
  $tag = [IO.Path]::GetFileName($releaseUri.AbsolutePath)
  if (-not $tag -or $tag -eq 'latest') {
    throw 'The latest GitHub release redirect does not have a tag'
  }
  return (ConvertTo-LLGoVersion $tag)
}

function Get-LLGoClangTarget {
  param([string]$Clang)

  if (-not $Clang) { return '' }
  try {
    return (& $Clang -dumpmachine 2>$null | Select-Object -First 1).Trim()
  } catch {
    return ''
  }
}

function Resolve-LLGoWindowsABI {
  param([Parameter(Mandatory = $true)][string]$Architecture)

  if ($env:LLGO_ABI) {
    $abi = $env:LLGO_ABI.ToLowerInvariant()
    if ($abi -notin @('msvc', 'mingw')) {
      throw 'LLGO_ABI must be msvc or mingw'
    }
    return $abi
  }

  if ($env:MSYSTEM -in @('CLANG64', 'CLANGARM64')) {
    return 'mingw'
  }
  if ($env:VSCMD_ARG_TGT_ARCH -or $env:VCINSTALLDIR) {
    return 'msvc'
  }

  $clang = Get-Command clang.exe -ErrorAction SilentlyContinue
  $target = if ($clang) { Get-LLGoClangTarget $clang.Source } else { '' }
  if ($target -match '(windows-gnu|mingw32)') { return 'mingw' }
  if ($target -match 'windows-msvc') { return 'msvc' }

  $mingwSubdirectory = if ($Architecture -eq 'arm64') { 'clangarm64' } else { 'clang64' }
  $mingwClang = Join-Path "${env:SystemDrive}\msys64" "$mingwSubdirectory\bin\clang.exe"
  $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
  $hasMinGW = Test-Path -LiteralPath $mingwClang
  $hasMSVC = Test-Path -LiteralPath $vswhere
  if ($hasMinGW -and -not $hasMSVC) { return 'mingw' }

  # MSVC is the native Windows ABI and the least surprising default when no
  # installed compiler or activated shell identifies an existing profile.
  return 'msvc'
}

function Add-LLGoProcessPath {
  param([Parameter(Mandatory = $true)][string[]]$Directories)

  $parts = [Collections.Generic.List[string]]::new()
  foreach ($directory in $Directories) {
    if ($directory -and (Test-Path -LiteralPath $directory)) {
      $parts.Add($directory)
    }
  }
  foreach ($directory in ($env:PATH -split ';')) {
    if ($directory) { $parts.Add($directory) }
  }
  $env:PATH = ($parts | Select-Object -Unique) -join ';'
}

function Add-LLGoUserPath {
  param([Parameter(Mandatory = $true)][string[]]$Directories)

  $existing = [Environment]::GetEnvironmentVariable('Path', 'User')
  $parts = [Collections.Generic.List[string]]::new()
  foreach ($directory in $Directories) {
    if ($directory -and (Test-Path -LiteralPath $directory)) {
      $parts.Add($directory)
    }
  }
  foreach ($directory in ($existing -split ';')) {
    if ($directory) { $parts.Add($directory) }
  }

  $seen = [Collections.Generic.HashSet[string]]::new(
    [StringComparer]::OrdinalIgnoreCase
  )
  $unique = foreach ($directory in $parts) {
    if ($seen.Add($directory.TrimEnd('\'))) { $directory.TrimEnd('\') }
  }
  [Environment]::SetEnvironmentVariable('Path', ($unique -join ';'), 'User')
}

function Invoke-LLGoWinGet {
  param([Parameter(Mandatory = $true)][string[]]$Arguments)

  $winget = Get-Command winget.exe -ErrorAction SilentlyContinue
  if (-not $winget) {
    throw 'WinGet is required to install Windows dependencies'
  }
  & $winget.Source @Arguments | Out-Host
  if ($LASTEXITCODE -ne 0) {
    throw "winget failed with exit code $LASTEXITCODE"
  }
}

function Install-LLGoWinGetPackage {
  param(
    [Parameter(Mandatory = $true)][string]$Identifier,
    [string]$Version,
    [string]$Override
  )

  $arguments = @(
    'install', '--id', $Identifier, '--exact',
    '--accept-package-agreements', '--accept-source-agreements',
    '--disable-interactivity'
  )
  if ($Version) { $arguments += @('--version', $Version) }
  if ($Override) { $arguments += @('--override', $Override) }
  Invoke-LLGoWinGet $arguments
}

function ConvertTo-LLGoGoVersion {
  param([Parameter(Mandatory = $true)][string]$Version)

  $match = [regex]::Match($Version, '(?:^|\s)go(\d+)\.(\d+)(?:\.(\d+))?')
  if (-not $match.Success) { throw "Invalid Go version: $Version" }
  $patch = if ($match.Groups[3].Success) { $match.Groups[3].Value } else { '0' }
  return [version]"$($match.Groups[1].Value).$($match.Groups[2].Value).$patch"
}

function Test-LLGoGoVersion {
  $go = Get-Command go.exe -ErrorAction SilentlyContinue
  if (-not $go) { return $false }
  try {
    $reportedVersion = (& $go.Source env GOVERSION).Trim()
    $version = ConvertTo-LLGoGoVersion $reportedVersion
    return $version -ge [version]$script:LLGoGoVersion
  } catch {
    return $false
  }
}

function Install-LLGoGo {
  if (Test-LLGoGoVersion) { return @() }

  Install-LLGoWinGetPackage -Identifier 'GoLang.Go' -Version $script:LLGoGoVersion
  $goBin = Join-Path $env:ProgramFiles 'Go\bin'
  if (-not (Test-Path -LiteralPath $goBin) -and ${env:ProgramFiles(x86)}) {
    $goBin = Join-Path ${env:ProgramFiles(x86)} 'Go\bin'
  }
  Add-LLGoProcessPath @($goBin)
  if (-not (Test-LLGoGoVersion)) {
    throw "Go $script:LLGoGoVersion was installed but is not usable"
  }
  return @($goBin)
}

function Get-LLGoVisualStudioComponents {
  param([Parameter(Mandatory = $true)][string]$Architecture)

  $components = @('Microsoft.VisualStudio.Component.VC.Tools.x86.x64')
  if ($Architecture -eq 'arm64') {
    $components += 'Microsoft.VisualStudio.Component.VC.Tools.ARM64'
  }
  return $components
}

function Find-LLGoVisualStudio {
  param([Parameter(Mandatory = $true)][string]$Architecture)

  $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
  if (-not (Test-Path -LiteralPath $vswhere)) { return '' }
  $arguments = @('-latest', '-products', '*', '-version', '[17.0,18.0)')
  foreach ($component in (Get-LLGoVisualStudioComponents $Architecture)) {
    $arguments += @('-requires', $component)
  }
  $arguments += @('-property', 'installationPath')
  return (& $vswhere @arguments).Trim()
}

function Install-LLGoVisualStudio {
  param([Parameter(Mandatory = $true)][string]$Architecture)

  if (Find-LLGoVisualStudio $Architecture) { return }

  $arguments = @('--wait', '--passive', '--norestart')
  foreach ($component in (Get-LLGoVisualStudioComponents $Architecture)) {
    $arguments += @('--add', $component)
  }
  $arguments += '--includeRecommended'
  Install-LLGoWinGetPackage `
    -Identifier 'Microsoft.VisualStudio.2022.BuildTools' `
    -Override ($arguments -join ' ')
  if (-not (Find-LLGoVisualStudio $Architecture)) {
    throw 'Visual Studio 2022 C++ Build Tools were installed but are not discoverable'
  }
}

function Install-LLGoGit {
  $git = Get-Command git.exe -ErrorAction SilentlyContinue
  if ($git) { return $git.Source }

  Install-LLGoWinGetPackage -Identifier 'Git.Git'
  $gitBin = Join-Path $env:ProgramFiles 'Git\cmd'
  Add-LLGoProcessPath @($gitBin)
  $git = Get-Command git.exe -ErrorAction SilentlyContinue
  if (-not $git) { throw 'Git was installed but is not usable' }
  return $git.Source
}

function Install-LLGoLLVM {
  $llvmBin = Join-Path $env:ProgramFiles 'LLVM\bin'
  $clang = Join-Path $llvmBin 'clang.exe'
  $versionPattern = "clang version $([regex]::Escape($script:LLGoLLVMVersion))(?:\s|$)"
  $version = if (Test-Path -LiteralPath $clang) {
    (& $clang --version | Select-Object -First 1)
  } else {
    ''
  }
  if ($version -notmatch $versionPattern) {
    Install-LLGoWinGetPackage `
      -Identifier 'LLVM.LLVM' `
      -Version $script:LLGoLLVMVersion
  }
  if (-not (Test-Path -LiteralPath $clang)) {
    throw "LLVM $script:LLGoLLVMVersion was installed but clang.exe is missing"
  }
  $version = (& $clang --version | Select-Object -First 1)
  if ($version -notmatch $versionPattern) {
    throw "LLVM $script:LLGoLLVMVersion is required, found: $version"
  }
  return $llvmBin
}

function Install-LLGoMSVCDependencies {
  param(
    [Parameter(Mandatory = $true)][string]$Root,
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Architecture
  )

  $goPaths = @(Install-LLGoGo)
  Install-LLGoVisualStudio $Architecture
  $git = Install-LLGoGit
  $llvmBin = Install-LLGoLLVM

  $dependencyRoot = Join-Path $Root "dependencies\$Version\windows-$Architecture-msvc"
  $manifestRoot = Join-Path $dependencyRoot 'manifest'
  $installRoot = Join-Path $dependencyRoot 'installed'
  New-Item -ItemType Directory -Force $manifestRoot | Out-Null
  $manifest = Join-Path $manifestRoot 'vcpkg.json'
  Invoke-LLGoDownload `
    -Uri "https://raw.githubusercontent.com/$script:LLGoRepository/v$Version/.github/windows/vcpkg/vcpkg.json" `
    -OutFile $manifest
  $configuration = Get-Content -Raw -LiteralPath $manifest | ConvertFrom-Json
  $baseline = $configuration.'builtin-baseline'
  if ($baseline -notmatch '^[0-9a-f]{40}$') {
    throw 'The release vcpkg manifest does not contain a valid baseline'
  }

  $sourceRoot = Join-Path $Root "toolchains\vcpkg\$baseline"
  if (-not (Test-Path -LiteralPath (Join-Path $sourceRoot '.git'))) {
    New-Item -ItemType Directory -Force (Split-Path $sourceRoot) | Out-Null
    & $git clone --no-checkout https://github.com/microsoft/vcpkg.git $sourceRoot | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Cloning vcpkg failed' }
    & $git -C $sourceRoot checkout --detach $baseline | Out-Host
    if ($LASTEXITCODE -ne 0) { throw "Checking out vcpkg $baseline failed" }
  }
  $vcpkg = Join-Path $sourceRoot 'vcpkg.exe'
  if (-not (Test-Path -LiteralPath $vcpkg)) {
    & (Join-Path $sourceRoot 'bootstrap-vcpkg.bat') -disableMetrics | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Bootstrapping vcpkg failed' }
  }

  $triplet = if ($Architecture -eq 'arm64') { 'arm64-windows' } else { 'x64-windows' }
  & $vcpkg install `
    "--triplet=$triplet" `
    "--host-triplet=$triplet" `
    "--x-manifest-root=$manifestRoot" `
    "--x-install-root=$installRoot" `
    --clean-after-build | Out-Host
  if ($LASTEXITCODE -ne 0) { throw 'Installing vcpkg dependencies failed' }

  $tripletRoot = Join-Path $installRoot $triplet
  $pkgconf = Join-Path $tripletRoot 'tools\pkgconf\pkgconf.exe'
  if (-not (Test-Path -LiteralPath $pkgconf)) {
    throw "vcpkg did not install pkgconf at $pkgconf"
  }
  $profileTools = Join-Path $Root "profiles\windows-$Architecture-msvc\bin"
  New-Item -ItemType Directory -Force $profileTools | Out-Null
  $pkgConfig = Join-Path $profileTools 'pkg-config.cmd'
  @"
@echo off
"$pkgconf" --dont-define-prefix "--with-path=$tripletRoot\lib\pkgconfig" "--with-path=$tripletRoot\share\pkgconfig" %*
"@ | Set-Content -Encoding ascii -LiteralPath $pkgConfig

  $paths = @($profileTools, $llvmBin, (Join-Path $tripletRoot 'bin')) + $goPaths
  Add-LLGoProcessPath $paths
  return $paths
}

function Find-LLGoMSYS2 {
  if ($env:LLGO_MSYS2_ROOT -and
      (Test-Path -LiteralPath (Join-Path $env:LLGO_MSYS2_ROOT 'usr\bin\bash.exe'))) {
    return $env:LLGO_MSYS2_ROOT
  }
  $candidate = Join-Path $env:SystemDrive 'msys64'
  if (Test-Path -LiteralPath (Join-Path $candidate 'usr\bin\bash.exe')) {
    return $candidate
  }
  return ''
}

function Install-LLGoMSYS2 {
  $root = Find-LLGoMSYS2
  if ($root) { return $root }

  Install-LLGoWinGetPackage -Identifier 'MSYS2.MSYS2'
  $root = Find-LLGoMSYS2
  if (-not $root) { throw 'MSYS2 was installed but is not available at C:\msys64' }
  return $root
}

function Install-LLGoMinGWDependencies {
  param(
    [Parameter(Mandatory = $true)][string]$Root,
    [Parameter(Mandatory = $true)][string]$Architecture
  )

  $goPaths = @(Install-LLGoGo)
  $msysRoot = Install-LLGoMSYS2
  $bash = Join-Path $msysRoot 'usr\bin\bash.exe'
  $mingwSubdirectory = if ($Architecture -eq 'arm64') { 'clangarm64' } else { 'clang64' }
  $env:LLGO_INSTALL_MINGW_ARCH = $Architecture
  $env:MSYSTEM = if ($Architecture -eq 'arm64') { 'CLANGARM64' } else { 'CLANG64' }
  $env:CHERE_INVOKING = '1'
  $env:MSYS2_PATH_TYPE = 'inherit'

  $setup = @'
set -euo pipefail

pacman_retry() {
  local attempt
  for attempt in 1 2 3; do
    if pacman "$@"; then return; fi
    if [[ "$attempt" == 3 ]]; then return 1; fi
    sleep $((attempt * 5))
  done
}

case "$LLGO_INSTALL_MINGW_ARCH" in
  arm64)
    repo=https://repo.msys2.org/mingw/clangarm64
    prefix=mingw-w64-clang-aarch64
    ;;
  amd64)
    repo=https://repo.msys2.org/mingw/clang64
    prefix=mingw-w64-clang-x86_64
    ;;
  *)
    echo "unsupported MinGW host architecture: $LLGO_INSTALL_MINGW_ARCH" >&2
    exit 1
    ;;
esac

llvm_version=22.1.8
package_version=22.1.8-2
runtime_package_version=22.1.8-1
packages=(
  "clang-$package_version"
  "clang-libs-$package_version"
  "compiler-rt-$package_version"
  "llvm-$package_version"
  "llvm-libs-$package_version"
  "llvm-tools-$package_version"
  "lld-$package_version"
  "libc++-$runtime_package_version"
  "libunwind-$runtime_package_version"
)
urls=()
for package in "${packages[@]}"; do
  urls+=("$repo/$prefix-$package-any.pkg.tar.zst")
done

pacman_retry -Sy --noconfirm
pacman_retry -U --noconfirm \
  --assume-installed "$prefix-cc-libs=$llvm_version" \
  "${urls[@]}"
pacman_retry -S --needed --noconfirm \
  "$prefix-pkgconf" "$prefix-sqlite3" "$prefix-libuv" \
  "$prefix-gc" "$prefix-libatomic_ops" "$prefix-libffi" \
  "$prefix-openssl" "$prefix-zlib" "$prefix-cjson" "$prefix-make" make

test "$(llvm-config --version)" = "$llvm_version"
cflags="$(llvm-config --cflags)"
ldflags="$(llvm-config --ldflags --libs all --system-libs)"
pc_dir="$MINGW_PREFIX/lib/pkgconfig"
mkdir -p "$pc_dir"
printf '%s\n' \
  'Name: LLVM 22' \
  'Description: LLVM 22 host compiler and linker flags' \
  "Version: $llvm_version" \
  "Cflags: ${cflags//$'\n'/ }" \
  "Libs: ${ldflags//$'\n'/ }" \
  > "$pc_dir/llvm-22.pc"
'@
  & $bash -lc $setup | Out-Host
  if ($LASTEXITCODE -ne 0) { throw 'Installing MSYS2 MinGW dependencies failed' }

  $mingwRoot = Join-Path $msysRoot $mingwSubdirectory
  $mingwBin = Join-Path $mingwRoot 'bin'
  $pkgconf = Join-Path $mingwBin 'pkgconf.exe'
  $clang = Join-Path $mingwBin 'clang.exe'
  if (-not (Test-Path -LiteralPath $pkgconf) -or -not (Test-Path -LiteralPath $clang)) {
    throw "The MSYS2 $mingwSubdirectory profile is incomplete"
  }

  $profileTools = Join-Path $Root "profiles\windows-$Architecture-mingw\bin"
  New-Item -ItemType Directory -Force $profileTools | Out-Null
  $pkgConfig = Join-Path $profileTools 'pkg-config.cmd'
  @"
@echo off
"$pkgconf" --define-prefix "--with-path=$mingwRoot\lib\pkgconfig" %*
"@ | Set-Content -Encoding ascii -LiteralPath $pkgConfig

  $paths = @($profileTools, $mingwBin) + $goPaths
  Add-LLGoProcessPath $paths
  return $paths
}

function Install-LLGoWindowsDependencies {
  param(
    [Parameter(Mandatory = $true)][string]$Root,
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Architecture,
    [Parameter(Mandatory = $true)][string]$ABI
  )

  $stateRoot = Join-Path $Root "dependencies\$Version\windows-$Architecture-$ABI"
  $marker = Join-Path $stateRoot 'installer-v1-paths.txt'
  if ((Test-Path -LiteralPath $marker) -and (Test-LLGoGoVersion)) {
    $savedPaths = @(Get-Content -LiteralPath $marker | Where-Object { $_ })
    $validPaths = @($savedPaths | Where-Object { Test-Path -LiteralPath $_ })
    if ($savedPaths.Count -gt 0 -and $validPaths.Count -eq $savedPaths.Count) {
      Add-LLGoProcessPath $validPaths
      return $validPaths
    }
  }

  $paths = if ($ABI -eq 'mingw') {
    @(Install-LLGoMinGWDependencies -Root $Root -Architecture $Architecture)
  } else {
    @(Install-LLGoMSVCDependencies `
      -Root $Root `
      -Version $Version `
      -Architecture $Architecture)
  }
  New-Item -ItemType Directory -Force $stateRoot | Out-Null
  $paths | Set-Content -Encoding utf8 -LiteralPath $marker
  return $paths
}

function Set-LLGoJunction {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string]$Target
  )

  $item = Get-Item -Force -LiteralPath $Path -ErrorAction SilentlyContinue
  if ($item) {
    if (-not ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
      throw "$Path exists and is not a junction"
    }
    Remove-Item -Force -LiteralPath $Path
  }
  New-Item -ItemType Junction -Path $Path -Target $Target | Out-Null
}

function Install-LLGoRelease {
  param(
    [Parameter(Mandatory = $true)][string]$Root,
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Architecture,
    [Parameter(Mandatory = $true)][string]$ABI,
    [string]$LocalArchive
  )

  $platform = "windows-$Architecture-$ABI"
  $asset = "llgo$Version.$platform.zip"
  $versionRoot = Join-Path $Root "versions\$Version\$platform"
  $executable = Join-Path $versionRoot 'bin\llgo.exe'
  if (-not (Test-Path -LiteralPath $executable)) {
    $temporary = Join-Path ([IO.Path]::GetTempPath()) "llgo-install-$([Guid]::NewGuid())"
    New-Item -ItemType Directory -Force $temporary | Out-Null
    try {
      $archive = Join-Path $temporary $asset
      if ($LocalArchive) {
        if (-not (Test-Path -LiteralPath $LocalArchive)) {
          throw "Local archive not found: $LocalArchive"
        }
        Copy-Item -LiteralPath $LocalArchive -Destination $archive
      } else {
        $base = "https://github.com/$script:LLGoRepository/releases/download/v$Version"
        $checksums = Join-Path $temporary 'checksums.txt'
        Invoke-LLGoDownload `
          -Uri "$base/llgo$Version.checksums.txt" `
          -OutFile $checksums
        Invoke-LLGoDownload -Uri "$base/$asset" -OutFile $archive
        $checksumLine = Get-Content -LiteralPath $checksums |
          Where-Object { $_ -match "^[0-9a-f]{64}\s+$([regex]::Escape($asset))$" } |
          Select-Object -First 1
        if (-not $checksumLine) { throw "Checksum for $asset is missing" }
        $expected = ($checksumLine -split '\s+')[0]
        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
        if ($actual -ne $expected) { throw "SHA-256 mismatch for $asset" }
      }

      $extracted = Join-Path $temporary 'extracted'
      Expand-Archive -LiteralPath $archive -DestinationPath $extracted
      if (-not (Test-Path -LiteralPath (Join-Path $extracted 'bin\llgo.exe'))) {
        throw "$asset does not contain bin\llgo.exe"
      }
      if (-not (Test-Path -LiteralPath (Join-Path $extracted 'runtime\go.mod'))) {
        throw "$asset does not contain the LLGo runtime"
      }
      New-Item -ItemType Directory -Force (Split-Path $versionRoot) | Out-Null
      Move-Item -LiteralPath $extracted -Destination $versionRoot
    } finally {
      if (Test-Path -LiteralPath $temporary) {
        Remove-Item -Recurse -Force -LiteralPath $temporary
      }
    }
  }

  New-Item -ItemType Directory -Force $Root | Out-Null
  $current = Join-Path $Root 'current'
  Set-LLGoJunction -Path $current -Target $versionRoot
  Set-LLGoJunction -Path (Join-Path $Root 'bin') -Target (Join-Path $current 'bin')
  return $executable
}

function Install-LLGo {
  if (-not $IsWindows -and $env:OS -ne 'Windows_NT') {
    throw 'install.ps1 supports Windows only; use install.sh on macOS or Linux'
  }

  $architecture = Resolve-LLGoInstallArchitecture
  $abi = Resolve-LLGoWindowsABI -Architecture $architecture
  $version = if ($env:LLGO_VERSION) {
    ConvertTo-LLGoVersion $env:LLGO_VERSION
  } else {
    Get-LLGoLatestVersion
  }
  $root = if ($env:LLGO_INSTALL_ROOT) {
    [IO.Path]::GetFullPath($env:LLGO_INSTALL_ROOT)
  } else {
    Join-Path $HOME '.llgo'
  }
  if (-not $root -or $root -eq [IO.Path]::GetPathRoot($root)) {
    throw "Unsafe installation root: $root"
  }

  $installDependencies = if ($env:LLGO_INSTALL_DEPS) {
    $env:LLGO_INSTALL_DEPS
  } elseif ($architecture -eq '386') {
    '0' # The maintained native x86 SDK/dependency environment is supplied separately.
  } else {
    '1'
  }
  $updatePath = if ($env:LLGO_UPDATE_PATH) { $env:LLGO_UPDATE_PATH } else { '1' }
  if ($installDependencies -notin @('0', '1')) {
    throw 'LLGO_INSTALL_DEPS must be 0 or 1'
  }
  if ($updatePath -notin @('0', '1')) {
    throw 'LLGO_UPDATE_PATH must be 0 or 1'
  }

  if ($installDependencies -eq '1' -and $architecture -eq '386') {
    throw 'Automatic x86 development dependency installation is unavailable; configure a native i686 toolchain and use LLGO_INSTALL_DEPS=0'
  }
  New-Item -ItemType Directory -Force $root | Out-Null
  $dependencyPaths = @()
  if ($installDependencies -eq '1') {
    $dependencyPaths = @(Install-LLGoWindowsDependencies `
      -Root $root `
      -Version $version `
      -Architecture $architecture `
      -ABI $abi)
  }

  $null = Install-LLGoRelease `
    -Root $root `
    -Version $version `
    -Architecture $architecture `
    -ABI $abi `
    -LocalArchive $env:LLGO_ARCHIVE_PATH
  $bin = Join-Path $root 'bin'
  Add-LLGoProcessPath (@($bin) + $dependencyPaths)
  if ($updatePath -eq '1') {
    Add-LLGoUserPath (@($bin) + $dependencyPaths)
  }
  if ($env:GITHUB_PATH) {
    (@($bin) + $dependencyPaths) | Add-Content -Encoding utf8 $env:GITHUB_PATH
  }

  & (Join-Path $bin 'llgo.exe') version
  if ($LASTEXITCODE -ne 0) { throw 'The installed llgo.exe did not start' }
  Write-Host "Installed LLGo $version for windows/$architecture ($abi) in $root\current"
  Write-Host "Stable command: $bin\llgo.exe"
  if ($updatePath -eq '1') {
    Write-Host 'Open a new terminal to use the updated user PATH.'
  }
}

if ($env:LLGO_INSTALLER_LIBRARY_ONLY -ne '1') {
  $previousErrorActionPreference = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Stop'
    Install-LLGo
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
}
