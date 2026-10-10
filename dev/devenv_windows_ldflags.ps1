$ErrorActionPreference = 'Stop'

$libraryDirectory = ((& llvm-config --libdir) -join ' ').Trim().Replace('\', '/')
if ($LASTEXITCODE -ne 0 -or -not (Test-Path $libraryDirectory)) {
  throw "llvm-config reported an invalid library directory: $libraryDirectory"
}

$libraryNames = ((& llvm-config --link-static --libnames all) -join ' ').Trim()
if ($LASTEXITCODE -ne 0 -or -not $libraryNames) {
  throw 'llvm-config failed to report the LLVM libraries'
}
$systemNames = ((& llvm-config --link-static --system-libs) -join ' ').Trim()
if ($LASTEXITCODE -ne 0) {
  throw 'llvm-config failed to report the system libraries'
}

$libraries = ($libraryNames, $systemNames) -split '\s+' | Where-Object {
  $_ -and $_ -notin @('libxml2s.lib', 'xml2s.lib', 'xml2.lib')
}
$flags = $libraries | ForEach-Object {
  if (-not $_.EndsWith('.lib', [StringComparison]::OrdinalIgnoreCase)) {
    throw "llvm-config reported an unsupported library: $_"
  }
  $name = [IO.Path]::GetFileNameWithoutExtension($_) -replace '\.dll$', ''
  '-l' + $name
}
# LLVM's runtime dependencies do not always include import libraries. Expose
# native development packages' lib directories explicitly without replacing
# the SDK/CRT's LIB environment.
$searchDirectories = @($libraryDirectory)
if ($env:LIBRARY_PATH) {
  $searchDirectories += $env:LIBRARY_PATH -split ';' | Where-Object { $_ }
}
$searchFlags = $searchDirectories | Select-Object -Unique | ForEach-Object {
  '"-L' + $_.Replace('\', '/') + '"'
}
($searchFlags -join ' ') + ' ' + ($flags -join ' ')
