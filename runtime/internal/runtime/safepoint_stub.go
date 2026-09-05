//go:build !llgo

package runtime

// CooperativeSafepoint keeps compiler-only host tests type-correct.
func CooperativeSafepoint() {}
