package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		fmt.Println("child process")
		os.Exit(7)
	}
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	cmd := exec.Command(exe, "child")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if _, ok := err.(*exec.ExitError); !ok {
		panic(fmt.Sprintf("expected child exit error, got %v", err))
	}
	fmt.Println("child exit:", cmd.ProcessState.ExitCode())

	dir := llvm.New("").BinDir()
	fmt.Println(dir)
}
