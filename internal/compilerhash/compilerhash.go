package compilerhash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/xgo-dev/llgo/internal/env"
)

var (
	// LinkedValue may be set at link time for release builds using -X.
	LinkedValue string

	once    sync.Once
	cached  string
	onceErr error

	executablePath           = os.Executable
	compilerHashFromPathFunc = compilerHashFromPath
	isDevel                  = env.Devel
)

// Value returns the compiler hash used for cache invalidation. Release builds
// leave it empty (unless preset via -X), while development builds hash the
// running executable's contents once per process. Copies of an unchanged
// executable reuse archives even when their filesystem timestamps differ.
func Value() string {
	ensure()
	return cached
}

func ensure() {
	once.Do(func() {
		switch {
		case LinkedValue != "":
			cached = LinkedValue
			return
		case !isDevel():
			cached = ""
			return
		}

		exe, err := executablePath()
		if err != nil {
			onceErr = fmt.Errorf("llgo: determine executable path: %w", err)
			return
		}
		hash, err := compilerHashFromPathFunc(exe)
		if err != nil {
			onceErr = fmt.Errorf("llgo: compute compiler hash: %w", err)
			return
		}
		cached = hash
	})
	if onceErr != nil {
		panic(onceErr)
	}
}

func compilerHashFromPath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("llgo: open executable: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("llgo: read executable: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func resetForTesting() {
	once = sync.Once{}
	cached = ""
	onceErr = nil
	LinkedValue = ""
	executablePath = os.Executable
	compilerHashFromPathFunc = compilerHashFromPath
	isDevel = env.Devel
}
