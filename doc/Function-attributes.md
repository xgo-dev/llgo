# Function attributes

This implements the function attributes in [proposal #2590](https://github.com/xgo-dev/llgo/issues/2590).

Write each attribute on its own line immediately before a named function or method declaration. Both `//llgo:` and `// llgo:` are accepted. Attributes can be combined and repeated.

```go
//llgo:cold
//llgo:noreturn
func Fatal(message string) {
    panic(message)
}
```

| Attribute | Meaning |
| --- | --- |
| `cold` | Calls are expected to be uncommon; this is an optimization hint. |
| `noreturn` | The function never returns normally. It may panic or run indefinitely. |

`noreturn` is a programmer promise. LLGo checks the declaration and directive syntax, but does not prove the function's behavior or add runtime checks. A function whose deferred recovery permits a normal return must not declare `noreturn`. Callers may treat the normal continuation after a `noreturn` call as unreachable; panic unwinding and recovery by the caller remain valid.

The properties apply to definitions and imported declarations, including methods, generic instances and declarations connected by linkname. They are preserved by ABI conversion and cached builds. The runtime uses the same source directives as ordinary packages.

Parameter, receiver and result attributes will be implemented in subsequent steps of #2590. Their directives currently produce a diagnostic.

