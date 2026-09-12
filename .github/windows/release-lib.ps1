# Shared checks for the Windows release builder and the extracted-archive test.
# Resolve DLLs only from the selected LLVM profile, never from the build PATH.

function Get-ReleasePE {
  param([string]$ReadObj, [string]$Path)

  $output = (& $ReadObj --file-headers --coff-imports $Path | Out-String)
  if ($LASTEXITCODE -ne 0) {
    throw "Inspecting $Path failed with exit code $LASTEXITCODE"
  }
  if ($output -notmatch 'Machine:\s+IMAGE_FILE_MACHINE_(\w+)\s') {
    throw "$Path does not have a COFF machine header"
  }
  $machine = $Matches[1]
  $imports = @([regex]::Matches($output, '(?im)^\s*Name:\s*(\S+\.dll)\s*$') |
    ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique)
  return [PSCustomObject]@{ Machine = $machine; Imports = $imports }
}

function Assert-ReleasePE {
  param([string]$ReadObj, [string]$Path, [string]$GoArch, [string]$Profile)

  $pe = Get-ReleasePE -ReadObj $ReadObj -Path $Path
  $want = @{ amd64 = 'AMD64'; arm64 = 'ARM64' }[$GoArch]
  if (-not $want -or $pe.Machine -ne $want) {
    throw "$Path has PE machine $($pe.Machine), expected windows/$GoArch ($want)"
  }
  foreach ($dll in $pe.Imports) {
    if ($dll -match '^(msys-2\.0|cygwin1|libwinpthread.*|libgcc_s.*|libstdc\+\+.*)\.dll$' -or
        ($Profile -eq 'msvc' -and $dll -match '^(libc\+\+|libunwind).*\.dll$')) {
      throw "$Path imports a runtime outside the $Profile release profile: $dll"
    }
  }
  return $pe
}

function Copy-ReleaseDLLs {
  param(
    [string]$ReadObj, [string]$Executable, [string]$GoArch, [string]$Profile,
    [string[]]$SourceDirectories,
    [string]$SystemDirectory = (Join-Path $env:SystemRoot 'System32'),
    [switch]$CheckOnly
  )

  $destination = Split-Path $Executable
  $queue = [Collections.Generic.Queue[string]]::new()
  $queue.Enqueue($Executable)
  $visited = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
  while ($queue.Count) {
    $path = $queue.Dequeue()
    if (-not $visited.Add($path)) { continue }
    $pe = Assert-ReleasePE -ReadObj $ReadObj -Path $path -GoArch $GoArch -Profile $Profile
    foreach ($dll in $pe.Imports) {
      # API-set DLLs are contracts provided by Windows, not redistributable files.
      if ($dll -match '^(api-ms-|ext-ms-)') { continue }
      $local = Join-Path $destination $dll
      if (Test-Path -LiteralPath $local) {
        $queue.Enqueue($local)
        continue
      }
      $source = $null
      if (-not $CheckOnly) {
        foreach ($directory in $SourceDirectories) {
          $candidate = Join-Path $directory $dll
          if (Test-Path -LiteralPath $candidate) {
            $source = $candidate
            break
          }
        }
      }
      if ($source) {
        Copy-Item -LiteralPath $source -Destination $local
        # Validate the packaged copy when it is dequeued, including its imports.
        $queue.Enqueue($local)
      } elseif (-not (Test-Path -LiteralPath (Join-Path $SystemDirectory $dll))) {
        throw "$path imports $dll, but it is absent from the release and selected LLVM profile"
      }
      # System32 is the OS dependency boundary, not a strict DLL allowlist.
      # A runner-specific DLL installed there can mask a missing redistributable;
      # the standalone test on a fresh runner checks that boundary independently.
    }
  }
}

function Invoke-ReleaseCapture {
  param([string]$Executable, [string[]]$ArgumentList)

  $savedPreference = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    $output = (& $Executable @ArgumentList 2>&1 | ForEach-Object { $_.ToString() }) -join "`n"
    $code = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $savedPreference
  }
  if ($code -ne 0) { throw "$Executable exited with $code`n$output" }
  return $output
}

