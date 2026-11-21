package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

func main() {
	if n, err := fmt.Printf("literal %%\n"); n != len("literal %\n") || err != nil {
		panic("incorrect literal write result")
	}
	if n, err := fmt.Printf(""); n != 0 || err != nil {
		panic("incorrect empty write result")
	}
	if len(os.Args) > 1 {
		n, err := fmt.Printf("no progress")
		checkWriteError(n, err, io.ErrUnexpectedEOF)
		checkOversizedWrite()
		if n, err := fmt.Printf("unlocked"); n != len("unlocked") || err != nil {
			panic("write failed after recovering from an oversized result")
		}
	}
	if err := syscall.Close(1); err != nil {
		panic(err)
	}
	n, err := fmt.Printf("closed descriptor")
	checkWriteError(n, err, syscall.EBADF)
	println("stdout count and path error ok")
}

func checkWriteError(n int, err, want error) {
	p, ok := err.(*os.PathError)
	if n != 0 || !ok || p.Op != "write" || p.Path != "/dev/stdout" || p.Err != want {
		panic("incorrect write error")
	}
}

func checkOversizedWrite() {
	defer func() {
		if recover() != "invalid return from write: got 91 from a write of 9" {
			panic("incorrect oversized write panic")
		}
	}()
	fmt.Printf("oversized")
}
