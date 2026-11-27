package api

import (
	"bytes"
	"fmt"
	"os"
	"runtime/coverage"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("LLGO_COVER_EARLY_EXIT") != "" {
		Hit(true)
		os.Exit(0)
	}
	code := m.Run()
	if os.Getenv("LLGO_COVER_TEST_APIS") != "" {
		if err := checkCoverageWriters(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
	}
	os.Exit(code)
}

func TestCoverageAPI(t *testing.T) {
	if testing.CoverMode() == "" {
		t.Skip("requires coverage instrumentation")
	}
	Hit(true)
	before := testing.Coverage()
	Hit(false)
	if before == 0 || testing.Coverage() <= before {
		t.Fatal("snapshot did not observe newly executed block")
	}
	// Go test initializes the writer's final metadata hash during M.Run's
	// teardown, not before the tests. Check successful writers in TestMain.
	if err := coverage.WriteMeta(&bytes.Buffer{}); err == nil {
		t.Fatal("WriteMeta unexpectedly succeeded before coverage teardown")
	}
}

func checkCoverageWriters() error {
	var meta, counters bytes.Buffer
	if err := coverage.WriteMeta(&meta); err != nil || meta.Len() == 0 {
		return fmt.Errorf("WriteMeta: %v, %d bytes", err, meta.Len())
	}
	dir, err := os.MkdirTemp("", "coverage-api-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := coverage.WriteMetaDir(dir); err != nil {
		return err
	}
	if testing.CoverMode() != "atomic" {
		if coverage.WriteCounters(&counters) == nil || coverage.ClearCounters() == nil {
			return fmt.Errorf("non-atomic mode accepted atomic-only APIs")
		}
		return nil
	}
	if err := coverage.WriteCounters(&counters); err != nil || counters.Len() == 0 {
		return fmt.Errorf("WriteCounters: %v, %d bytes", err, counters.Len())
	}
	if err := coverage.WriteCountersDir(dir); err != nil {
		return err
	}
	if err := coverage.ClearCounters(); err != nil {
		return err
	}
	if testing.Coverage() != 0 {
		return fmt.Errorf("ClearCounters did not clear package counters")
	}
	Hit(true)
	return nil
}

func TestFailureProfile(t *testing.T) {
	if os.Getenv("LLGO_COVER_TEST_FAIL") == "" {
		t.Skip("failure-path fixture")
	}
	Hit(false)
	t.Error("intentional coverage failure")
}
