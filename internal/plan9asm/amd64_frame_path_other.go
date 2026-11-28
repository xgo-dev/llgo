//go:build !windows

package plan9asm

import "path/filepath"

func physicalFramePath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
