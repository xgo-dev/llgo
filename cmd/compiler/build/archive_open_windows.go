package build

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func openPCLNArchive(path string) (*os.File, error) {
	return openPCLNArchiveWithOpener(path, os.Open)
}

func openPCLNArchiveWithOpener(path string, open func(string) (*os.File, error)) (*os.File, error) {
	// Another compiler can briefly hold a conflicting handle while publishing
	// the same cache entry. Retry only ERROR_SHARING_VIOLATION, before reading
	// any bytes; malformed archives and permanent filesystem errors still fail.
	const sharingViolation = syscall.Errno(32)
	deadline := time.Now().Add(2 * time.Second)
	for {
		f, err := open(path)
		if !errors.Is(err, sharingViolation) || !time.Now().Before(deadline) {
			return f, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
