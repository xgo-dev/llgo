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
        $null = Assert-ReleasePE -ReadObj $ReadObj -Path $source -GoArch $GoArch -Profile $Profile
        Copy-Item -LiteralPath $source -Destination $local
        $queue.Enqueue($local)
      } elseif (-not (Test-Path -LiteralPath (Join-Path $SystemDirectory $dll))) {
        throw "$path imports $dll, but it is absent from the release and selected LLVM profile"
      }
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

function Expand-ReleaseXz {
  param([string]$URL, [string]$SHA256, [string]$Destination, [string[]]$Entries = @())

  $temporary = Join-Path $env:RUNNER_TEMP ('llgo-release-download-' + [Guid]::NewGuid())
  New-Item -ItemType Directory -Force $temporary, $Destination | Out-Null
  try {
    $archive = Join-Path $temporary 'payload.tar.xz'
    & curl.exe --fail --location --retry 5 --retry-all-errors --output $archive $URL
    if ($LASTEXITCODE -ne 0) { throw "Downloading $URL failed" }
    $actual = (Get-FileHash -Algorithm SHA256 $archive).Hash
    if ($actual -ne $SHA256) { throw "SHA-256 for $URL is $actual, expected $SHA256" }
    # Use 7-Zip for xz: the Windows ARM64 runner's bsdtar cannot extract all
    # upstream sparse xz archives. tar.gz release archives use bsdtar normally.
    $sevenZip = Join-Path $env:ProgramFiles '7-Zip/7z.exe'
    & $sevenZip x -y "-o$temporary" $archive | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Decompressing the release payload failed' }
    Remove-Item -LiteralPath $archive
    & $sevenZip x -y "-o$Destination" (Join-Path $temporary 'payload.tar') @Entries | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Extracting the release payload failed' }
  } finally {
    Remove-Item -LiteralPath $temporary -Recurse -Force
  }
}
