//go:build !llgo

package run

import "testing"

func TestRunAndCmpTestBuildFlagsAreIndependent(t *testing.T) {
	if runGoBuildFlags == cmpTestGoBuildFlags {
		t.Fatal("run and cmptest share a build-flag collector")
	}
	if Cmd.Flag.Lookup("ldflags") == nil || CmpTestCmd.Flag.Lookup("ldflags") == nil {
		t.Fatal("run or cmptest does not register -ldflags")
	}
	if Cmd.Flag.Lookup("timeout") == nil {
		t.Fatal("run does not register -timeout")
	}
	if CmpTestCmd.Flag.Lookup("timeout") != nil {
		t.Fatal("cmptest unexpectedly registers the run-only -timeout")
	}
	if runGoBuildFlags.Flag != &Cmd.Flag || cmpTestGoBuildFlags.Flag != &CmpTestCmd.Flag {
		t.Fatal("run or cmptest build flags are bound to the wrong command")
	}
}
