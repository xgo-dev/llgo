//go:build !llgo

package targets

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/mockable"
	targetcfg "github.com/xgo-dev/llgo/internal/targets"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestTargets(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("base", `{"goos":"linux","goarch":"arm","build-tags":["baremetal"],"linker":"ld.lld"}`)
	write("board", `{"inherits":["base"],"cpu":"cortex-m0plus","build-tags":["board"]}`)
	write("broken", `{`)
	resolver := targetcfg.NewResolver(dir)

	var output bytes.Buffer
	if err := run(nil, &output, &output, resolver); err != nil {
		t.Fatal(err)
	}
	if got := strings.ReplaceAll(output.String(), "\r\n", "\n"); got != "base\nboard\nbroken\n" {
		t.Fatalf("targets = %q", got)
	}
	output.Reset()
	if err := run([]string{"-json"}, &output, &output, resolver); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("bulk JSON with malformed target error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("bulk JSON emitted a partial document: %q", &output)
	}

	output.Reset()
	if err := run([]string{"-json", "board"}, &output, &output, resolver); err != nil {
		t.Fatal(err)
	}
	var got []targetInfo
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "board" || got[0].GOOS != "linux" || got[0].GOARCH != "arm" || got[0].CPU != "cortex-m0plus" || strings.Join(got[0].BuildTags, ",") != "baremetal,board" {
		t.Fatalf("resolved target = %#v", got)
	}

	if err := run([]string{"missing"}, &output, &output, resolver); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing target error = %v", err)
	}
	if err := run([]string{"broken"}, &output, &output, resolver); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("broken explicit target error = %v", err)
	}
	if err := run([]string{"-bad"}, &output, &output, resolver); err == nil {
		t.Fatal("unknown flag succeeded")
	}
	if err := run([]string{"-h"}, &output, &output, resolver); err != nil {
		t.Fatalf("help: %v", err)
	}
}

func TestTargetsDoesNotCreateCache(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime", "go.mod"), []byte("module github.com/xgo-dev/llgo/runtime\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "targets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "targets", "board.json"), []byte(`{"goos":"linux","goarch":"arm"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(t.TempDir(), "cache")
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
	var output bytes.Buffer
	if err := run([]string{"-json"}, &output, &output, targetcfg.NewDefaultResolver()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("targets query created cache: %v", err)
	}
}

func TestTargetsErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "board.json"), []byte(`{"goos":"linux","goarch":"arm"}`), 0644); err != nil {
		t.Fatal(err)
	}
	resolver := targetcfg.NewResolver(dir)
	for _, args := range [][]string{{"board"}, {"-json", "board"}} {
		if err := run(args, failingWriter{}, failingWriter{}, resolver); err == nil || !strings.Contains(err.Error(), "write failed") {
			t.Errorf("targets %q write error = %v", args, err)
		}
	}
	if err := run(nil, failingWriter{}, failingWriter{}, targetcfg.NewResolver(filepath.Join(dir, "missing"))); err == nil || !strings.Contains(err.Error(), "read targets directory") {
		t.Fatalf("missing directory error = %v", err)
	}
}

func TestRunCmdFailure(t *testing.T) {
	t.Setenv("LLGO_ROOT", t.TempDir())
	mockable.EnableMock()
	defer mockable.DisableMock()
	defer func() {
		if got := recover(); got != "exit" || mockable.ExitCode() != 1 {
			t.Errorf("exit = %v, %d", got, mockable.ExitCode())
		}
	}()
	Cmd.Run(Cmd, []string{"missing"})
}
