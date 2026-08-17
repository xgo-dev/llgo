package compilerhash

import (
	"crypto/sha256"
	"encoding/binary"
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

	writeUint64LE            = func(w io.Writer, v uint64) error { return binary.Write(w, binary.LittleEndian, v) }
	executablePath           = os.Executable
	compilerHashFromPathFunc = compilerHashFromPath
	isDevel                  = env.Devel
)

// Value returns the compiler hash used for cache invalidation. Release builds
// leave it empty (unless preset via -X), while development builds hash the
// running executable's metadata.
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
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("llgo: stat executable: %w", err)
	}
	return hashMetadata(info.ModTime().UTC().UnixNano(), info.Size())
}

func hashMetadata(modTimeNano int64, size int64) (string, error) {
	if size < 0 {
		return "", fmt.Errorf("llgo: invalid executable size: %d", size)
	}
	h := sha256.New()
	if err := writeUint64LE(h, uint64(modTimeNano)); err != nil {
		return "", fmt.Errorf("llgo: hash metadata timestamp: %w", err)
	}
	if err := writeUint64LE(h, uint64(size)); err != nil {
		return "", fmt.Errorf("llgo: hash metadata size: %w", err)
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
	writeUint64LE = func(w io.Writer, v uint64) error { return binary.Write(w, binary.LittleEndian, v) }
	isDevel = env.Devel
}
