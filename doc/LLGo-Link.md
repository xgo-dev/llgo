Binding Go Symbols to C/C++/Python Symbols via //go:linkname and //llgo:link
=====

## Summary

LLGo lets Go code call C, C++ and Python libraries, and lets Go code be called from C, by attaching an external symbol name to a Go declaration. It does this with two directives:

- `//go:linkname`, the standard Go directive, whose *linkname* syntax LLGo extends with a calling-convention prefix.
- `//llgo:link`, an LLGo-specific directive for placeholder function and **method** bindings. `//go:linkname` cannot be applied to methods.

For `//go:linkname`, **the prefix decides the ABI, and the presence of a body decides whether the declaration defines the symbol or references it.** These are two independent questions. A body never turns prefix interpretation off. `C.name` on a function with a body exports a C entry point, and `C.name` on a prototype references an external C function. In contrast, `//llgo:link` requires a placeholder body and only references its target; that body is ignored.

Variables follow the same split, with one wrinkle: for a Go variable, the absence of an initializer does not mean the declaration is only a reference. Without an extension prefix, every such declaration is a (zero-initialized) definition, exactly as in gc. Extension prefixes (`C.`, `stdcall.`) mark *extern storage* instead.

This document describes both directives and the extended linkname syntax so that binding authors (including generators such as `llcppg`) have a single reference.

## Motivation

Go's `//go:linkname` normally takes the form `pkgPath.Name` and connects a local Go symbol to a symbol in another Go package. LLGo compiles Go to native code through LLVM, so it can also cross the language boundary in both directions: call C functions and variables, C++ functions (by their mangled names) and Python module attributes, and expose Go functions as C or stdcall entry points. Doing so requires the linkname to say three things:

1. Which language or calling convention the target uses.
2. What the target's name is in that world.
3. In some cases, that the "function" is not a real function at all but a compiler instruction.

LLGo meets these needs by giving the linkname a `<callconv>.<name>` form instead of `pkgPath.Name`.

## Reading a linkname: prefix and body

For `//go:linkname <GoName> <target>`, read the declaration in two steps.

1. **The prefix picks the ABI.** No prefix or `pkgPath.` means the Go ABI. `C.` means the C ABI (`cdecl`). `stdcall.` means the `stdcall` ABI. `py.` and `llgo.` are special and have their own sections.
2. **The body (or, for variables, the prefix) picks the role.** A function with a body defines the symbol. A prototype references a symbol defined elsewhere. Variables are covered in their own section.

### Functions

| Target | With a body | Prototype only |
|--------|-------------|----------------|
| `name` / `pkg.name` | Defines a Go symbol using the Go ABI | References the Go symbol |
| `C.name` | Exports a C ABI entry, with thread/GC boundary handling, similar to `//export` | References an external C function |
| `stdcall.name` | Exports a `stdcall` entry with the same runtime boundary handling | References an external `stdcall` function |

Targets are literal linker symbols, not names recursively resolved through another declaration's linkname. A bare target does not imply C. The single-argument form `//go:linkname Add` retains the default `pkgpath.Add` symbol; it does not create a bare `Add` symbol.

Why the body does not disable prefix interpretation:

- A native caller entering Go code needs a boundary: the thread may not be known to the Go runtime, and the GC must be able to see the frames. An exported entry (the `C.` and `stdcall.` cells) is where that boundary handling is generated, the same job `//export` does.
- Ordinary Go entries need no additional native-to-Go boundary, because Go calls Go. This is why the Go-ABI row is just a plain define or reference.
- The prefix always means the same thing. `//go:linkname f C.f` is about the C symbol `f` whether or not `f` has a body. Code that only wants to give a Go function a link name should use a bare name or `<pkgPath>.<name>`.

An example of each role:

```go
// Reference: call an external C function.
//go:linkname Puts C.puts
func Puts(s *Char) CInt

// Export: a C caller can call add_ints. The runtime boundary is generated.
//go:linkname addInts C.add_ints
func addInts(a, b int32) int32 {
	return a + b
}

// Define a Go symbol that other Go packages may link to by name.
//go:linkname fastRand example.fastRand
func fastRand() uint32 {
	return 4
}
```

### `C.<name>`: C and C++ functions

Use `C.` for functions with the default `cdecl` convention. For C++ symbols, `<name>` is the mangled name (`manglingName`) as it appears in the object file. For plain C this is just the C identifier.

```go
//go:linkname Puts C.puts
func Puts(s *Char) CInt
```

With a body, the declaration exports a C ABI entry under that name.

Calls from Go to that implementation keep the Go ABI. Native entries currently require non-generic, non-variadic functions with at most one result. Methods cannot define native entries this way. The native symbol must differ from the implementation's qualified Go name. Use a native linkname or `//export` on a declaration, not both.

### `stdcall.<name>`: stdcall functions

Identical to `C.<name>`, except the entry or the call uses the `stdcall` convention. With a body, it exports a `stdcall` entry with the same runtime boundary handling as `C.<name>`.

