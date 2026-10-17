package runtime

// Replace GOROOT's "gc" declaration before importers fold runtime.Compiler.
// Linking the alternative runtime later cannot change those constants.
const Compiler = "llgo"
