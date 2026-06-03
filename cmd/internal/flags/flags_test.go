package flags

import (
	"bytes"
	"flag"
	"testing"

	"github.com/goplus/llgo/internal/optlevel"
)

func TestBuildOptimizationFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want optlevel.Level
	}{
		{name: "O0 bool flag", args: []string{"-O0"}, want: optlevel.O0},
		{name: "O3 bool flag", args: []string{"-O3"}, want: optlevel.O3},
		{name: "Oz bool flag", args: []string{"-Oz"}, want: optlevel.Oz},
		{name: "O equals value", args: []string{"-O=s"}, want: optlevel.Os},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			OptLevel = optlevel.Unset
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			fs.SetOutput(new(bytes.Buffer))
			AddBuildFlags(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse(%v) unexpected error: %v", tt.args, err)
			}
			if OptLevel != tt.want {
				t.Fatalf("OptLevel = %v, want %v", OptLevel, tt.want)
			}
		})
	}
}

func TestBuildOptimizationFlagInvalid(t *testing.T) {
	OptLevel = optlevel.Unset
	fs := flag.NewFlagSet("invalid", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	AddBuildFlags(fs)
	if err := fs.Parse([]string{"-O=fast"}); err == nil {
		t.Fatal("Parse(-O=fast) expected error")
	}
}

func TestBuildOptimizationFlagsMutuallyExclusive(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "bool flags conflict", args: []string{"-O2", "-O3"}},
		{name: "bool and valued conflict", args: []string{"-O3", "-O=2"}},
		{name: "duplicate bool flag", args: []string{"-O2", "-O2"}},
		{name: "duplicate valued flag", args: []string{"-O=2", "-O=2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			OptLevel = optlevel.Unset
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			fs.SetOutput(new(bytes.Buffer))
			AddBuildFlags(fs)
			if err := fs.Parse(tt.args); err == nil {
				t.Fatalf("Parse(%v) expected conflict error", tt.args)
			}
		})
	}
}
