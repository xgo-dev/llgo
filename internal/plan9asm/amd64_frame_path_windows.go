//go:build windows

package plan9asm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// Go's EvalSymlinks can retain Windows directory junctions and short names.
// The ABI0 source owner requires the opened SDK/package's physical spelling.
func physicalFramePath(path string) (resolved string, resultErr error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	buffer := make([]uint16, 32768)
	n, err := windows.GetFinalPathNameByHandle(windows.Handle(file.Fd()), &buffer[0], uint32(len(buffer)), 0)
	if err != nil {
		return "", err
	}
	if n == 0 || n >= uint32(len(buffer)) {
		return "", fmt.Errorf("physical frame path exceeds Windows path bound")
	}
	resolved = windows.UTF16ToString(buffer[:n])
	if strings.HasPrefix(resolved, `\\?\UNC\`) {
		resolved = `\\` + strings.TrimPrefix(resolved, `\\?\UNC\`)
	} else if strings.HasPrefix(resolved, `\\?\`) {
		resolved = strings.TrimPrefix(resolved, `\\?\`)
	}
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("physical frame path is not absolute")
	}
	return resolved, nil
}