```go
//go:linkname MessageBeep stdcall.MessageBeep
func MessageBeep(typ uint32) int32
```

`stdcall.` is supported on Windows targets only. On other targets, LLGo rejects it when its ABI is needed; an unused declaration may not trigger that diagnostic.

On Windows 386 it selects stdcall; on Windows amd64 and arm64 it uses the platform C ABI. The native-entry restrictions above also apply.

### `py.<name>`: Python functions

`py.` is not a general calling convention, so the body/prototype table does not apply to it. It binds a Go prototype to a function inside a Python module. The module is declared once per package through the `LLGoPackage` constant, and each function is then bound by its name inside that module:

```go
const LLGoPackage = "py.math"   // moduleName = "math"

//go:linkname Sqrt py.sqrt
func Sqrt(x *Object) *Object
```

Here `math` is the Python module name, and `sqrt` is the function's name within it.

`py.` is reference-only. If a function annotated with `//go:linkname` and a `py.` target has a body, LLGo ignores that body and binds the Python module attribute. Placeholder `//llgo:link` bindings retain their specialized Python behavior.

### `llgo.<name>`: compiler instructions

`llgo.` is also outside the body/prototype table. Some declarations look like functions in Go source but are expanded by the compiler. The linkname `llgo.<name>` marks them. The canonical example is in package `c`:

```go
//go:linkname Str llgo.cstr
func Str(string) *Char
```

`c.Str` is **not** a call to a function. It defines a C-style string constant. For example, `c.Str("Hello")` yields a constant `*c.Char` pointing at a 6-byte, `\0`-terminated string.

For `//go:linkname`, a function targeting `llgo.<name>` expands to a compiler instruction; any Go body is ignored. Placeholder `//llgo:link` instruction bindings retain their specialized behavior. `llgo.<name>` is not allowed on variables.

## Linkname on variables

A Go variable declaration without an initializer is still a definition (it is zero-initialized). So "has an initializer" cannot be what separates a definition from a reference. The prefix does that job.

### Extern storage: `C.<name>` and `stdcall.<name>`

A variable with an extension prefix denotes **external storage**. The storage comes from the external provider, and the Go declaration only describes how Go sees it.

```go
// Native extern: storage comes from the external provider.
//go:linkname Counter C.native_counter
var Counter int32
```

Rules:

- `stdcall.native_counter` also denotes external storage (Windows targets only; other targets reject it when its ABI is needed).
- For scalars and other non-function types, no calling convention is involved. The prefix is still required, because it is what says "this is extern storage" rather than "this is a Go definition".
- For function-pointer variables, the prefix additionally selects the calling convention, and the variable uses the native function-pointer representation:

  ```go
  //go:linkname Foo C.foo
  var Foo func()                 // cdecl function pointer

  //go:linkname Handler stdcall.handler
  var Handler func(code int32)   // stdcall function pointer
  ```

- Multiple extern bindings that name the same symbol reference the same external storage.
- Go initializers on these declarations are rejected. The Go side does not own the storage, so it cannot provide its initial value.
- `llgo.<name>` is not allowed on variables.

Native bindings must therefore not omit the prefix, even for scalars. Without the prefix the declaration is a Go definition with Go storage semantics, which is a different thing from an extern.

Migration: existing native variable bindings with bare targets must add `C.` or `stdcall.`; bare targets now define Go-owned storage.

### Go variable semantics: no extension prefix

Without an extension prefix (a bare `<name>` or `<pkgPath>.<name>`), variables have gc's Go semantics. Consider two packages that select the same symbol:

```go
// In package p (with unsafe imported):
//go:linkname A shared_var
var A int

// In package q (with unsafe imported):
//go:linkname B shared_var
var B int
```

**Both declarations contribute zero-initialized storage definitions, even though neither has an initializer.** gc merges them into one storage object: their addresses are equal and writes through either name are visible through the other. Neither is an extern-only declaration, and either also works independently without another provider.

This was verified with Go 1.27.2 across packages using matching types:

| Definitions selecting the same symbol | Result |
|---------------------------------------|--------|
| Both zero-initialized | One shared zero-initialized storage object |
| One static nonzero initializer, others zero-initialized | Shared storage retaining the nonzero initial value |
| Two static nonzero initializers, e.g. `7` and `41` | Duplicate-definition link error |
| Both initialized by runtime function calls | Shared storage; both initializers execute in Go initialization order |

Two conclusions follow:

- Initializer presence alone cannot distinguish a definition from a reference. A declaration with no initializer is still a definition.
- Multiple initializers do not always imply a duplicate-definition error. Only conflicting *static* nonzero initializers do.

Trusted `//go:cgo_import_dynamic` and `//go:cgo_import_static` declarations also mark their Go symbol as imported storage. This suppresses ordinary zero/BSS definitions across packages, including a declaration with its default package-qualified name. A nonzero static Go initializer remains a definition and takes precedence. These directives require `unsafe` in the same source file or a generated `_cgo_` filename; a sibling file or `//line` directive does not grant that trust.

