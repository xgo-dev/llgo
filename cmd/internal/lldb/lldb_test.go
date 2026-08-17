//go:build !llgo

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lldb

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/internal/mockable"
)

func TestParseLLDBVersion(t *testing.T) {
	tests := []struct {
		version string
		want    lldbVersion
		ok      bool
	}{
		{"lldb version 18.1.8", lldbVersion{major: 18}, true},
		{"lldb-1703.0.236.21\nApple Swift version 6.2", lldbVersion{major: 1703, apple: true}, true},
		{"LLDB version 21.0.0git", lldbVersion{major: 21}, true},
		{"clang version 21.0.0", lldbVersion{}, false},
	}
	for _, test := range tests {
		got, ok := parseLLDBVersion(test.version)
		if got != test.want || ok != test.ok {
			t.Errorf("parseLLDBVersion(%q) = (%+v, %v), want (%+v, %v)", test.version, got, ok, test.want, test.ok)
		}
	}
}

func TestLLDBImportCommandEscapesPath(t *testing.T) {
	got := lldbImportCommand(`a\b"c.py`)
	want := `command script import "a\\b\"c.py"`
	if got != want {
		t.Fatalf("lldbImportCommand() = %q, want %q", got, want)
	}
}

func TestValidateLLDB(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	newLLDB := writeFakeLLDB(t, "lldb version 18.1.8", "")
	if got, err := validateLLDB(newLLDB); err != nil || got != newLLDB {
		t.Fatalf("validateLLDB() = (%q, %v), want (%q, nil)", got, err, newLLDB)
	}

	appleLLDB := writeFakeLLDB(t, "lldb-1703.0.236.21\nApple Swift version 6.2", "")
	if got, err := validateLLDB(appleLLDB); err != nil || got != appleLLDB {
		t.Fatalf("validateLLDB(Apple) = (%q, %v), want (%q, nil)", got, err, appleLLDB)
	}

	oldLLDB := writeFakeLLDB(t, "lldb version 17.0.6", "")
	if _, err := validateLLDB(oldLLDB); err == nil || !strings.Contains(err.Error(), "version 18 or newer") {
		t.Fatalf("validateLLDB(old) error = %v", err)
	}

	unparseable := writeFakeLLDB(t, "clang version 18.1.8", "")
	if _, err := validateLLDB(unparseable); err == nil || !strings.Contains(err.Error(), "cannot parse") {
		t.Fatalf("validateLLDB(unparseable) error = %v", err)
	}

	if _, err := validateLLDB(filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "find") {
		t.Fatalf("validateLLDB(missing) error = %v", err)
	}
}

func TestFindLLDBPrecedenceAndFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	newLLDB := writeFakeLLDB(t, "lldb version 18.1.8", "")
	oldLLDB := writeFakeLLDB(t, "lldb version 17.0.6", "")

	if got, err := findLLDBFrom(newLLDB, oldLLDB, nil); err != nil || got != newLLDB {
		t.Fatalf("configured findLLDBFrom() = (%q, %v)", got, err)
	}
	if got, err := findLLDBFrom("", newLLDB, nil); err != nil || got != newLLDB {
		t.Fatalf("environment findLLDBFrom() = (%q, %v)", got, err)
	}
	if got, err := findLLDBFrom("", "", []string{oldLLDB, newLLDB, newLLDB}); err != nil || got != newLLDB {
		t.Fatalf("fallback findLLDBFrom() = (%q, %v)", got, err)
	}
	if _, err := findLLDBFrom("", "", []string{oldLLDB, filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("findLLDBFrom() succeeded without a supported LLDB")
	}
}

func TestRunImportsEmbeddedPluginAndPassesArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	capture := filepath.Join(t.TempDir(), "arguments")
	t.Setenv("LLGO_LLDB_TEST_CAPTURE", capture)
	fake := writeFakeLLDB(t, "lldb version 18.1.8", `
printf '%s\n' "$@" > "$LLGO_LLDB_TEST_CAPTURE"
plugin=$(printf '%s\n' "$2" | sed 's/^command script import "//; s/"$//')
test -s "$plugin"
grep -q __llgo_debugger_marker_v1 "$plugin"
`)

	var stdout, stderr bytes.Buffer
	if err := run(fake, []string{"--batch", "./program", "-o", "run"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("%v: %s", err, stderr.String())
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{"-O\n", "command script import \"", "--batch\n", "./program\n", "-o\n", "run\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("LLDB arguments %q do not contain %q", got, want)
		}
	}
}

func TestRunRequiresExecutable(t *testing.T) {
	err := run("lldb", nil, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || err.Error() != "llgo lldb: no executable specified" {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunCmdParseErrorExits(t *testing.T) {
	var cmd base.Command
	mockable.EnableMock()
	defer mockable.DisableMock()

	exited := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == "exit" {
					exited = true
					return
				}
				panic(recovered)
			}
		}()
		runCmd(&cmd, []string{"--batch", "./program"})
	}()
	if !exited || mockable.ExitCode() != 2 {
		t.Fatalf("runCmd parse error exit = (%v, %d), want (true, 2)", exited, mockable.ExitCode())
	}
}

func TestEmbeddedPluginIdentity(t *testing.T) {
	source := string(pluginSource)
	for _, want := range []string{
		"__lldb_init_module",
		"__llgo_debugger_marker_v1",
		"is_llgo_compiler",
		"inspect_target",
		"LLGO_DEBUGGER_SCHEMAS",
		"LLGO_RUNTIME_LAYOUTS",
		"string_summary",
		"slice_summary",
		"SliceSyntheticProvider",
		"llgo status",
		"llgo print",
		"llgo vars",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("embedded plugin is missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"llgo_plugin.print_go_expression p'",
		"llgo_plugin.print_all_variables v'",
	} {
		if strings.Contains(source, unwanted) {
			t.Errorf("embedded plugin overrides stock LLDB command in %q", unwanted)
		}
	}
}

func writeFakeLLDB(t *testing.T, version, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lldb")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  printf '%s\\n' '" + version + "'\n" +
		"  exit 0\n" +
		"fi\n" + body
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
