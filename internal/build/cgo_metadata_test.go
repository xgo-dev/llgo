package build

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCgoMetadataEnvironmentScope(t *testing.T) {
	base := []string{"CGO_ENABLED=1", "CC=clang --target=i686-pc-windows-msvc"}
	for _, tc := range []struct{ version, triple string }{
		{"go1.27.0", "i686-pc-windows-msvc"},
		{"go1.26.7", "i686-w64-windows-gnu"},
		{"go1.26.7", "aarch64-unknown-linux-gnu"},
	} {
		if got := cgoMetadataEnvironment(base, tc.version, tc.triple); !reflect.DeepEqual(got, base) {
			t.Fatalf("changed unrelated metadata environment for %+v", tc)
		}
	}
	before := append([]string{}, base...)
	got := cgoMetadataEnvironment(base, "go1.26.7", "i686-pc-windows-msvc")
	if !reflect.DeepEqual(base, before) || (commandEnv{environ: got}).lookup("CC") != (commandEnv{environ: base}).lookup("CC") {
		t.Fatal("changed the native compiler or its environment")
	}
	custom := withEnv(base, "CCC_OVERRIDE_OPTIONS=# +-DKEEP=1")
	value := (commandEnv{environ: cgoMetadataEnvironment(custom, "go1.26.7", "i686-pc-windows-msvc")}).lookup("CCC_OVERRIDE_OPTIONS")
	if value != "# +-DKEEP=1 x-mthreads" {
		t.Fatal("lost existing Clang overrides", value)
	}
}

func TestGo126MSVCCgoMetadataCompiler(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		triple  string
		machine uint16
	}{
		{"i686-pc-windows-msvc", pe.IMAGE_FILE_MACHINE_I386},
		{"x86_64-pc-windows-msvc", pe.IMAGE_FILE_MACHINE_AMD64},
		{"aarch64-pc-windows-msvc", pe.IMAGE_FILE_MACHINE_ARM64},
	} {
		t.Run(target.triple, func(t *testing.T) {
			object := filepath.Join(t.TempDir(), "metadata.obj")
			cmd := exec.Command(clang, "--target="+target.triple, "-mthreads", "-Werror", "-x", "c", "-c", "-", "-o", object)
			cmd.Stdin = strings.NewReader("#ifndef _MSC_VER\n#error wrong C ABI\n#endif\n#ifdef __MINGW32__\n#error GNU C ABI selected\n#endif\nint value;\n")
			cmd.Env = cgoMetadataEnvironment(withEnv(os.Environ(), "CGO_ENABLED=1", "CCC_OVERRIDE_OPTIONS="), "go1.26.7", target.triple)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("old Go cgo flags failed: %v\n%s", err, output)
			}
			file, err := pe.Open(object)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if file.Machine != target.machine {
				t.Fatalf("compiler target changed: %x, want %x", file.Machine, target.machine)
			}
		})
	}
}
