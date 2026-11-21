//go:build llgo && wasm

package callback

import _ "unsafe"

//llgo:cdecl
type StringSource func() string

//llgo:cdecl
type SliceSource func() []byte

//go:linkname matchSources C.llgo_test_match_sources
func matchSources(second int32, text StringSource, data SliceSource) int32

// MatchSources converts ordinary Go funcvals back to C callbacks in a different
// package from the caller that converted the native pointers to Go funcvals.
//
//go:noinline
func MatchSources(second int32, text func() string, data func() []byte) bool {
	return matchSources(second, StringSource(text), SliceSource(data)) != 0
}

// Box's layout is independent of the function type used as its type argument.
type Box[T any] struct{ Value int32 }

type argument = Box[func(right uint8)]

//llgo:cdecl
type genericSource func(argument) string

//go:linkname matchGeneric C.llgo_test_match_generic
func matchGeneric(genericSource) int32

//go:noinline
func MatchGeneric(f func(argument) string) bool {
	return f(argument{Value: 17}) == "ok" && matchGeneric(genericSource(f)) != 0
}
