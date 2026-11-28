package mod

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/mockable"
)

func setupModuleEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GOENV", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOFLAGS", "")
	t.Chdir(t.TempDir())
}

func TestModMatchesGo(t *testing.T) {
	setupModuleEnv(t)
	for _, args := range [][]string{
		nil,
		{"-h"},
		{"tidy", "-h"},
		{"unknown-subcommand"},
		{"tidy", "-unknown-flag"},
		{"edit", "-json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var wantOut, wantErr, gotOut, gotErr bytes.Buffer
			cmd := exec.Command("go", append([]string{"mod"}, args...)...)
			cmd.Stdout, cmd.Stderr = &wantOut, &wantErr
			want := cmd.Run()
			got := run(args, nil, &gotOut, &gotErr)
			if exitCode(got) != exitCode(want) || gotOut.String() != wantOut.String() || gotErr.String() != wantErr.String() {
				t.Fatalf("mod %q: got (%q, %q, %v), want (%q, %q, %v)", args, &gotOut, &gotErr, got, &wantOut, &wantErr, want)
			}
		})
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	return -1
}

func TestModInitEditTidyVendorLLGoDependency(t *testing.T) {
	setupModuleEnv(t)
	// Module maintenance must retain imports behind the llgo tag even though
	// the delegated Go command is not given LLGo-specific build flags.
	for name, content := range map[string]string{
		"main.go":    "package main\nfunc main() {}\n",
		"llgo.go":    "//go:build llgo\n\npackage main\nimport _ \"example.com/dep\"\n",
		"dep/go.mod": "module example.com/dep\n\ngo 1.20\n",
		"dep/dep.go": "package dep\n",
	} {
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "example.com/app"},
		{"edit", "-replace=example.com/dep=./dep"},
		{"tidy"},
		{"vendor"},
	} {
		var output bytes.Buffer
		if err := run(args, nil, &output, &output); err != nil {
			t.Fatalf("mod %q: %v\n%s", args, err, &output)
		}
	}
	var output, diagnostic bytes.Buffer
	if err := run([]string{"edit", "-json"}, nil, &output, &diagnostic); err != nil {
		t.Fatalf("mod edit -json: %v\n%s", err, &diagnostic)
	}
	var info struct {
		Module  struct{ Path string }
		Require []struct{ Path string }
		Replace []struct{ Old, New struct{ Path string } }
	}
	if err := json.Unmarshal(output.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.Module.Path != "example.com/app" || len(info.Require) != 1 || info.Require[0].Path != "example.com/dep" || len(info.Replace) != 1 || info.Replace[0].New.Path != "./dep" {
		t.Fatalf("unexpected module after tidy: %s", &output)
	}
	data, err := os.ReadFile(filepath.Join("vendor", "example.com", "dep", "dep.go"))
	if err != nil || string(data) != "package dep\n" {
		t.Fatalf("vendor omitted llgo-only dependency: %q, %v", data, err)
	}
}

func TestMainExitStatus(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		missingGo bool
		want      int
	}{
		{name: "success", args: []string{"init", "example.com/app"}},
		{name: "invalid flag", args: []string{"tidy", "-unknown-flag"}, want: 2},
		{name: "missing module", args: []string{"tidy"}, want: 1},
		{name: "missing Go", missingGo: true, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupModuleEnv(t)
			if tc.missingGo {
				t.Setenv("PATH", t.TempDir())
			}
			mockable.EnableMock()
			defer mockable.DisableMock()
			defer func() {
				panicked := recover()
				if tc.want == 0 && panicked != nil || tc.want != 0 && panicked != "exit" || mockable.ExitCode() != tc.want {
					t.Fatalf("panic = %v, exit = %d, want %d", panicked, mockable.ExitCode(), tc.want)
				}
			}()
			Main(tc.args)
		})
	}
}
