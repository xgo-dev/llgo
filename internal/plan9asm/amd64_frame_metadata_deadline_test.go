package plan9asm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAMD64DownloadedFrameMetadataDeadline(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	environment := frameEnvironment(os.Environ(), map[string]string{
		"LLGO_FRAME_METADATA_TEST_CHILD": "deadline",
		"LLGO_FRAME_METADATA_TEST_TRACE": filepath.Join(t.TempDir(), "commands.jsonl"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = prepareDownloadedFrameMetadata(ctx, binary, temporary, environment, "example.com/sdkframe", "v1.0.0")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 5*time.Second {
		t.Fatal("owned metadata command did not retain its bounded cancellation:", err)
	}
	entries, readErr := os.ReadDir(temporary)
	if readErr != nil || len(entries) != 0 {
		t.Fatal("cancelled metadata command retained its private module:", entries, readErr)
	}
}
