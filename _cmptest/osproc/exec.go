package main

import (
	"fmt"
	"os"
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
	proc, err := os.StartProcess(exe, []string{exe, "child"}, &os.ProcAttr{
		Files: []*os.File{nil, os.Stdout, os.Stderr},
	})
	if err != nil {
		panic(err)
	}
	state, err := proc.Wait()
	if err != nil {
		panic(err)
	}
	fmt.Println("child exit:", state.ExitCode())
}
