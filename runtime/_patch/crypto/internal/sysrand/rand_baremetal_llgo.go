//go:build llgo && baremetal && linux && arm

package sysrand

import (
	"errors"
	_ "unsafe"
)

// Every LLGo baremetal target masquerades as GOOS=linux/GOARCH=arm, so the
// stdlib rand_getrandom.go would otherwise provide read and die on a
// getrandom(2) / /dev/urandom path with no operating system behind it (the
// urandom fallback crashes with a load fault on targets without a console
// file layer). Declaring read here drops the stdlib declaration through the
// source-patch skip rule and routes the entropy request to the runtime
// instead: llgo.idf targets bridge to the ESP-IDF hardware TRNG, every
// other baremetal target fails honestly rather than pretending the bytes
// are random. Hosted targets never match this file's build constraint and
// keep the stdlib implementations untouched.

//go:linkname llgoEntropyFill runtime.llgoEntropyFill
func llgoEntropyFill(b []byte) bool

func read(b []byte) error {
	if llgoEntropyFill(b) {
		return nil
	}
	return errors.New("hardware entropy source unavailable on this llgo baremetal target")
}
