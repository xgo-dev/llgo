## LLGo Plugin of LLDB

### Build with debug info

```shell
llgo build -O0 -ldflags=-w=false -o cl/_testdata/debug/out ./cl/_testdata/debug
```

LLGo temporarily omits DWARF when `-w` is absent because the current debug
information path is not yet safe for broad use. The native executable build
above explicitly uses `-ldflags=-w=false` to enable DWARF. Use
`-ldflags=-w` to explicitly omit it. `-O0` is recommended for the most
complete local variable inspection in LLDB. The former `LLGO_DEBUG` and
`LLGO_DEBUG_SYMBOLS` environment variables are no longer read. This uses
LLGo's existing runnable DWARF path; improving its metadata quality and making
it optimization-independent are separate follow-up work.

LLGo currently marks compile units as `DW_LANG_C` because stock LLDB does not
provide a Go language plugin and otherwise hides valid frame variables.
`DW_AT_producer` remains `LLGo`, and a versioned debugger marker lets this
plugin distinguish LLGo binaries from ordinary C targets. Upstream LLDB
requires an RFC and a long-term maintainer before restoring native Go language
support, so LLGo keeps its language-specific adapter external; the decision is
recorded in [issue #2154](https://github.com/xgo-dev/llgo/issues/2154).

### Debug with lldb

```shell
llgo lldb ./cl/_testdata/debug/out
```

Use `-lldb` or `LLGO_LLDB` to select a particular LLDB 18+ executable. Arguments
after the LLGo flags are passed through to LLDB; use `--` when the first LLDB
argument begins with `-`:

```shell
llgo lldb -lldb /opt/homebrew/bin/lldb -- --batch ./cl/_testdata/debug/out
```

The command embeds and loads the LLGo Python adapter, so an installed `llgo`
does not depend on a source checkout. `cmd/llgo/lldbtest/runlldb.sh` remains as
a thin compatibility wrapper. Adapter commands live under `llgo`, including
`llgo status`, `llgo print`, and `llgo vars`; stock LLDB commands and aliases
such as `p` and `v` are left unchanged. `llgo status` reports the recognized
debugger schema, runtime-layout version, target triple, pointer size, and byte
order. Unknown marker versions disable only the LLGo-specific commands; raw
LLDB debugging remains available.

For recognized LLGo targets, the adapter also gives strings length-bounded
quoted summaries and slices `len`/`cap` summaries with indexed synthetic
children. These views cover named string and slice types as well as the
predeclared types. Explicit `llgo print` slice views respect LLDB's
`target.max-children-count` setting. Ordinary C targets and targets with
unknown or ambiguous LLGo markers retain LLDB's raw presentation.

The integration fixture follows LLDB's API-test style: `main.go` marks
executable breakpoint lines with `LLDB_BREAK`, while `test.py` keeps the
expected variables and values in an explicit SB API test table. Assertions are
not parsed from source comments.

```text
# github.com/xgo-dev/llgo/cl/_testdata/debug
Breakpoint 1: no locations (pending).
Breakpoint set in dummy target, will get copied into future targets.
(lldb) target create "./cl/_testdata/debug/out"
Current executable set to '/Users/lijie/source/goplus/llgo/cl/_testdata/debug/out' (arm64).
(lldb) r
Process 21992 launched: '/Users/lijie/source/goplus/llgo/cl/_testdata/debug/out' (arm64)
globalInt: 301
s: 0x100123e40
0x100123be0
5 8
called function with struct
1 2 3 4 5 6 7 8 9 10 +1.100000e+01 +1.200000e+01 true (+1.300000e+01+1.400000e+01i) (+1.500000e+01+1.600000e+01i) [3/3]0x1001129a0 [3/3]0x100112920 hello 0x1001149b0 0x100123ab0 0x100123d10 0x1001149e0 (0x100116810,0x1001149d0) 0x10011bf00 0x10010fa80 (0x100116840,0x100112940) 0x10001b4a4
9
1 (0x1001167e0,0x100112900)
called function with types
0x100123e40
0x1000343d0
Process 21992 stopped
* thread #1, queue = 'com.apple.main-thread', stop reason = breakpoint 1.1
    frame #0: 0x000000010001b3b4 out`main at in.go:225:12
   222 		println(globalStructPtr)
   223 		println(&globalStruct)
   224 		s.i8 = 0x12
-> 225 		println(s.i8) // LLDB_BREAK: main_struct_updated
(lldb) llgo vars
var i int = <variable not available>
var s github.com/xgo-dev/llgo/cl/_testdata/debug.StructWithAllTypeFields = {
  i8 = '\x12',
  i16 = 2,
  i32 = 3,
  i64 = 4,
  i = 5,
  u8 = '\x06',
  u16 = 7,
  u32 = 8,
  u64 = 9,
  u = 10,
  f32 = 11,
  f64 = 12,
  b = true,
  c64 = {real = 13, imag = 14},
  c128 = {real = 15, imag = 16},
  slice = []int{21, 22, 23},
  arr = [3]int{24, 25, 26},
  arr2 = [3]github.com/xgo-dev/llgo/cl/_testdata/debug.E{{i = 27}, {i = 28}, {i = 29}},
  s = "hello",
  e = {i = 30},
  pf = 0x0000000100123d10,
  pi = 0x00000001001149e0,
  intr = {type = 0x0000000100116810, data = 0x00000001001149d0},
  m = {count = 4296130304},
  c = {},
  err = {type = 0x0000000100116840, data = 0x0000000100112940},
  fn = {f = 0x000000010001b4a4, data = 0x00000001001149c0},
  pad1 = 100,
  pad2 = 200
}
var globalStructPtr *github.com/xgo-dev/llgo/cl/_testdata/debug.StructWithAllTypeFields = <variable not available>
var globalStruct github.com/xgo-dev/llgo/cl/_testdata/debug.StructWithAllTypeFields = {
  i8 = '\x01',
  i16 = 2,
  i32 = 3,
  i64 = 4,
  i = 5,
  u8 = '\x06',
  u16 = 7,
  u32 = 8,
  u64 = 9,
  u = 10,
  f32 = 11,
  f64 = 12,
  b = true,
  c64 = {real = 13, imag = 14},
  c128 = {real = 15, imag = 16},
  slice = []int{21, 22, 23},
  arr = [3]int{24, 25, 26},
  arr2 = [3]github.com/xgo-dev/llgo/cl/_testdata/debug.E{{i = 27}, {i = 28}, {i = 29}},
  s = "hello",
  e = {i = 30},
  pf = 0x0000000100123d10,
  pi = 0x00000001001149e0,
  intr = {type = 0x0000000100116810, data = 0x00000001001149d0},
  m = {count = 4296130304},
  c = {},
  err = {type = 0x0000000100116840, data = 0x0000000100112940},
  fn = {f = 0x000000010001b4a4, data = 0x00000001001149c0},
  pad1 = 100,
  pad2 = 200
}
var globalInt int = 301
var err error = {type = 0x0000000100112900, data = 0x000000000000001a}
```
