//go:build go1.25

package slog_test

import (
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRecordSource(t *testing.T) {
	pcs := make([]uintptr, 1)
	runtime.Callers(1, pcs)
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "source", pcs[0])
	source := record.Source()
	if source == nil || !strings.Contains(source.Function, "TestRecordSource") || source.Line == 0 {
		t.Fatalf("Record.Source = %#v", source)
	}
}
