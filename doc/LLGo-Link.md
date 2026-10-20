Binding Go Symbols to C/C++/Python Symbols via //go:linkname and //llgo:link
=====

## Summary

LLGo lets Go code call C, C++ and Python libraries by attaching an external symbol name to a Go declaration. It does this with two directives:

- `//go:linkname`, the standard Go directive, whose *linkname* syntax LLGo extends with a calling-convention prefix.
- `//llgo:link`, an LLGo-specific directive that does the same job for **methods**, which `//go:linkname` cannot handle.

`//go:linkname` has two distinct uses, and it is important to keep them apart:

1. **Linking to an external symbol.** The declaration has no body (or no initial value), and the linkname points at something defined elsewhere. All the LLGo extensions described below apply here.
2. **Naming a local symbol.** The declaration has a body (or an initial value), and the linkname only gives it a link name. This is the standard Go compiler behavior, which LLGo stays compatible with. None of the LLGo extensions apply (see [Linkname on declarations with a body](#linkname-on-declarations-with-a-body-or-initial-value)).

This document describes both directives and the extended linkname syntax so that binding authors (including generators such as `llcppg`) have a single reference.

## Motivation

Go's `//go:linkname` normally takes the form `pkgPath.Name` and connects a local Go symbol to a symbol in another Go package. LLGo compiles Go to native code through LLVM, so it can also link against symbols that live outside the Go world: C functions and variables, C++ functions (by their mangled names), and Python module attributes. Doing so requires the linkname to say three things:

1. Which language or calling convention the target uses.
2. What the target's name is in that world.
3. In some cases, that the "function" is not a real function at all but a compiler instruction.

LLGo meets these needs by giving the linkname a `<callconv>.<name>` form instead of `pkgPath.Name`.

## Extended `//go:linkname` syntax

This syntax applies to declarations that link to an external symbol, that is, functions without a body and variables without an initial value.

```go
//go:linkname <GoName> <callconv>.<name>
```

| Prefix | Meaning of `<name>` | Calling convention |
|--------|---------------------|--------------------|
| `C.<name>` | The target's **mangled name** (`manglingName`). For plain C this is just the C identifier. | `cdecl` |
| `stdcall.<name>` | The target's **mangled name** (`manglingName`). | `stdcall` |
| `py.<name>` | The name of the function inside a Python module. Must be combined with `LLGoPackage` (see below). | n/a |
| `llgo.<name>` | Not a symbol at all: `<name>` selects a built-in LLGo **instruction**. | n/a |

### `C.<name>`: C and C++ functions

Use `C.` for ordinary C functions with the default `cdecl` convention. For C++ symbols, `<name>` is the mangled name as it appears in the object file.

```go
//go:linkname Puts C.puts
func Puts(s *Char) CInt
```

### `stdcall.<name>`: stdcall functions

Identical to `C.<name>`, except the call uses the `stdcall` convention.

```go
//go:linkname MessageBeep stdcall.MessageBeep
func MessageBeep(typ uint32) int32
```

### `py.<name>`: Python functions

For Python, the module is declared once per package through the `LLGoPackage` constant, and each function is then bound by its name inside that module:

```go
const LLGoPackage = "py.math"   // moduleName = "math"

//go:linkname Sqrt py.sqrt
func Sqrt(x *Object) *Object
```

Here `math` is the Python module name, and `sqrt` is the function's name within it.

### `llgo.<name>`: compiler instructions

Some declarations look like functions in Go source but are expanded by the compiler. The linkname `llgo.<name>` marks them. The canonical example is in package `c`:

```go
//go:linkname Str llgo.cstr
func Str(string) *Char
```

`c.Str` is **not** a call to a function. It defines a C-style string constant. For example, `c.Str("Hello")` yields a constant `*c.Char` pointing at a 6-byte, `\0`-terminated string.

## Linkname on global variables

`//go:linkname` also works on package-level variables that link to an external symbol (no initial value):

```go
//go:linkname Foo C.foo
var Foo func()
```

Here `Foo` is a C function pointer variable: the `C.` prefix tells LLGo that the function stored in `Foo` follows the `cdecl` convention.

For variables the `<callconv>` prefix is **optional**. When it is omitted, LLGo uses the **Go calling convention (`godecl`)**. This matches the behavior of the official Go compiler, so Go code that relies on linkname for variables keeps working unchanged. The practical consequences are:

- If the variable has a function type, a plain linkname means the variable holds a **Go closure**, exactly as an ordinary `func()` variable does in Go. It is not a C function pointer.
- If the variable is not a function type (an `int`, a struct, a pointer, and so on), there is no call involved, so `godecl` and `cdecl` behave identically. You can leave the prefix out.

So the declaration below is **not** the same as the `C.foo` form above. `Foo` is a Go closure here:

```go
//go:linkname Foo foo
var Foo func()
```

To get a callback variable that LLGo calls through a non-Go convention, spell the prefix out:

```go
//go:linkname Foo C.foo
var Foo func()          // cdecl function pointer

//go:linkname Handler stdcall.handler
var Handler func(code int32)   // stdcall function pointer
```

A variable can never be an instruction, so `llgo.<name>` is **not allowed** on `var`. Every other form (`C.`, `stdcall.`, `py.`, or no prefix) is possible.

## Linkname on declarations with a body or initial value

`//go:linkname` is not only for linking to external symbols. When the declaration **has a body** (a function) or **an initial value** (a variable), the directive does not link to anything. It only gives that Go function or variable a link name, so that other packages can refer to it by that name:

```go
//go:linkname fastRand example.fastRand
func fastRand() uint32 {
	// real implementation
	return 4
}

//go:linkname counter example.counter
var counter = 0
```

This is the standard behavior of the official Go compiler, and LLGo is compatible with it. In this case the linkname is an ordinary Go linkname, written either as a bare `<name>` or as `<pkgPath>.<name>` (the form used in the examples above), and **none of the extensions in this document apply**: `C.<name>`, `stdcall.<name>`, `py.<name>` and `llgo.<name>` are not interpreted as calling-convention prefixes or instructions.

The rule of thumb is simple: the extensions only make sense when there is an external symbol to link to. If the Go declaration provides its own body or value, there is nothing to link, and the linkname is just a name.

`//llgo:link` is different: it only supports linking to an external symbol, never naming a local one. The body it carries is a fake body (see below), not a real implementation, so the "has a body" rule above does not apply to it.

## `//llgo:link` for methods

`//go:linkname` cannot be applied to methods, because a method has no standalone Go identifier to name. `//llgo:link` extends it for this case. The first argument is written as `TypeName.MethodName`, and the second argument is the same `<callconv>.<name>` linkname as above:

```go
//llgo:link TranslationUnit.Cursor C.clang_getTranslationUnitCursor
func (self TranslationUnit) Cursor() Cursor {
	return Cursor{}
}
```

This binds the Go method `TranslationUnit.Cursor` to the C function `clang_getTranslationUnitCursor`, with the receiver passed as the first C argument.

`//llgo:link` only supports linking to an external symbol. Unlike `//go:linkname`, it has no naming-only mode, so the extended syntax always applies to it.

**About the fake body.** A method annotated with `//llgo:link` carries a placeholder body (`return Cursor{}` above). This exists only to satisfy the Go compiler, which requires a method to have a body. The body has no runtime meaning: LLGo replaces the method with the external symbol.

## Summary of rules

| Target | Directive | Linkname form | Notes |
|--------|-----------|---------------|-------|
| C / C++ function | `//go:linkname` | `C.<manglingName>` | `cdecl`; no body |
| stdcall function | `//go:linkname` | `stdcall.<manglingName>` | `stdcall`; no body |
| Python function | `//go:linkname` | `py.<name>` | needs `const LLGoPackage = "py.<moduleName>"`; no body |
| Compiler instruction | `//go:linkname` | `llgo.<name>` | e.g. `llgo.cstr`; functions only, not `var` |
| Global variable (external) | `//go:linkname` | `<name>` or `<callconv>.<name>` | no initial value; without callconv defaults to `godecl` (a `func` variable is a Go closure); no `llgo.` |
| Function with body / variable with initial value | `//go:linkname` | `<name>` or `<pkgPath>.<name>` | only names the local symbol, standard Go behavior; extensions do **not** apply |
| Method | `//llgo:link` | `<callconv>.<name>` | requires a placeholder body |

## Compatibility

- Standard Go linknames (`pkgPath.Name`) continue to work unchanged, including the naming-only use on declarations with a body or initial value. The extended forms are only recognized when the declaration links to an external symbol and the prefix is one of `C`, `stdcall`, `py` or `llgo`.
- A linkname on a variable without a calling-convention prefix uses `godecl`, as in the official Go compiler.
- Because `//llgo:link` is an ordinary comment to the Go toolchain, packages using it still parse and type-check with the standard Go compiler, which is why the placeholder body is needed.
