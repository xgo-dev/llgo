package main

import (
	"os"
	"runtime"
	"syscall"
	"time"
)

func probe(retry bool) bool {
	for _, six := range []bool{false, true} {
		var r1, r2 uintptr
		var err syscall.Errno
		if six {
			r1, r2, err = syscall.AllThreadsSyscall6(syscall.SYS_GETPID, 0, 0, 0, 0, 0, 0)
		} else {
			r1, r2, err = syscall.AllThreadsSyscall(syscall.SYS_GETPID, 0, 0, 0)
		}
		if retry && (err == syscall.EAGAIN || err == syscall.ENOTSUP) {
			return false
		}
		if err != 0 || r1 != uintptr(os.Getpid()) || r2 != 0 {
			println("getpid", r1, r2, uintptr(err))
			panic("controlled all-thread result")
		}
	}
	return true
}

func await() {
	deadline := time.Now().Add(5 * time.Second)
	for !probe(true) {
		if time.Now().After(deadline) {
			probe(false)
			panic("controlled cohort never became available")
		}
		time.Sleep(time.Millisecond)
	}
}

func main() {
	ready, release, done := make(chan bool, 4), make(chan bool), make(chan bool, 4)
	for id := 0; id < 4; id++ {
		go func(id int) {
			if id&1 == 0 {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
			}
			ready <- true
			<-release
			runtime.GC()
			done <- true
		}(id)
	}
	for range 4 {
		<-ready
	}
	await()
	for range 20 {
		probe(false)
	}
	// No-op credentials exercise the real kernel setter without changing IDs.
	if r1, _, err := syscall.AllThreadsSyscall6(syscall.SYS_SETRESUID,
		^uintptr(0), ^uintptr(0), ^uintptr(0), 0, 0, 0); err != 0 || r1 != 0 {
		panic("controlled credentials")
	}
	mask := syscall.Umask(0077)
	_, _, err := syscall.AllThreadsSyscall(syscall.SYS_UMASK, 0022, 0, 0)
	if after := syscall.Umask(mask); err != syscall.ENOTSUP || after != 0077 {
		panic("unlisted syscall executed")
	}
	finished := make(chan bool)
	go func() {
		for range 30 {
			exited := make(chan bool)
			go func() { runtime.GC(); exited <- true }()
			<-exited
		}
		finished <- true
	}()
	for range 30 {
		probe(true)
		runtime.Gosched()
	}
	<-finished
	close(release)
	for range 4 {
		<-done
	}
	await()
	println("allthreads-controlled-ok")
}
