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
	"slices"
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
schema=$(dirname "$plugin")/llgo_debugger_schema_v1.json
test -s "$schema"
grep -q '"contract": "llgo.debugger"' "$schema"
grep -q __llgo_debugger_abi_v1 "$schema"
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
	if !strings.HasPrefix(got, "-O\ncommand script import \"") {
		t.Fatalf("LLDB arguments %q do not import the plugin after target creation", got)
	}
	for _, want := range []string{"--batch\n", "./program\n", "-o\n", "run\n"} {
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
		"LLGO_DEBUGGER_MARKER_PREFIX",
		"is_llgo_compiler",
		"inspect_target",
		"LLGO_DEBUGGER_SCHEMAS",
		"LLGO_RUNTIME_LAYOUTS",
		"LLGO_DEBUGGER_RECORD_SYMBOL",
		"llgo_debugger_schema_v1.json",
		"string_summary",
		"slice_summary",
		"SliceSyntheticProvider",
		"interface_summary",
		"function_summary",
		"map_summary",
		"MapSyntheticProvider",
		"channel_summary",
		"ChannelSyntheticProvider",
		"LLGO_GOROUTINE_LAYOUTS",
		"print_goroutines",
		"print_goroutine",
		"llgo status",
		"llgo print",
		"llgo vars",
		"llgo goroutines",
		"llgo goroutine",
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

func TestLLDBEnvironmentIsolatesWindowsPython(t *testing.T) {
	original := []string{
		`Path=C:\host-tools;C:\Windows`, `PythonHome=C:\python312`,
		`PYTHONPATH=C:\python312\site-packages`, "PYTHONUTF8=0",
		`LLGO_WINDOWS_LLDB_PYTHON_HOME=C:\python311`, "UNCHANGED=1",
	}
	snapshot := append([]string(nil), original...)
	got := lldbEnvironment(original, "windows", "amd64")
	for _, expected := range []string{
		`PATH=C:\python311;C:\host-tools;C:\Windows`,
		`PYTHONHOME=C:\python311`, "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8", "UNCHANGED=1",
	} {
		if !slices.Contains(got, expected) {
			t.Errorf("Windows debugger environment missing %q: %q", expected, got)
		}
	}
	for _, item := range got {
		if strings.HasPrefix(strings.ToUpper(item), "PYTHONPATH=") {
			t.Errorf("host Python module path leaked into LLDB: %q", item)
		}
	}
	if !slices.Equal(original, snapshot) {
		t.Fatalf("debugger setup mutated the parent environment: %q", original)
	}
	for _, goos := range []string{"darwin", "linux"} {
		if got := lldbEnvironment(original, goos, "arm64"); !slices.Equal(got, original) {
			t.Errorf("%s environment changed: %q", goos, got)
		}
	}
	explicit := append(append([]string(nil), original...), `LLGO_WINDOWS_LLDB_PYTHONPATH=C:\lldb\python`)
	if got := lldbEnvironment(explicit, "windows", "amd64"); !slices.Contains(got, `PYTHONPATH=C:\lldb\python`) {
		t.Errorf("explicit debugger module path missing: %q", got)
	}
	unconfigured := []string{`PATH=C:\tools`, `PYTHONHOME=C:\custom-debugger-python`}
	if got := lldbEnvironment(unconfigured, "windows", "amd64"); !slices.Equal(got, unconfigured) {
		t.Errorf("unconfigured debugger environment changed: %q", got)
	}
}

func TestLLDBEnvironmentWindowsARM64ProcessPlugin(t *testing.T) {
	for _, override := range []string{"", "LLDB_USE_LLDB_SERVER=1", "lldb_use_lldb_server=0"} {
		t.Run(override, func(t *testing.T) {
			original := make([]string, 1, 4)
			original[0] = `PATH=C:\tools`
			if override != "" {
				original = append(original, override)
			}
			snapshot := slices.Clone(original[:cap(original)])
			got := lldbEnvironment(original, "windows", "arm64")
			if override == "" {
				if !slices.Contains(got, "LLDB_USE_LLDB_SERVER=0") {
					t.Fatalf("native Windows process plugin not selected: %q", got)
				}
			} else if !slices.Equal(got, original) {
				t.Fatalf("explicit process plugin selection changed: %q", got)
			}
			if !slices.Equal(original[:cap(original)], snapshot) {
				t.Fatal("debugger setup mutated the parent environment backing array")
			}
		})
	}
}
