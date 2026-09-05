//go:build baremetal && !nogc

package tinygogc

func preserveFinalizableObjects() {}

func scheduleFinalizers() {}

func noteFinalizerReference(uintptr) {}
