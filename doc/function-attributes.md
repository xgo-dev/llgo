# Function attributes

LLGo accepts `cold` and `noreturn` on named functions and methods, including
external declarations. Each attribute occupies its own comment line:

```go
//llgo:cold
//llgo:noreturn
func Fatal(message string) { panic(message) }
```

The spaced spelling `// llgo:cold` is also accepted. Attributes take no
arguments; identical repetitions are harmless. They cannot annotate `init`,
blank functions, variables, types, or statements.

`cold` is an optimization hint that calls are uncommon. `noreturn` promises
that the function never returns normally: panic and infinite loops are allowed.
It does not imply `nounwind`. A function that recovers its own panic and returns
does not satisfy the promise. Registering a deferred call or starting a goroutine
still returns normally even when its target has `noreturn`.

Attributes are programmer promises and insert no runtime checks. Invalid
promises can cause incorrect optimization. Syntax and placement are checked;
arbitrary function bodies are not proved correct by the compiler.

Imported declarations, generic instances, and backend-created runtime and method
entries consume the prepared `internal/directive` records. A package patch's
active implementation supplies its callable attributes, while analysis of source
bodies retains their own source properties. Different declarations sharing a
linker symbol remain distinct; every promise must hold for the implementation
it names. Unknown indirect calls do not acquire properties from a function type.

The build driver prepares and binds records before concurrent lowering. Standalone
clients preparing imports use `ParsePkgSyntaxWithOptions` and then bind their
checked `types.Info` (or use `BindScope` when only checked types are available).
No additional comment index or backend source scanner is required.

## Parameters and results

Select a whole Go value with `param(name)` / `param(index)`, `result(name)` /
`result(index)`, or `receiver`. Indices are zero based and exclude receivers and
hidden ABI arguments. Unnamed and blank values use indices. `result` alone selects
the sole result and is invalid for zero or multiple results.

```go
//llgo:result(out) nonnull sameas(p)
//llgo:result(count) range(0, 64)
func Make(p *int, n uint32) (out *int, count uint32) {
    if p == nil { panic("nil pointer") }
    return p, n & 63
}
```

| Attribute | Supported values | Meaning |
| --- | --- | --- |
| `nonnull` | Pointer parameters and results, including unsafe.Pointer | The selected pointer is not nil. |
| `range(lo, hi)` | Integer parameters and results | A nonempty, non-wrapping, half-open interval of integer literals within the type's domain. |
| `nonnegative` | Integer parameters and results | Nonnegative at the target's Go integer width; unrestricted for unsigned types. |
| `sameas(name)` | Integer or pointer results | Equals the entry value of the named parameter; integer types must agree and pointer conversions preserve the pointer. |

Multiple attributes on a selector line are separated by whitespace. `range` and
`nonnegative` intersect; an empty intersection is invalid. Identical repetitions
are accepted. Incompatible explicit contracts on known link aliases are diagnosed
without merging source declaration records. Generic type checks are deferred to
concrete instances when necessary.

Input facts hold at entry. Result facts hold after normal return, including changes
made by defers. A result's `nonnull` does not prohibit nil input followed by panic.
`sameas` retains the call and all side effects, and forwards the already-evaluated
input rather than reloading memory after the call.

Before ABI conversion, the backend emits value assumptions on entry, normal returns,
and known calls' normal continuations. Multiple results use logical tuple paths.
ABI conversion carries these expressions through packing, splitting, and indirect
storage; restrictions never apply to transfer-storage addresses by accident.
Direct matching values also receive LLVM attributes. `returned` is used only when
the single LLVM result has the input's representation.

The build driver materializes module-local value plans before ABI conversion.
Standalone backend clients call `Program.MaterializeValueAttributes` at the same
boundary. Source records remain Go data shared across backends; LLVM plans are local.

## Pointer access and alias restrictions

Pointer parameters and receivers additionally accept:

- `access(none)`, `access(read)`, `access(write)`, or `access(readwrite)`: restrict
  accesses through that pointer and pointers derived from it, throughout the call.
  Other parameters and globals remain unrestricted by this attribute alone.
- `noalias`: accessed memory that is modified during the call must be accessed
  only through this pointer or pointers derived from it. Shared reads of unmodified
  memory remain valid. This does not imply nonnull, pointer inequality, or no capture.

```go
//llgo:param(dst) access(write) noalias
//llgo:param(src) access(read)
func CopyOne(dst, src *int) { *dst = *src }
```

These restrictions include callees. They use LLVM parameter attributes, preserving
logical parameter indices across receivers, environments, aggregate arguments and
sret. `access(readwrite)` adds no LLVM restriction. The compiler never infers a
function-wide memory restriction from an individual pointer's access mode.

For explicit GC-root or cooperative-safepoint modes, pointer restrictions are
currently omitted from LLVM definitions and imports alike: collector/scheduler
accesses are not yet modeled precisely. Value contracts remain enabled. In other
modes, retained pointer restrictions are rejected at their source annotation if
lowering inserts an unmodeled runtime protocol, such as defer state, closure heap
allocation, local-context access, shadow-stack updates or late ABI allocation.
This conservative diagnostic does not prove arbitrary source implementations.

Overlapping runtime copies receive access restrictions but no noalias promise.
The runtime annotations cover audited read/copy/clear helpers; ordinary packages
use the same mechanism.

Cgo-specific parsing and handling remain outside this feature.
