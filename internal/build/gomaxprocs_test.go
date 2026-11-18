package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestGOMAXPROCSStartupEnvironment(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module gomaxprocsstartup\n\ngo 1.27\n",
		"main.go": `package main
import (
 "os"
 "runtime"
)
func init() { os.Setenv("GOMAXPROCS", "9") }
func main() {
 println(runtime.GOMAXPROCS(0), runtime.GOMAXPROCS(1), runtime.GOMAXPROCS(-1), runtime.NumCPU())
 runtime.SetDefaultGOMAXPROCS()
 println(runtime.GOMAXPROCS(0))
}
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "startup")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value       string
		want        int
		wantDefault bool
	}{
		{"", 0, true}, {"0", 0, true}, {"-1", 0, true}, {"+4", 0, true}, {"invalid", 0, true},
		{" 4", 0, true}, {"4 ", 0, true}, {"4x", 0, true}, {"0x4", 0, true}, {"1.5", 0, true},
		{"2147483648", 0, true}, {"999999999999999999999999999999999999", 0, true},
		{"1", 1, false}, {"4", 4, false}, {"0004", 4, false}, {"2147483647", 2147483647, false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			cmd := exec.Command(conf.OutFile)
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "GOMAXPROCS=") {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			cmd.Env = append(cmd.Env, "GOMAXPROCS="+tc.value)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("startup: %v\n%s", err, out)
			}
			fields := strings.Fields(string(out))
			if len(fields) != 5 {
				t.Fatalf("unexpected startup output: %s", out)
			}
			values := make([]int, len(fields))
			for i, field := range fields {
				values[i], err = strconv.Atoi(field)
				if err != nil {
					t.Fatal(err)
				}
			}
			want := tc.want
			if tc.wantDefault {
				want = values[3]
			}
			if values[0] != want || values[1] != want || values[2] != 1 || values[3] < 1 || values[4] != values[3] {
				t.Fatalf("startup GOMAXPROCS=%q: got %v, want [%d %d 1 NumCPU NumCPU]", tc.value, values, want, want)
			}
		})
	}
}
