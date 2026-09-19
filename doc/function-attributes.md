# Function attributes

LLGo accepts `cold` and `noreturn` on named function and method declarations,
including external declarations. Write each attribute on its own comment line:

```go
//llgo:cold
//llgo:noreturn
func Fatal(message string) {
    panic(message)
}
```

The spaced spelling `// llgo:cold` is also accepted. These attributes take no
arguments. Identical repetitions are harmless, and their order does not matter.

- `cold` is an optimization hint that calls are uncommon.
- `noreturn` promises that the function never returns normally. It may panic or
  run indefinitely. Panic unwinding and deferred functions retain their normal
  behavior. A function that recovers its own panic and returns does not satisfy
  this promise.

Attributes are programmer promises, not runtime checks. An incorrect `noreturn`
annotation can cause incorrect optimization. The compiler checks annotation
placement and syntax; it does not prove the behavior of arbitrary function bodies.

Imported declarations and generic instances retain the attributes of their source
declarations. Package patches select the replacement declaration's annotations.
An omitted annotation supplies no guarantee. Source declarations remain separate
even when `go:linkname` makes them refer to one backend symbol; each promise must
be valid for the implementation it names. An indirect call with an unknown target
does not acquire guarantees merely from its Go function type.

For embedders using `cl.Options`, share a `FunctionAttributes` index between
`ParsePkgSyntaxWithOptions` calls and package compilations to provide annotations
for imports without Go SSA syntax. Finish populating it before concurrent backend
compilation. The normal build driver does this automatically, including cached
dependencies. One-shot compilation collects the current package's comments itself.
