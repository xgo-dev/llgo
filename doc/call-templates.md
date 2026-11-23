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
//llgointernal:template:gofmt format=format args=a
func Appendf(b []byte, format string, a ...any) []byte

//llgointernal:template:cfmt format=pattern args=__llgo_va_list
//go:linkname Snprintf C.snprintf
func Snprintf(dst *Char, n uintptr, pattern *Char, __llgo_va_list ...any) Int
```

LLGo applies equivalent bindings to the supported standard `fmt` and `lib/c` declarations. Format specialization runs in the entry package for `-Oz` builds with dead-code dropping and Full LTO.

- Go formatting uses the constant format and static operand types to emit typed calls for `Printf`, `Sprintf`, `Appendf`, and `Fprintf` with a standard `*os.File`. It supports sequential boolean, integer and string operands, escaped percent signs, literal width and zero padding. Calls using dynamic formats, custom formatting methods or unsupported verbs keep the existing path.
- C formatting folds C string literals and matching 32-bit integer constants into a constant format, with remaining operands forwarded using `__llgo_va_list`. The original libc call handles output, truncation and return values. Positional arguments, dynamic widths and unsupported format syntax keep the original call.
