//go:build go1.25
// +build go1.25

package runtime

func SetDefaultGOMAXPROCS() {
	GOMAXPROCS(gomaxprocsDefault())
}
