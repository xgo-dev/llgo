//go:build !llgo

package main

import "testing"

func TestTargetsCommand(t *testing.T) {
	cmd := &Cmd_targets{App: new(App)}
	cmd.Main("targets")
	if cmd.Command.Command.Use != "targets [-json] [name ...]" || cmd.Command.Command.Short != "List and inspect LLGo target configurations" {
		t.Fatalf("targets metadata = (%q, %q)", cmd.Command.Command.Use, cmd.Command.Command.Short)
	}
	if cmd.Classfname() != "targets" || !cmd.DisableFlagParsing || cmd.Run == nil {
		t.Fatal("targets command was not generated with pass-through parsing")
	}
}
