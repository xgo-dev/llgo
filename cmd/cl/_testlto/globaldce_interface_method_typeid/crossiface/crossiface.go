package crossiface

type Wide interface {
	M() int
	N() int
}

// Call deliberately lives in a dependency module while its concrete
// implementation lives in main. The Full LTO test therefore proves that the
// checked-load interface attributes survive package bitcode linking.
//
//go:noinline
func Call(v Wide) int {
	return v.M()
}
