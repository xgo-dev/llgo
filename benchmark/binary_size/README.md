# Binary size workloads

These fixed programs keep LLGo binary-size measurements comparable across
linker and PCLN packaging changes. The comprehensive linker/PCLN matrix is
maintained by `llgo-test/benchmarks`; `benchmark/baseline` reuses the three
smallest workloads for the per-commit lightweight size and timing gate.

- `cprintf`: the smallest C-library hello world, using only `lib/c.Printf`.
- `println`: a hello world using the built-in `println`.
- `fmtprintf`: a hello world using `fmt.Printf`.
- `texttemplate`: a heavier standard-library program using `text/template`.
- `nilfault`: an import-free abnormal path that dereferences a nil pointer.

The first four programs must exit successfully with their expected output.
`nilfault` must exit unsuccessfully with a nil-pointer diagnostic in every
mode; for `external`, check it once with the sidecar present and once after the
sidecar has been removed. The runtime integration suite separately verifies
that a hardware-fault signal path never starts sidecar I/O.

Size reports for `-pclntab=external` must record the executable and its
`.pclntab` sidecar separately, with their sum shown explicitly. Run every
linker/PCLN variant at the same output path: DWARF records paths, so changing
the output path can otherwise make byte counts look different even when two
flag combinations have equivalent code-generation semantics.

External PCLN is built from the final linked entry and PC-line sites. Dead
function and PC-line records are removed together with unused strings, and
function indexes and hash tables are rebuilt before writing the sidecar. To
check this optimization, compare normal and Full-LTO `cprintf` and `fmtprintf`
builds with `-pclntab=external -ldflags=-w`, recording the sidecar's function,
PC-line and site counts as well as its bytes. The executable's text bytes must
remain unchanged. Embedded PCLN still uses its existing metadata layout.

This initial matrix applies to native Darwin/Linux executable builds;
generation, archive/shared, and fixed-target policies are outside this
benchmark.

The initial comparison matrix is the Cartesian product of
`embedded|external|none` and these linker groups: default, `-w`, `-s`, and
`-s -w=false`. Until native symbol-table deletion is implemented, also assert
that `-s` has the same byte counts as `-w`, and `-s -w=false` has the same byte
counts as the default, when each pair uses the exact same output path.
