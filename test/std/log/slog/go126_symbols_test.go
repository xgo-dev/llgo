//go:build go1.26

package slog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestMultiHandler(t *testing.T) {
	var first, second bytes.Buffer
	multi := slog.NewMultiHandler(
		slog.NewTextHandler(&first, nil),
		slog.NewJSONHandler(&second, nil),
	)
	ctx := context.Background()
	if !multi.Enabled(ctx, slog.LevelInfo) {
		t.Fatal("MultiHandler unexpectedly disabled info records")
	}

	direct := slog.NewRecord(time.Unix(1, 0), slog.LevelInfo, "direct", 0)
	if err := multi.Handle(ctx, direct); err != nil {
		t.Fatal(err)
	}
	withAttrs := multi.WithAttrs([]slog.Attr{slog.String("component", "compiler")})
	if err := withAttrs.Handle(ctx, slog.NewRecord(time.Unix(1, 0), slog.LevelInfo, "compiled", 0)); err != nil {
		t.Fatal(err)
	}
	withGroup := multi.WithGroup("details")
	record := slog.NewRecord(time.Unix(1, 0), slog.LevelInfo, "grouped", 0)
	record.AddAttrs(slog.GroupAttrs("build", slog.Int("files", 2)))
	if err := withGroup.Handle(ctx, record); err != nil {
		t.Fatal(err)
	}
	textOutput := first.String()
	for _, want := range []string{"msg=direct", "msg=compiled component=compiler", "msg=grouped details.build.files=2"} {
		if !strings.Contains(textOutput, want) {
			t.Fatalf("text handler output %q does not contain %q", textOutput, want)
		}
	}
	jsonLines := bytes.Split(bytes.TrimSpace(second.Bytes()), []byte("\n"))
	if len(jsonLines) != 3 {
		t.Fatalf("JSON handler wrote %d records, want 3: %q", len(jsonLines), second.String())
	}
	records := make([]map[string]any, len(jsonLines))
	for i, line := range jsonLines {
		if err := json.Unmarshal(line, &records[i]); err != nil {
			t.Fatalf("JSON record %d is invalid: %v: %q", i, err, line)
		}
	}
	if records[0]["msg"] != "direct" {
		t.Fatalf("first JSON record = %#v", records[0])
	}
	if records[1]["msg"] != "compiled" || records[1]["component"] != "compiler" {
		t.Fatalf("second JSON record = %#v", records[1])
	}
	details, ok := records[2]["details"].(map[string]any)
	if !ok {
		t.Fatalf("third JSON record has no details group: %#v", records[2])
	}
	build, ok := details["build"].(map[string]any)
	if !ok || build["files"] != float64(2) {
		t.Fatalf("third JSON record has wrong build group: %#v", records[2])
	}
}
