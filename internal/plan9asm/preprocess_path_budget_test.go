package plan9asm

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// The same bounded assembly must not exhaust an observation budget merely
// because its source directory is longer on a hosted runner.
func TestPreprocessSourcePathBudget(t *testing.T) {
	pkg, _, opt := preprocessFixture(t)
	source := []byte("TEXT ·Included(SB),4,$0-0\n" + strings.Repeat("MOVQ $1, AX\n", 40000) + "RET\n")
	if len(source) >= maxAssemblySourceBytes {
		t.Fatal("test source must remain below the existing input bound")
	}
	var first []byte
	root := filepath.VolumeName(t.TempDir()) + string(filepath.Separator)
	for _, directory := range []string{filepath.Join(root, "a"), filepath.Join(root, strings.Repeat("selected-source-directory"+string(filepath.Separator), 8))} {
		pkg.Dir = directory
		got, err := PreprocessSourceForPkg(pkg, filepath.Join(directory, "entry.s"), source, "linux", "amd64", opt)
		if err != nil {
			t.Fatalf("bounded source under %d-byte directory: %v", len(directory), err)
		}
		if first == nil {
			first = got
		} else if !bytes.Equal(first, got) {
			t.Fatal("changing only the source path changed expanded assembly")
		}
	}
}
