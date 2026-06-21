package flags

import (
	"bytes"
	"flag"
	"testing"

	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/lto"
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

func TestBuildLTOFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      lto.Mode
		specified bool
	}{
		{name: "default off", args: nil, want: lto.Off, specified: false},
		{name: "thin value", args: []string{"-lto=thin"}, want: lto.Thin, specified: true},
		{name: "full value", args: []string{"-lto=full"}, want: lto.Full, specified: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			fs.SetOutput(new(bytes.Buffer))
			AddBuildFlags(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse(%v) unexpected error: %v", tt.args, err)
			}
			if LTO.Specified != tt.specified {
				t.Fatalf("LTO.Specified = %v, want %v", LTO.Specified, tt.specified)
			}
			if LTO.Mode != tt.want {
				t.Fatalf("LTO.Mode = %v, want %v", LTO.Mode, tt.want)
			}
			conf := &build.Config{}
			if err := UpdateConfig(conf); err != nil {
				t.Fatalf("UpdateConfig error: %v", err)
			}
			if conf.LTO != tt.want {
				t.Fatalf("conf.LTO = %v, want %v", conf.LTO, tt.want)
			}
		})
	}
}

func TestBuildLTOFlagInvalid(t *testing.T) {
	tests := [][]string{
		{"-lto"},
		{"-lto=true"},
		{"-lto=false"},
		{"-lto=off"},
		{"-lto=fat"},
	}
	for _, args := range tests {
		fs := flag.NewFlagSet("invalid-lto", flag.ContinueOnError)
		fs.SetOutput(new(bytes.Buffer))
		AddBuildFlags(fs)
		if err := fs.Parse(args); err == nil {
			t.Fatalf("Parse(%v) expected error", args)
		}
	}
}

func TestUpdateConfigPreservesLTOWhenUnspecified(t *testing.T) {
	fs := flag.NewFlagSet("lto-unspecified", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	AddBuildFlags(fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("Parse(nil) unexpected error: %v", err)
	}

	conf := &build.Config{LTO: lto.Full}
	if err := UpdateConfig(conf); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}
	if conf.LTO != lto.Full {
		t.Fatalf("conf.LTO = %v, want %v", conf.LTO, lto.Full)
	}
}
