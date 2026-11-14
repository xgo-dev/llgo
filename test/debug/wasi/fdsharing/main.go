// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package main

import (
	"os"
	"runtime"
	"syscall"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "exit":
			os.Exit(19)
		case "worker-exit":
			go func() { os.Exit(23) }()
			select {}
		}
	}
	checkSharedFD("/tmp")
	checkSharedFD(".")
	println("wasi-shared-fd-ok")
}

func checkSharedFD(directory string) {
	ready, start, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var file *os.File
	go func() {
		runtime.LockOSThread()
		close(ready)
		<-start
		if _, err := file.WriteString("shared-wasi-fd"); err != nil {
			panic(err)
		}
		if err := file.Close(); err != nil {
			panic(err)
		}
		close(done)
	}()
	// Create the worker's host instance before adding a descriptor. Copying an
	// instance's old descriptor snapshot cannot satisfy this exchange.
	<-ready
	var err error
	file, err = os.CreateTemp(directory, "llgo-wasi-fd-")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	fd := int(file.Fd())
	close(start)
	<-done
	if err := syscall.Close(fd); err != syscall.EBADF {
		panic("worker close did not remove the process descriptor")
	}
	contents, err := os.ReadFile(file.Name())
	if err != nil || string(contents) != "shared-wasi-fd" {
		panic("worker did not write the process descriptor")
	}
}
