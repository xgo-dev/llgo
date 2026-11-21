package main

import (
	"fmt"
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
	if err := syscall.Close(1); err != nil {
		panic(err)
	}
	n, err := fmt.Printf("closed descriptor")
	p, ok := err.(*os.PathError)
	if n != 0 || !ok || p.Op != "write" || p.Path != "/dev/stdout" || p.Err != syscall.EBADF {
		panic("incorrect closed descriptor result")
	}
	println("stdout count and path error ok")
}
