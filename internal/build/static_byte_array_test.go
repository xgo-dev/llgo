package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestStaticByteArrayGlobalRoundTrip(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	const n = 4096
	dir := t.TempDir()
	var src strings.Builder
	src.WriteString("package main\n\nvar Table = [...]byte{")
	var first, last byte
	for i := 0; i < n; i++ {
		if i > 0 {
			src.WriteByte(',')
		}
		b := byte(i*17 + 3)
		if i == 0 {
			first = b
		}
		if i == n-1 {
			last = b
		}
		fmt.Fprintf(&src, "%d", b)
	}
	src.WriteString("}\nfunc main() {\n")
	fmt.Fprintf(&src, "\tif Table[0] != %d || Table[%d] != %d {\n\t\tpanic(\"static byte array payload was not restored\")\n\t}\n", first, n-1, last)
	src.WriteString("\tprintln(\"ok\")\n}\n")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src.String()), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.local/staticbytes\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "app")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = out
	if _, err := Build(Invocation{Args: []string{"."}, Dir: dir, Config: conf}); err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, got)
	}
	if strings.TrimSpace(string(got)) != "ok" {
		t.Fatalf("output = %q, want ok", got)
	}
}

func TestStaticByteArrayDistinctMains(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	mod := t.TempDir()
	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.local/tables\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	write := func(pkg string, fill byte) {
		t.Helper()
		dir := filepath.Join(mod, pkg)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		var src strings.Builder
		src.WriteString("package main\n\nvar Table = [...]byte{")
		for i := 0; i < 4096; i++ {
			if i > 0 {
				src.WriteByte(',')
			}
			fmt.Fprintf(&src, "%d", fill)
		}
		src.WriteString("}\nfunc main() { println(Table[0], Table[4095]) }\n")
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src.String()), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("a", 1)
	write("b", 2)

	outDir := filepath.Join(t.TempDir(), "bin") + string(os.PathSeparator)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = outDir
	if _, err := Build(Invocation{Args: []string{"./a", "./b"}, Dir: mod, Config: conf}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		want string
	}{
		{"a", "1 1"},
		{"b", "2 2"},
	} {
		bin := filepath.Join(outDir, tc.name)
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		got, err := exec.Command(bin).CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", tc.name, err, got)
		}
		if strings.TrimSpace(string(got)) != tc.want {
			t.Fatalf("%s output = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestStaticByteArrayLinknameAlias(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	mod := t.TempDir()
	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/linkreview\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(mod, "target"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "target", "table.go"), []byte("package target\n\nvar Table [4096]byte\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var src strings.Builder
	src.WriteString("package main\n\nimport (\n\t\"example.com/linkreview/target\"\n\t_ \"unsafe\"\n)\n\n//go:linkname alias example.com/linkreview/target.Table\nvar alias = [4096]byte{")
	for i := 0; i < 4096; i++ {
		if i > 0 {
			src.WriteByte(',')
		}
		src.WriteByte('1')
	}
	src.WriteString("}\nfunc main() { println(target.Table[0], alias[0]) }\n")
	if err := os.WriteFile(filepath.Join(mod, "main.go"), []byte(src.String()), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "app")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = out
	if _, err := Build(Invocation{Args: []string{"."}, Dir: mod, Config: conf}); err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, got)
	}
	if strings.TrimSpace(string(got)) != "1 1" {
		t.Fatalf("output = %q, want %q", got, "1 1")
	}
}

func TestStaticByteArrayPatchSkipKeepsInit(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	origPath := filepath.Join(runtime.GOROOT(), "src", "internal", "abi", "abi.go")
	orig, err := os.ReadFile(origPath)
	if err != nil {
		t.Skip(err)
	}
	altPath := filepath.Join(root, "runtime", "internal", "lib", "internal", "abi", "abi.go")
	alt, err := os.ReadFile(altPath)
	if err != nil {
		t.Fatal(err)
	}

	const n = 4096
	var lit strings.Builder
	lit.Grow(n * 2)
	for i := 0; i < n; i++ {
		if i > 0 {
			lit.WriteByte(',')
		}
		lit.WriteByte('1')
	}
	orig = append(append([]byte{}, orig...), []byte(fmt.Sprintf("\nvar ReviewStaticTable = [%d]byte{%s}\nfunc init() { println(ReviewStaticTable[0], ReviewStaticTable[%d]) }\n", n, lit.String(), n-1))...)
	alt = append(append([]byte{}, alt...), []byte("\n//llgo:skip ReviewStaticTable\ntype reviewStaticByteArraySkip int\nvar ReviewStaticTable [4096]byte\n")...)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.local/patchskip\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"ok\") }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "app")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = out
	conf.Overlay = map[string][]byte{origPath: orig, altPath: alt}
	var mu sync.Mutex
	var abiStores int
	var sawTable bool
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != "internal/abi" || pkg.SSA == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		_, sawTable = pkg.SSA.Members["ReviewStaticTable"]
		abiStores = countBuildInitStores(pkg.SSA)
	}
	if _, err := Build(Invocation{Args: []string{"."}, Dir: dir, Config: conf}); err != nil {
		t.Fatal(err)
	}
	if !sawTable {
		t.Fatal("internal/abi overlay did not define ReviewStaticTable")
	}
	if abiStores < n {
		t.Fatalf("internal/abi init stores = %d, want at least %d", abiStores, n)
	}
	got, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, got)
	}
	if !strings.Contains(string(got), "1 1") || !strings.Contains(string(got), "ok") {
		t.Fatalf("output = %q, want 1 1 and ok", got)
	}
}

func countBuildInitStores(pkg *ssa.Package) int {
	if pkg == nil {
		return 0
	}
	initFn := pkg.Func("init")
	if initFn == nil {
		return 0
	}
	stores := 0
	for _, b := range initFn.Blocks {
		for _, instr := range b.Instrs {
			if _, ok := instr.(*ssa.Store); ok {
				stores++
			}
		}
	}
	return stores
}