Ordinary function variables keep Go function-value semantics. `//go:linkname Foo foo` with `var Foo func()` makes `Foo` a Go closure variable, exactly as an ordinary `func()` variable is. It is not a C function pointer. To get a native function pointer, spell the prefix out as shown above.

Slices and strings also retain their ordinary Go representation. Explicit `go:cgo_import_static` and `go:cgo_import_dynamic` metadata can instead make an eligible zero definition an import, including a variable with its default package name. Windows dynamic import slots hold imported addresses. These cgo placeholders must not become ordinary Go zero storage.

### Go-defined variables seen from C

Direct C `extern` access to a variable defined by Go is not guaranteed. The supported way for C to reach Go state is an exported accessor function (a function with a body under a `C.` linkname):

```go
var state int32

//go:linkname getState C.get_state
func getState() int32 { return state }
```

## `//llgo:link` placeholder bindings

`//llgo:link` requires a placeholder body for a function or method. LLGo ignores the body completely: the declaration only references its target and never exports an implementation. This applies to Go targets as well as `C.` and `stdcall.` targets. It is not interchangeable with a defining `//go:linkname`.

```go
//llgo:link Add C.native_add
func Add(a, b int32) int32 { panic("placeholder") }
```

### Methods

`//go:linkname` cannot be applied to methods, because a method has no standalone Go identifier to name. `//llgo:link` extends it for this case. The first argument is written as `TypeName.MethodName`, and the second argument is the same `<callconv>.<name>` linkname as above:

```go
//llgo:link TranslationUnit.Cursor C.clang_getTranslationUnitCursor
func (self TranslationUnit) Cursor() Cursor {
	return Cursor{}
}
```

This binds the Go method `TranslationUnit.Cursor` to the C function `clang_getTranslationUnitCursor`, with the receiver passed as the first C argument.

`//llgo:link` only references an external symbol. It has no export mode and no naming-only mode.

**About the fake body.** A method annotated with `//llgo:link` carries a placeholder body (`return Cursor{}` above). This exists only to satisfy the Go compiler, which requires a method to have a body. The body has no runtime meaning: LLGo replaces the method with the external symbol. The same placeholder rule applies to function bindings and is a property of `//llgo:link`, not of `//go:linkname`.

## Summary of rules

### Functions

| Target | Directive | Linkname form | With a body | Prototype only |
|--------|-----------|---------------|-------------|----------------|
| Go symbol | `//go:linkname` | `<name>` or `<pkgPath>.<name>` | Defines a Go symbol (Go ABI) | References the Go symbol |
| C / C++ function | `//go:linkname` | `C.<manglingName>` | Exports a C entry, with thread/GC boundary handling | References an external C function |
| stdcall function | `//go:linkname` | `stdcall.<manglingName>` | Exports a stdcall entry, same boundary handling (Windows only) | References an external stdcall function (Windows only) |
| Python function | `//go:linkname` | `py.<name>` | Body ignored; references the Python module function | References a Python module function; needs `const LLGoPackage = "py.<moduleName>"` |
| Compiler instruction | `//go:linkname` | `llgo.<name>` | Body ignored; expanded by the compiler | Expanded by the compiler (e.g. `llgo.cstr`) |
| Function or method placeholder | `//llgo:link` | `<name>`, `<pkgPath>.<name>` or `<callconv>.<name>` | References the target; placeholder body ignored | Rejected; a placeholder body is required |

On non-Windows targets, a `stdcall.` binding is rejected when its ABI is needed; this is not an eager check of every unused declaration.

### Variables

| Declaration | Linkname form | Meaning |
|-------------|---------------|---------|
| Non-function type, extern | `C.<name>` or `stdcall.<name>` | External storage; no Go initializer allowed; repeated bindings share the storage |
| Function pointer, extern | `C.<name>` or `stdcall.<name>` | External storage holding a native function pointer; the prefix selects the calling convention; no Go initializer allowed |
| Go variable, any type | none, `<name>` or `<pkgPath>.<name>` | Go definition with gc semantics: zero-initialized storage merged by symbol; function-typed variables hold Go closures |
| Any variable | `llgo.<name>` | Not allowed |

## Compatibility

- Standard Go linknames (`pkgPath.Name`) work unchanged, for functions with or without a body, and for variables with or without an initial value.
- A linkname on a variable without a prefix has `godecl` semantics and gc's merge rules, as in the official Go compiler.
- Because `//llgo:link` is an ordinary comment to the Go toolchain, packages using it still parse and type-check with the standard Go compiler, which is why the placeholder body is needed.

## Regression coverage

[`test/go/linkname_storage_test.go`](../test/go/linkname_storage_test.go) runs ordinary linkname examples with both gc and LLGo. LLGo-only native entries, variables, stdcall and placeholder bindings belong in [`test/llgoext`](../test/llgoext). Compiler tests additionally check source-to-IR lowering and reject invalid native declarations.
