package clang

import (
	"slices"
	"testing"
)

func TestCmdMergesFlagsFromExplicitEnv(t *testing.T) {
	t.Setenv("CCFLAGS", "-DPROCESS_CCFLAG")
	t.Setenv("CFLAGS", "-DPROCESS_CFLAG")
	t.Setenv("LDFLAGS", "-L/process/lib")

	cmd := New("clang", Config{
		CCFLAGS: []string{"-DCFG_CCFLAG"},
		CFLAGS:  []string{"-DCFG_CFLAG"},
		LDFLAGS: []string{"-L/cfg/lib"},
	})
	cmd.Env = []string{
		"CCFLAGS=-DENV_CCFLAG",
		"CFLAGS=-DENV_CFLAG",
		"LDFLAGS=-L/env/lib",
	}

	gotCompile := cmd.mergeCompilerFlags()
	wantCompile := []string{"-DENV_CCFLAG", "-DENV_CFLAG", "-DCFG_CCFLAG", "-DCFG_CFLAG"}
	if !slices.Equal(gotCompile, wantCompile) {
		t.Fatalf("mergeCompilerFlags() = %v, want %v", gotCompile, wantCompile)
	}

	gotLink := cmd.mergeLinkerFlags()
	wantLink := []string{"-DENV_CCFLAG", "-L/env/lib", "-L/cfg/lib"}
	if !slices.Equal(gotLink, wantLink) {
		t.Fatalf("mergeLinkerFlags() = %v, want %v", gotLink, wantLink)
	}
}
