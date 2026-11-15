//go:build !windows

package build

import "os"

func openPCLNArchive(path string) (*os.File, error) {
	return os.Open(path)
}
