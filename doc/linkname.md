# Linkname bindings

`go:linkname` selects a linker symbol for a package function or variable. LLGo
also recognizes native ABI prefixes, as discussed in [#2879](https://github.com/xgo-dev/llgo/issues/2879).

## Functions

```go
import _ "unsafe"

//go:linkname Add shared_add
func Add(a, b int32) int32 { return a + b }

//go:linkname CallAdd shared_add
func CallAdd(int32, int32) int32
```

`Add` defines `shared_add`; `CallAdd` references that literal symbol. Both use
the Go ABI. A bare target does not imply C, and targets are not recursively
resolved through another declaration's linkname.

The single-argument form `//go:linkname Add` retains the default `pkgpath.Add`
symbol; it does not create a bare `Add` symbol.

```go
//go:linkname NativeAdd C.native_add
func NativeAdd(a, b int32) int32 { return a + b }

//go:linkname Puts C.puts
func Puts(*byte) int32
```

A `C.` function with a body defines a native entry around its Go implementation.
The entry handles foreign-thread registration and the GC boundary, like
`//export`. Calls from Go to the implementation keep the Go ABI. A prototype
without a body references an external native function.

`stdcall.` follows the same definition/reference rule with the Windows calling
convention: stdcall on 386, the platform C ABI on amd64 and arm64. Native entries
currently require non-generic, non-variadic functions with at most one result.
Methods cannot define native entries this way. The native symbol must differ
from the implementation's qualified Go name. Use a native linkname or `//export`
on a declaration, not both.

## Variables

```go
//go:linkname Counter shared_counter
var Counter int32
```

An ordinary Go variable defines storage even without an initializer. If two
packages name the same symbol:

- Two zero definitions share storage and have the same address.
- One nonzero static initializer supplies the shared value, overriding zeros.
- Two nonzero static definitions are a duplicate-symbol error.
- Dynamic initializers still execute in package initialization order and write
  the shared storage.

Ordinary function variables retain Go closure representation; slices and strings
retain their Go representation. Native variables need an explicit prefix:

```go
//go:linkname Counter C.native_counter
var Counter int32

//go:linkname Callback stdcall.native_callback
var Callback func(int32) int32
```

These reference storage owned by native code and cannot have Go initializers.
A function variable stores a native function pointer. `stdcall.` determines how
to call that pointer; scalar variables use ordinary native storage. `llgo.` is
not a variable namespace.

Explicit `go:cgo_import_static` and `go:cgo_import_dynamic` metadata can instead
make an eligible zero definition an import, including a variable with its default
package name. Windows dynamic import slots hold imported addresses. These cgo
placeholders must not become ordinary Go zero storage.

## Placeholder bindings

```go
//llgo:link Add C.native_add
func Add(a, b int32) int32 { panic("placeholder") }
```

`llgo:link` requires a placeholder body for a function or method. LLGo ignores
that body completely: the declaration only references its target and never
exports an implementation. This applies to Go targets as well as `C.` and
`stdcall.` targets. It is not interchangeable with a defining `go:linkname`.

Method bindings can name the receiver, for example:

```go
//llgo:link Handle.Close C.close_handle
func (h Handle) Close() int32 { panic("placeholder") }
```

The receiver is passed as the first native argument. Existing `py.` module
bindings and `llgo.` compiler instructions keep their specialized behavior.

## Regression coverage

[`test/go/linkname_storage_test.go`](../test/go/linkname_storage_test.go) runs
ordinary linkname examples with both gc and LLGo. LLGo-only native entries,
variables, stdcall and placeholder bindings belong in
[`test/llgoext`](../test/llgoext). Compiler tests additionally check source-to-IR
lowering and reject invalid native declarations.
