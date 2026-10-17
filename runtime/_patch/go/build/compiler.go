package build

// Default follows cmd/go's source and archive conventions. runtime.Compiler
// identifies LLGo, which is not an archive backend understood by go/build.
var Default = func() Context {
	ctx := defaultContext()
	ctx.Compiler = "gc"
	return ctx
}()
