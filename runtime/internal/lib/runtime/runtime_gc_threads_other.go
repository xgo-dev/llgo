//go:build !nogc && !baremetal && !windows && !darwin && !linux

package runtime

func enableForeignThreadRegistration() {}
