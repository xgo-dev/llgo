package runtime

// Replace GOROOT's "gc" declaration before importers fold runtime.Compiler.
// Linking the alternative runtime later cannot change those constants.
// Keep the value in sync with runtime/internal/lib/runtime/compiler.go,
// which supplies the compiler identity in that linked runtime.
const Compiler = "llgo"
