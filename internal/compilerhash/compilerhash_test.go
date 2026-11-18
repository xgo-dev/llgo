package compilerhash

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Copies of the same compiler must reuse archives even when the filesystem
// timestamps differ. Conversely, same-size edits must invalidate them even if
// a packaging tool preserves the old timestamp.
func TestCompilerHashFromPath(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	second := filepath.Join(dir, "second")
	data := []byte("compiler one")
	timestamp := time.Unix(1735584000, 0)
	for i, path := range []string{first, second} {
		if err := os.WriteFile(path, data, 0755); err != nil {
			t.Fatal(err)
		}
		stamp := timestamp.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	hash := func(path string) string {
		t.Helper()
		got, err := compilerHashFromPath(path)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	before := hash(first)
	if got := hash(second); got != before {
		t.Errorf("identical compiler contents have different identities: %s != %s", got, before)
	}
	if err := os.WriteFile(first, []byte("compiler two"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(first, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
	if got := hash(first); got == before {
		t.Error("changed contents retained identity with unchanged size and timestamp")
	}
	for _, path := range []string{filepath.Join(dir, "missing"), dir} {
		if _, err := compilerHashFromPath(path); err == nil {
			t.Errorf("unreadable compiler %q accepted", path)
		}
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
