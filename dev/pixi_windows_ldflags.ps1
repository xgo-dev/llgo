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
'"-L' + $libraryDirectory + '" ' + ($flags -join ' ')
