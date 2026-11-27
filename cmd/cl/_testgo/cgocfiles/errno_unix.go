//go:build !windows && !baremetal

package main

import "syscall"

func init() {
	checkErrno = requireEACCES
}

func requireEACCES(err error) {
	if err != syscall.EACCES {
		panic("errno aggregate wrapper")
	}
}
