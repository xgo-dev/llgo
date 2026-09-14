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

## 中文说明

本阶段实现 #2590 的两个函数属性：`cold` 表示调用预计较少，是优化提示；`noreturn` 表示函数不会正常返回，可以 panic 或一直运行。

每个属性独占一行，放在具名函数或方法声明前；支持 `//llgo:` 和 `// llgo:`，可以组合、重复。`noreturn` 是程序员的保证，编译器不证明函数行为，也不插入运行时检查。函数自身通过 defer/recover 正常返回时不能声明该属性；调用者仍可恢复其 panic。

属性适用于定义、导入声明、泛型实例与 linkname，并在 ABI 转换和缓存构建中保留。参数、接收者及返回值属性按 #2590 后续分步实现，目前使用这些指令会报错。