function Get-ReleaseMSVCLinkFlags {
  param([string]$LibraryDirectory, [string[]]$LibraryFlags = @())

  # cmd/go only recognizes a quote at the start of an argument. -L"path"
  # preserves literal quotes and breaks the generated //go:cgo_ldflag pragma.
  $path = $LibraryDirectory.Replace('\', '/')
  if ($path -match '["\r\n]') { throw "Invalid LLVM library directory: $path" }
  return (@('"-L' + $path + '"') + $LibraryFlags) -join ' '
}

function Get-ReleaseArchive {
  param([string]$URL, [string]$SHA256, [string]$CacheDirectory)

  if ($SHA256 -notmatch '^[0-9a-fA-F]{64}$') { throw 'Invalid release SHA-256' }
  New-Item -ItemType Directory -Force $CacheDirectory | Out-Null
  $archive = Join-Path $CacheDirectory "$SHA256.tar.xz"
  if (-not (Test-Path -LiteralPath $archive)) {
    $partial = "$archive.$([Guid]::NewGuid()).partial"
    try {
      & curl.exe --fail --location --retry 5 --retry-all-errors --output $partial $URL
      if ($LASTEXITCODE -ne 0) { throw "Downloading $URL failed" }
      $actual = (Get-FileHash -Algorithm SHA256 $partial).Hash
      if ($actual -ne $SHA256) { throw "SHA-256 for $URL is $actual, expected $SHA256" }
      Move-Item -LiteralPath $partial -Destination $archive -Force
    } finally {
      if (Test-Path -LiteralPath $partial) { Remove-Item -LiteralPath $partial }
    }
  }
  # Restored caches are untrusted. Verify the pinned archive on every use and
  # always extract it afresh; neither extracted files nor markers are cached.
  $actual = (Get-FileHash -Algorithm SHA256 $archive).Hash
  if ($actual -ne $SHA256) { throw "SHA-256 for $URL is $actual, expected $SHA256" }
  return $archive
}

function Expand-ReleaseXz {
  param(
    [string]$URL, [string]$SHA256, [string]$Destination,
    [string]$CacheDirectory, [string[]]$Entries = @()
  )

  $temporary = Join-Path $env:RUNNER_TEMP ('llgo-release-download-' + [Guid]::NewGuid())
  New-Item -ItemType Directory -Force $temporary, $Destination | Out-Null
  try {
    $archive = Get-ReleaseArchive -URL $URL -SHA256 $SHA256 -CacheDirectory $CacheDirectory
    # Use 7-Zip for xz: the Windows ARM64 runner's bsdtar cannot extract all
    # upstream sparse xz archives.
    $sevenZip = Join-Path $env:ProgramFiles '7-Zip/7z.exe'
    & $sevenZip x -y "-o$temporary" $archive | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Decompressing the release payload failed' }
    & $sevenZip x -y "-o$Destination" (Join-Path $temporary "$SHA256.tar") @Entries | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Extracting the release payload failed' }
  } finally {
    Remove-Item -LiteralPath $temporary -Recurse -Force
  }
}

function Assert-ReleaseESPPayload {
  param([string]$Root, [string]$GoArch, [string]$Version)

  $target = @{ amd64 = 'x86_64-w64-mingw32'; arm64 = 'aarch64-w64-mingw32' }[$GoArch]
  $want = @{ amd64 = 'AMD64'; arm64 = 'ARM64' }[$GoArch]
  $manifest = Get-Content -LiteralPath (Join-Path $Root 'LLGO-LLVM-MANIFEST.txt')
  if (-not $target -or $manifest -notcontains "host_target=$target") {
    throw "The ESP payload manifest does not match windows/$GoArch"
  }
  if (-not $Version -or $manifest -notcontains "payload_version=$Version") {
    throw "The ESP payload manifest does not match version $Version"
  }
  $bin = Join-Path $Root 'bin'
  $readObj = Join-Path $bin 'llvm-readobj.exe'
  $binaries = @(Get-ChildItem -LiteralPath $bin -File |
    Where-Object { $_.Extension -in @('.exe', '.dll') })
  if (-not $binaries.Count) { throw 'The ESP payload contains no PE binaries' }
  foreach ($binary in $binaries) {
    # ESP tools execute in separate processes and carry their own MinGW DLLs,
    # independently of the LLGo compiler's MSVC or MinGW runtime profile.
    $pe = Get-ReleasePE -ReadObj $readObj -Path $binary.FullName
    if ($pe.Machine -ne $want) {
      throw "$($binary.Name) has PE machine $($pe.Machine), expected ESP host $want"
    }
    foreach ($dll in $pe.Imports) {
      if ($dll -match '^(api-ms-|ext-ms-)') { continue }
      if (-not (Test-Path -LiteralPath (Join-Path $bin $dll)) -and
          -not (Test-Path -LiteralPath (Join-Path $env:SystemRoot "System32/$dll"))) {
        throw "$($binary.Name) imports $dll, which is absent from the ESP payload and Windows"
      }
    }
  }
  $savedPath = $env:PATH
  try {
    $env:PATH = "$bin;$env:SystemRoot/System32;$env:SystemRoot"
    foreach ($tool in @('clang', 'clang++', 'ld.lld', 'llc', 'opt', 'llvm-config')) {
      $null = Invoke-ReleaseCapture (Join-Path $bin "$tool.exe") @('--version')
    }
  } finally {
    $env:PATH = $savedPath
  }
  Write-Host "Validated native ESP tools and DLLs for windows/$GoArch"
}
