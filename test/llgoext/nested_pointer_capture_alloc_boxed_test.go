//go:build llgo && !nogc && llgo_closure_env_explicit

package llgoext

// Explicit-env targets keep a boxed wrapper cell for single-pointer captures.
const singlePtrClosureEnvAllocs float64 = 1
