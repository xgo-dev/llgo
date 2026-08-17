package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

func main() {
	ls := "ls"
	if runtime.GOOS == "windows" {
		ls = "dir"
	}
	cmd := exec.Command(ls)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	dir := llvm.New("").BinDir()
	fmt.Println(dir)
}
