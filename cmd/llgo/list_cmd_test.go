//go:build !llgo

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestListCommand(t *testing.T) {
	cmd := &Cmd_list{App: new(App)}
	cmd.Main("list")
	if cmd.Command.Command.Use != "list [-target name] [list flags] [packages]" || cmd.Command.Command.Short != "List packages using LLGo source-selection rules" {
		t.Fatalf("list metadata = (%q, %q)", cmd.Command.Command.Use, cmd.Command.Command.Short)
	}
	if cmd.Classfname() != "list" || !cmd.DisableFlagParsing || cmd.Run == nil {
		t.Fatal("list command was not generated with pass-through parsing")
	}
}

func TestListGoAlias(t *testing.T) {
	if os.Getenv("LLGO_TEST_LIST_CHILD") == "1" {
		os.Args = []string{"go", "list", "-f", "{{context.GOARCH}} {{context.Compiler}}", "--", "unsafe"}
		main()
		os.Exit(0)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.Link(self, filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_TEST_LIST_CHILD", "1")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, self, "-test.run=^TestListGoAlias$").CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("list recursed or hung: %v", ctx.Err())
	}
	wantArch := os.Getenv("GOARCH")
	if wantArch == "" {
		wantArch = runtime.GOARCH
	}
	if err != nil || strings.TrimSpace(string(output)) != wantArch+" gc" {
		t.Fatalf("list through go alias: %v, %s", err, output)
	}
}
