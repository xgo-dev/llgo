package build

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestCmpTestExitHelper(t *testing.T) {
	if code := os.Getenv("LLGO_CMPTEST_HELPER_EXIT"); code != "" {
		n, err := strconv.Atoi(code)
		if err != nil {
			panic(err)
		}
		os.Exit(n)
	}
}

func TestCmpTestExitStatus(t *testing.T) {
	exit := func(code string) error {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCmpTestExitHelper$")
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
