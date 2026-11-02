package test

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/mockable"
)

func TestArtifactsUseInvocationOutputDirectory(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", filepath.Clean(filepath.Join(old, "../../..")))
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOWORK", "off")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/artifacts\ngo 1.27\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second"} {
		pkg := filepath.Join(dir, name)
		if err := os.Mkdir(pkg, 0700); err != nil {
			t.Fatal(err)
		}
		source := `package fixture
import ("os"; "path/filepath"; "testing")
func TestArtifact(t *testing.T) {
 if err := os.WriteFile(filepath.Join(t.ArtifactDir(), "result.txt"), []byte("retained"), 0600); err != nil { t.Fatal(err) }
}
`
		if err := os.WriteFile(filepath.Join(pkg, "artifact_test.go"), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Error(err)
		}
	}()
	mockable.EnableMock()
	defer mockable.DisableMock()
	defer resetTestFlags()
	oldExec := *testExec
	defer func() { *testExec = oldExec }()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	logs := t.TempDir()
	t.Setenv("LLGO_TEST_EXEC_FORWARD", "1")
	for _, outputName := range []string{"", "results", "wrapped"} {
		resetTestFlags()
		args := []string{"./...", "-test.artifacts"}
		output := dir
		if outputName != "" {
			output = filepath.Join(dir, outputName)
			if err := os.Mkdir(output, 0700); err != nil {
				t.Fatal(err)
			}
			args = []string{"-artifacts", "./...", "-outputdir=" + outputName}
		}
		if outputName == "wrapped" {
			quote := func(s string) string { return `"` + s + `"` }
			wrapper := strings.Join([]string{quote(self), "-test.run=^TestTestExecHelper$", "--", quote(logs)}, " ")
			args = append([]string{"-exec=" + wrapper}, args...)
		}
		runCmd(Cmd, args)
		files := 0
		err := filepath.WalkDir(filepath.Join(output, "_artifacts"), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Name() == "result.txt" {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if string(data) != "retained" {
					t.Errorf("artifact %s = %q", path, data)
				}
				files++
			}
			return nil
		})
		if err != nil || files != 2 {
			t.Fatalf("package artifacts in %s = %d, want 2: %v", output, files, err)
		}
	}
	entries, err := os.ReadDir(logs)
	if err != nil || len(entries) != 2 {
		t.Fatalf("execution wrapper logs = %d, want 2: %v", len(entries), err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(logs, entry.Name()))
		var args []string
		wantDir, wantErr := os.Stat(filepath.Join(dir, "wrapped"))
		if err != nil || json.Unmarshal(data, &args) != nil || len(args) == 0 ||
			!slices.Contains(args, "-test.artifacts") || wantErr != nil {
			t.Fatalf("wrapper lost driver-generated binary flags: %q, %v", data, err)
		}
		outputDir := ""
		for _, arg := range args {
			if strings.HasPrefix(arg, "-test.outputdir=") {
				outputDir = strings.TrimPrefix(arg, "-test.outputdir=")
			}
		}
		gotDir, err := os.Stat(outputDir)
		if err != nil || !filepath.IsAbs(outputDir) || !os.SameFile(gotDir, wantDir) {
			t.Fatalf("wrapper output directory = %q, want %q: %v", outputDir, filepath.Join(dir, "wrapped"), err)
		}
	}
}

func TestTestExecHelper(t *testing.T) {
	if os.Getenv("LLGO_TEST_EXEC_FORWARD") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			args := os.Args[i+1:]
			file, err := os.CreateTemp(args[0], "wrapper-*.json")
			if err != nil || json.NewEncoder(file).Encode(args[1:]) != nil {
				os.Exit(98)
			}
			file.Close()
			command := exec.Command(args[1], args[2:]...)
			command.Stdout, command.Stderr, command.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := command.Run(); err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					os.Exit(exit.ExitCode())
				}
				os.Exit(97)
			}
			os.Exit(0)
		}
	}
	os.Exit(99)
}
