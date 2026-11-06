// Package typesizes shares LLGo's source layout with the go/types source patch.
// The generated adapter has no LLVM dependency and can run before types have
// finished checking.
//
//go:generate go run generate.go
package typesizes
