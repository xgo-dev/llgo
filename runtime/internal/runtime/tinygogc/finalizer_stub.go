//go:build baremetal && !nogc

package tinygogc

func noteFinalizerReference(uintptr) {}
