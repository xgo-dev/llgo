# Call templates

`//llgo:template` associates a function with typed implementations declared in the same package. LLGo selects an implementation at a direct call when the compile-time argument types satisfy its signature.

```go
//llgo:template encodeInt encodePair encodeSlice
//llgo:template encodeCompact if compact == true
func Encode(a, b any, compact bool) string { /* generic implementation */ }

func encodeInt(a int, b any, compact bool) string { /* ... */ }
func encodePair(a int, b string, compact bool) string { /* ... */ }
func encodeSlice[T comparable](a []T, b []T, compact bool) string { /* ... */ }
func encodeCompact(a []byte, b any, compact bool) string { /* ... */ }
```

The implementations keep the original parameter count and result types. An interface parameter may stay unchanged or be narrowed to an exact concrete type. LLGo uses ordinary Go inference and constraint checking for a generic implementation. If multiple implementations match, one must be strictly more specific through the parameters it narrows or its constant guard. Calls with no unique match, a function value, or an interface argument in a narrowed position use the original function. Template definitions are visible across package imports, and specialization is emitted in the calling package.

## Format templates

The internal format directives bind roles by their parameter names. For example, `fmt.Appendf` has an argument before the format string, and `snprintf` has two.

```go
//llgointernal:template:gofmt format=format args=a result=append
func Appendf(b []byte, format string, a ...any) []byte

//llgointernal:template:cfmt format=pattern args=__llgo_va_list
//go:linkname Snprintf C.snprintf
func Snprintf(dst *Char, n uintptr, pattern *Char, __llgo_va_list ...any) Int
```

The `fmt` source patch binds the standard entry points by name. LLGo derives the supported `lib/c` bindings from their C symbols and variadic signatures. Format specialization runs in the entry package for `-Oz` builds with dead-code dropping and Full LTO.

- Go formatting uses the constant format and static operand types to emit typed calls for `Printf`, `Sprintf`, `Appendf`, and `Fprintf` with a standard `*os.File`. It supports sequential boolean, integer and string operands, escaped percent signs, literal width and zero padding. Calls using dynamic formats, custom formatting methods or unsupported verbs keep the existing path.
- C formatting folds C string literals and matching 32-bit integer constants into a constant format, with remaining operands forwarded using `__llgo_va_list`. A shared literal stays in the argument list when substituting it would duplicate its bytes. The original libc call handles output, truncation and return values. Positional arguments, dynamic widths and unsupported format syntax keep the original call.


## Source-patch templates

A standard-library patch can bind a typed implementation to an existing declaration. The implementation's signature uses the same matching rules as `//llgo:template`. Format bindings also support a named format parameter and variadic argument list.

```go
//llgointernal:template:bind Size any helper=llgoSize body=llgoSizeBody
//llgointernal:template:bind Read any fallback
//llgointernal:template:bind Appendf gofmt format=format args=a result=append
```

An `any` body is a Go string constant interpreted at compile time using `text/template`. The engine creates a caller-local function with the selected signature and gives its arguments the names `a0`, `a1`, and so on. The body can inspect instantiated argument types, array lengths, field names and tags, and value or pointer method sets. It can emit shared typed helper functions and decline a call with `reject`. Generated source passes the ordinary Go type checker before it is bound to any call.

In a Full LTO size build, a provider's bindings apply when all uses of its declared entries can be specialized in the executable package. An indirect call, a linker alias, or an entry marked `fallback` keeps the provider's generic implementation. The patch supplies the binary wire layout and JSON codec rules. The generated code and cache records belong to the executable package, so other clients can reuse the same patched library archive.
