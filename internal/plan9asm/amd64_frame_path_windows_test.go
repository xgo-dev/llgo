//go:build windows

package plan9asm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPhysicalFramePathResolvesJunction(t *testing.T) {
	root := t.TempDir()
	original, alias := filepath.Join(root, "original"), filepath.Join(root, "alias")
	if err := os.Mkdir(original, 0700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", alias, original).CombinedOutput(); err != nil {
		t.Fatalf("create directory junction: %v\n%s", err, output)
	}
	defer os.Remove(alias)
	for _, name := range []string{"", "go.exe"} {
		if name != "" {
			if err := os.WriteFile(filepath.Join(original, name), []byte("identity"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		want, err := physicalFramePath(filepath.Join(original, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := physicalFramePath(filepath.Join(alias, name))
		if err != nil || got != want {
			t.Fatalf("junction path = %q, %v, want %q", got, err, want)
		}
		again, err := physicalFramePath(got)
		if err != nil || again != got {
			t.Fatalf("physical spelling is unstable: %q, %v", again, err)
		}
	}
}
