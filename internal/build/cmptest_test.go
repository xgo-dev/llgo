package build

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCmpTestExitStatus(t *testing.T) {
	if code := os.Getenv("LLGO_CMPTEST_HELPER_EXIT"); code != "" {
		n, err := strconv.Atoi(code)
		if err != nil {
			panic(err)
		}
		os.Exit(n)
	}
	exit := func(code string) error {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCmpTestExitStatus$")
		cmd.Env = append(os.Environ(), "LLGO_CMPTEST_HELPER_EXIT="+code)
		return cmd.Run()
	}
	one, otherOne, seven := exit("1"), exit("1"), exit("7")
	for _, err := range []error{one, otherOne, seven} {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("helper did not exit normally: %v", err)
		}
	}
	for _, tc := range []struct {
		name       string
		got, want  error
		mismatched bool
	}{
		{"success", nil, nil, false},
		{"same nonzero status", one, otherOne, false},
		{"different nonzero status", one, seven, true},
		{"unexpected failure", one, nil, true},
		{"unexpected success", nil, one, true},
		{"launch failure", errors.New("cannot start executable"), nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if panicked := recover() != nil; panicked != tc.mismatched {
					t.Fatalf("comparison failed = %v, want %v", panicked, tc.mismatched)
				}
			}()
			checkEqualRunErr(tc.got, tc.want)
		})
	}
}

func TestCmpTestReferenceBuild(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module cmptest\n\ngo 1.20\n",
		"main.go": `package main
import "os"
func main() {
 if len(os.Args) != 2 || os.Args[1] != "-test.run=^TestCmpTestExitStatus$" {
  panic("reference arguments were not forwarded")
 }
}`,
		"broken.go": "package main\nnot valid Go source\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	compare := func(file string) {
		commands := commandEnv{dir: dir, environ: append(os.Environ(), "LLGO_CMPTEST_HELPER_EXIT=0")}
		cmpTest(commands, dir, ".", os.Args[0], false, []string{"-test.run=^TestCmpTestExitStatus$"}, file)
	}
	t.Run("selected files and program arguments", func(t *testing.T) {
		// The invalid neighboring file must not be part of the Go reference.
		compare("main.go")
	})
	t.Run("reference compilation failure", func(t *testing.T) {
		defer func() {
			if got := fmt.Sprint(recover()); !strings.Contains(got, "build cmptest reference:") {
				t.Fatalf("reference build failure was not reported: %s", got)
			}
		}()
		compare("broken.go")
	})
	t.Run("temporary executable directory failure", func(t *testing.T) {
		// Point all OS-specific temp variables at an existing regular file.
		for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
			t.Setenv(key, filepath.Join(dir, "main.go"))
		}
		defer func() {
			if got := fmt.Sprint(recover()); !strings.Contains(got, "main.go") {
				t.Fatalf("temporary directory failure was not reported: %s", got)
			}
		}()
		compare("main.go")
	})
}
