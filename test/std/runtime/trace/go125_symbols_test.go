//go:build go1.25

package trace_test

import (
	"bytes"
	"context"
	"runtime/trace"
	"testing"
	"time"
)

func TestFlightRecorder(t *testing.T) {
	if trace.IsEnabled() {
		t.Skip("another execution trace is already enabled")
	}
	recorder := trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:   time.Millisecond,
		MaxBytes: 1 << 20,
	})
	if recorder.Enabled() {
		t.Fatal("new flight recorder is enabled before Start")
	}
	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	if !recorder.Enabled() {
		t.Fatal("flight recorder is disabled after Start")
	}
	trace.WithRegion(context.Background(), "go1.25", func() {
		trace.Log(context.Background(), "stdlib", "flight recorder")
	})
	var output bytes.Buffer
	n, err := recorder.WriteTo(&output)
	if err != nil {
		recorder.Stop()
		t.Fatal(err)
	}
	if n != int64(output.Len()) || n < 16 {
		recorder.Stop()
		t.Fatalf("WriteTo wrote %d bytes, buffer length %d", n, output.Len())
	}
	recorder.Stop()
	if recorder.Enabled() {
		t.Fatal("flight recorder is enabled after Stop")
	}
}
