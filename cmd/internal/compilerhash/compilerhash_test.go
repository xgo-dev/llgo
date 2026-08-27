package compilerhash

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHashMetadataDeterminism(t *testing.T) {
	resetForTesting()

	const (
		modTime = int64(1700000000000000000)
		size    = int64(4096)
	)

	first, err := hashMetadata(modTime, size)
	if err != nil {
		t.Fatalf("hashMetadata returned error: %v", err)
	}
	second, err := hashMetadata(modTime, size)
	if err != nil {
		t.Fatalf("hashMetadata repeated call error: %v", err)
	}
	if first != second {
		t.Fatalf("hashMetadata is not deterministic: %q vs %q", first, second)
	}

	zeroSize, err := hashMetadata(modTime, 0)
	if err != nil {
		t.Fatalf("hashMetadata zero size error: %v", err)
	}
	if zeroSize == "" {
		t.Fatal("hashMetadata zero size returned empty hash")
	}

	modTimeChanged, err := hashMetadata(modTime+1, size)
	if err != nil {
		t.Fatalf("hashMetadata modTimeChanged error: %v", err)
	}
	if modTimeChanged == first {
		t.Fatalf("hashMetadata should change when mod time changes")
	}

	sizeChanged, err := hashMetadata(modTime, size+1)
	if err != nil {
		t.Fatalf("hashMetadata sizeChanged error: %v", err)
	}
	if sizeChanged == first {
		t.Fatalf("hashMetadata should change when size changes")
	}
}

func TestHashMetadataNegativeSize(t *testing.T) {
	resetForTesting()
	if _, err := hashMetadata(0, -1); err == nil {
		t.Fatal("hashMetadata should error for negative size")
	}
}

func TestHashMetadataTimestampWriteError(t *testing.T) {
	resetForTesting()
	wantErr := errors.New("write timestamp")
	writeUint64LE = func(ioWriter io.Writer, v uint64) error {
		return wantErr
	}

	if _, err := hashMetadata(1, 1); !errors.Is(err, wantErr) {
		t.Fatalf("hashMetadata error = %v, want %v", err, wantErr)
	}
}

func TestHashMetadataSizeWriteError(t *testing.T) {
	resetForTesting()
	wantErr := errors.New("write size")
	call := 0
	writeUint64LE = func(ioWriter io.Writer, v uint64) error {
		call++
		if call == 2 {
			return wantErr
		}
		return nil
	}

	if _, err := hashMetadata(1, 1); !errors.Is(err, wantErr) {
		t.Fatalf("hashMetadata error = %v, want %v", err, wantErr)
	}
}

func TestCompilerHashFromPath(t *testing.T) {
	resetForTesting()

	dir := t.TempDir()
	path := filepath.Join(dir, "llgo-bin")
	data := []byte("fake binary data")
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	modTime := time.Unix(1735584000, 123_000_000)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	got, err := compilerHashFromPath(path)
	if err != nil {
		t.Fatalf("compilerHashFromPath: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	want, err := hashMetadata(info.ModTime().UTC().UnixNano(), info.Size())
	if err != nil {
		t.Fatalf("hashMetadata: %v", err)
	}
	if got != want {
		t.Fatalf("compilerHashFromPath = %q, want %q", got, want)
	}

	missingPath := filepath.Join(dir, "missing")
	if _, err := compilerHashFromPath(missingPath); err == nil {
		t.Fatal("compilerHashFromPath should error for missing file")
	}
}

func TestValueUsesLinkedValue(t *testing.T) {
	resetForTesting()
	LinkedValue = "linked"
	if got := Value(); got != "linked" {
		t.Fatalf("Value() = %q, want %q", got, "linked")
	}
}

func TestValueSkipsHashForRelease(t *testing.T) {
	resetForTesting()
	isDevel = func() bool { return false }
	executablePath = func() (string, error) {
		t.Fatal("executablePath should not be called for release builds")
		return "", nil
	}
	if got := Value(); got != "" {
		t.Fatalf("Value() = %q, want empty", got)
	}
}

func TestValueExecutableErrorPanics(t *testing.T) {
	resetForTesting()
	isDevel = func() bool { return true }
	executablePath = func() (string, error) { return "", errors.New("boom") }

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Value should panic when executable lookup fails")
		}
	}()
	_ = Value()
}

func TestValueHashErrorPanics(t *testing.T) {
	resetForTesting()
	isDevel = func() bool { return true }
	executablePath = func() (string, error) { return "/fake", nil }
	compilerHashFromPathFunc = func(string) (string, error) { return "", errors.New("hash fail") }

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Value should panic when hashing fails")
		}
	}()
	_ = Value()
}

func TestValueComputesHashOnce(t *testing.T) {
	resetForTesting()
	isDevel = func() bool { return true }
	executablePath = func() (string, error) { return "/fake", nil }
	calls := 0
	compilerHashFromPathFunc = func(string) (string, error) {
		calls++
		return "computed", nil
	}

	if got := Value(); got != "computed" {
		t.Fatalf("Value() = %q, want %q", got, "computed")
	}
	if got := Value(); got != "computed" {
		t.Fatalf("second Value() = %q, want %q", got, "computed")
	}
	if calls != 1 {
		t.Fatalf("compilerHashFromPathFunc called %d times, want 1", calls)
	}
}
