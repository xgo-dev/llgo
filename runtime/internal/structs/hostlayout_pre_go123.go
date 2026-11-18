//go:build !go1.23

package structs

// HostLayout is the compiler-recognized compatibility form of structs.HostLayout.
type HostLayout struct{ _ hostLayout }
type hostLayout struct{}
