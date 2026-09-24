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

Value and pointer attributes are added in separate follow-up changes. Cgo-specific
parsing and handling remain outside this feature.
