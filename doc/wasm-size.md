# Final WebAssembly size optimization

Use LLGo's cross-platform size levels and Go-compatible linker stripping:

```sh
llgo build -Oz -ldflags=-s -o app ./app
llgo build -target=wasi -Oz -ldflags=-s -o app.wasm ./app
llgo build -target=emscripten -Oz -ldflags=-s -o app.mjs ./app
```

Explicit `-Os` and `-Oz` select the corresponding LLVM size optimization on
all targets. Explicit `-Oz` also runs a final Binaryen `-Oz` pass for hosted
W32, J32 and J64 executables after required linker/Asyncify transformations. The pass
uses an explicit feature allowlist; only J64 enables memory64, and WASI retains its multimemory capability. Inferred target
optimization defaults and `-Os` retain their existing pipeline. Generation, libraries
and firmware conversions continue to use their normal compiler optimization.

`-ldflags=-s` strips optional Wasm names and DWARF, as it strips native symbols
and debug information. `-ldflags="-s -w=0"` strips names while retaining DWARF
and source maps. `-w` alone controls DWARF and does not request name stripping.
The built-in metadata stripper requires no Binaryen when size optimization is
disabled. Code/data section bytes, runtime PCLN, target features, build identity,
external DWARF references and unknown custom sections are retained.

For explicit Wasm metadata policy, `-wasm-strip=debug` removes DWARF, name and
source-map metadata; `-wasm-strip=all` additionally removes producer metadata.
The default is `none` unless `-s` requests stripping. Explicit
`-wasm-strip=none` conflicts with `-s`. PCLN policy remains independently
controlled by `-pclntab`.

For Emscripten, size mode sets `EXPORT_ALL=0`. Explicit native/public exports
and the existing runtime methods (`cwrap`, `allocateUTF8`, `stringToUTF8`,
`UTF8ToString`, `FS`, `setValue`, `getValue`, `ENV`) remain available; incidental
JS implementation symbols are no longer published on `Module`. `EXPORT_ALL`
controls publication, not reachability; the final pass does not remove these
explicit runtime APIs.

## Toolchain and debugging

Use LLGo Binaryen `llgo-v132.3`, selected through `WASMOPT` or
`EM_BINARYEN_ROOT`. Browser builds must also set `EM_BINARYEN_ROOT` so
Emscripten uses the patched tools during linking, not only the final pass.
CI pins the release archive checksum. Size optimization
checks the Binaryen 132 version string and records the executable SHA-256;
the version string alone cannot distinguish upstream Binaryen from LLGo's
patched build.

Debug packaging happens after optimization:

```sh
llgo build -target=emscripten -Oz \
  -debug-artifact=external -wasm-strip=debug -o app.mjs ./app
```

Embedded DWARF is incompatible with explicit stripping. External DWARF keeps a
matching `app.debug.wasm`; stripping applies only to the deployed module.
When retaining DWARF, the final pass skips Binaryen 132's `precompute`,
`precompute-propagate` and `local-cse` passes, which lose surviving Go call-site
line mappings. Other `-Oz` passes retain Binaryen's DWARF compatibility checks.
Emscripten source maps requested through `LDFLAGS=-gsource-map` are regenerated
for the optimized code, retaining their URL and source locations after publication.
Debug stripping removes the map file and URL together.
CI checks Go/C++ source locations and executes the final modules for W32/J32/J64.

## Artifact manifest

An enabled final pipeline writes `app.wasm-manifest.json` beside the output.
It records compiler/source-Go/LLVM/link-driver identity, target profile and
pointer width, PCLN and final optimization modes, Binaryen identity/arguments
(empty for built-in stripping), effective linker driver arguments (including
environment flags and generated defaults), declared and allowed Wasm features, and each artifact's byte count,
gzip-9 byte count and SHA-256. Its build ID covers sorted artifact names and
hashes. Tool names and absolute flag paths are normalized to omit local machine
directories; the manifest is a build record, not a portable linker command.
Debugger build IDs remain paired with the external DWARF container.

`payload` includes Wasm, JS/HTML loaders, worker assets and runtime symbols;
`debug` counts external debugger containers and source maps separately. Gzip totals sum files
compressed independently. Totals exclude the manifest itself; LLGo's artifact
report includes its actual size. Include it when measuring a deployment that
ships the manifest. External debug containers contain the module's standard
sections as well as DWARF, so their size is not just the DWARF payload.

The complete output set is staged before publication. Ordinary publication
failures restore the previous files; an unsuccessful restore retains its backup
and reports the recovery path. The manifest is published last. This is not an
atomic multi-file deployment: serve a versioned directory or verify artifact
hashes before switching readers. Successful builds remove obsolete debug/glue
siblings, and a successful rebuild without final optimization or stripping removes
the old manifest.

## Measurement

Darwin arm64, Go 1.27.2, LLVM 22.1.8, Emscripten 6.0.8 and LLGo Binaryen
`llgo-v132.3`; fixtures `benchmark/binary_size/{cprintf,println,fmtprintf}`.
The baseline uses LLGo's default optimization and `-debug-artifact=none`;
the final column additionally selects explicit `-Oz` and metadata stripping.
The named targets already default to LLVM `Oz`. Neither column uses
development DCE or LTO. Baseline: `057851875`. The measurements do not include the separate initializer/metadata pruning in PR #2908. All 18 final
products executed successfully in Wasmer or the matching Node runner.

Sizes below are **complete artifact bytes / independently compressed Brotli-11
bytes**, including generated JS, `wasm_fs.js` and the final column's manifest.
There are no deferred runtime assets in these cases. Small browser programs
can have a slightly larger compressed total because of the manifest.

| Profile | Example | Baseline | Final | Raw reduction |
| --- | --- | ---: | ---: | ---: |
| W32 | cprintf | 160,243 / 44,666 | 118,610 / 40,517 | 25.98% |
| W32 | println | 159,933 / 44,678 | 118,358 / 40,387 | 26.00% |
| W32 | fmtprintf | 2,371,607 / 514,578 | 1,937,708 / 495,349 | 18.30% |
| J32 | cprintf | 252,135 / 70,298 | 248,991 / 70,371 | 1.25% |
| J32 | println | 251,679 / 70,215 | 248,508 / 70,129 | 1.26% |
| J32 | fmtprintf | 3,139,290 / 703,680 | 3,072,575 / 700,958 | 2.13% |
| J64 | cprintf | 242,604 / 66,670 | 239,736 / 66,807 | 1.18% |
| J64 | println | 242,194 / 66,485 | 239,296 / 66,574 | 1.20% |
| J64 | fmtprintf | 2,908,050 / 642,098 | 2,848,135 / 640,863 | 2.06% |
