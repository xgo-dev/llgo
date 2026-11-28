//go:build llgo && !nogc && !llgo_closure_env_explicit

package llgoext

// Hidden-env targets pass single-pointer closure environments directly without
// allocating a pointer-sized wrapper cell.
const singlePtrClosureEnvAllocs float64 = 0
