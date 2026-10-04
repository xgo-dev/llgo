# Size Report Options

The `llgo build -size` flag measures the final linked artifact after Wasm
optimization, PCLN/debug packaging, stripping and firmware format conversion.
It reports file bytes separately from code/data payload and lists the final
build artifacts. A requested report that cannot be produced fails the build;
it is not silently replaced by a warning.

## Parsing Strategy

- ELF is read directly using section flags/types and symbol sizes. All
  `SHF_ALLOC` sections are included, including unwind and initialization arrays.
  Zero-size labels do not split functions, and aliases/overlapping symbols are
  counted once per section. Overlaps belonging to different owners appear as
  `(shared <section>)`; bytes without a sized owner appear as
  `(unknown <section>)`, which may include padding. Stripped files still have
  accurate section totals even when source attribution is unavailable.
- Wasm is read directly, without invoking `llvm-readelf`. Code is the encoded
  function bodies (including local declarations); data is the stored segment
  content, including passive segments. Names are taken from the final `name`
  section when available, otherwise code/data stay in explicit unknown buckets.
  Custom-section bytes and the remaining encoding bytes are listed separately:
  `code + data + custom_bytes + structure_bytes == file_size`.
- Browser output resolves the sibling `.wasm` instead of reading JavaScript or
  HTML as an object file. The artifact list includes the glue, owned host
  resources, runtime symbols and external DWARF described by the build output.
  Alternate firmware formats are listed individually, not summed as if every
  format must be deployed together.
- PE/COFF is read directly using section characteristics. Section payload excludes
  raw-file alignment padding; virtual zero-fill is counted as BSS. COFF symbol
  ownership uses address-range estimates because symbols do not carry ELF-style
  sizes. Stripped Windows executables retain section totals.
- Mach-O retains the `llvm-readelf` reader and address-range estimates.
  Parentheses within Go method names are preserved.

Native `flash` remains the sum of allocated code/rodata/data section payload;
`ram` is data plus zero-fill (ELF NOBITS or PE virtual zero-fill). These are not the ELF container size,
load-image padding, physical RAM after target address-alias resolution, or
dynamic runtime usage. Linker reservations may be included in NOBITS.

Wasm reports imported/defined linear-memory initial and maximum sizes, shared
status and memory64 separately. It does not infer BSS from unused linear memory
or claim that data bytes equal RAM usage; `flash` and `ram` fields are omitted
for Wasm. Stack/heap reservations and worker/runtime peaks need additional
target/runtime measurements.

Symbol attribution describes the final physical owner. Inlining and LTO may
move work across package boundaries. LLVM instruction counts and archive sizes
are not substituted for final bytes; this reader also works with cached build
inputs and after LLVM modules have been released.

The parsers, aggregation and output live in `internal/sizereport`, with only
standard-library dependencies. The build layer selects the final module, converts
package/artifact metadata and invokes the Mach-O tool fallback.

## Aggregation Levels

`-size-level` controls how symbol names are grouped prior to reporting:

| Level     | Behavior                                                                 |
|-----------|---------------------------------------------------------------------------|
| `full`    | Keeps the raw owner from the symbol name (previous behavior).             |
| `package` | Uses the list of packages built in `build.Do` and groups by `pkg.PkgPath`. |
| `module`* | Default. Groups by `pkg.Module.Path` (or `pkg.PkgPath` if the module is nil). |

Matching is performed by checking whether the demangled symbol name begins with
`pkg.PkgPath + "."`. Unmatched entries keep the owner derived from their symbol
name so compiler-generated functions remain visible as ordinary functions
rather than being grouped by an implementation-specific category.

Defaults:

- `-size` alone enables the report with `-size-format=text` and `-size-level=module`.
- `-size-format` accepts `text` (table output) or `json`; omitting the flag uses `text`.
- `-size-level` defaults to `module`, with `package` and `full` as the other options.

Examples:

```sh
llgo build -size .                     # module-level aggregation (default)
llgo build -size -size-level=package . # collapse by package ID
llgo build -size -size-level=full .    # show raw symbol owners
llgo build -size -size-format=json .   # JSON output (works with all levels)
```

## Validation

JSON reports have `version: 1`, `stage: "final"`, `format`, `file_size`,
`binary`, `modules`, `total` and `artifacts`. Wasm additionally has a `wasm`
object containing sections, memory limits and encoding overhead. Optional
diagnostic warnings do not change the measured totals. Native per-module fields
remain compatible with the previous report.

1. Unit tests:
   ```sh
   CGO_ENABLED=0 go test ./internal/sizereport -count=1
   go test ./internal/build -run 'Test(SizeReport|FinalSize|ReportBuildOutputs)' -count=1
   ```
2. Real binary test:
   ```sh
   LLGO_SIZE_REPORT_BIN=/absolute/path/to/app.wasm \
     go test ./test/sizereport -run '^TestCollectFinalSizeRealBinary$' -count=1
   ```
3. Manual smoke test: `llgo build -size -size-level=module .` (or
   `package`/`full` as desired).
