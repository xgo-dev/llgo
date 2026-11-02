/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compat_test

import (
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	buildcmd "github.com/xgo-dev/llgo/cmd/internal/build"
	"github.com/xgo-dev/llgo/cmd/internal/clean"
	"github.com/xgo-dev/llgo/cmd/internal/compat"
	"github.com/xgo-dev/llgo/cmd/internal/install"
	"github.com/xgo-dev/llgo/cmd/internal/monitor"
	"github.com/xgo-dev/llgo/cmd/internal/run"
	testcmd "github.com/xgo-dev/llgo/cmd/internal/test"
)

func TestManifestNamesOwnersAndRegistration(t *testing.T) {
	for _, name := range []string{"", " \t\n", "unknown"} {
		if record := compat.Command(name); record != (compat.Record{}) {
			t.Errorf("unknown command %q: %+v", name, record)
		}
	}
	surface := compat.GoSurface()
	pinned, err := os.ReadFile("../../../.go-version")
	if err != nil {
		t.Fatal(err)
	}
	if surface.GoVersion != strings.TrimSpace(string(pinned)) {
		t.Fatal("manifest version differs from .go-version; review the Go surface")
	}
	if surface.TestFlagPrefix != "test." {
		t.Fatal("missing test flag prefix contract")
	}
	if len(surface.Commands) < 19 {
		t.Fatal("incomplete Go command inventory")
	}
	for _, name := range strings.Fields(compat.TestBinaryFlags) {
		bare := testcmd.Cmd.Flag.Lookup(name)
		alias := testcmd.Cmd.Flag.Lookup(surface.TestFlagPrefix + name)
		if bare == nil || alias == nil {
			t.Errorf("test -%s must register both bare and test. spellings", name)
		}
	}
	for command, names := range surface.Commands {
		c := compat.Command(command)
		if c.Status == "" || c.Owner == "" || c.Note == "" {
			t.Errorf("missing command contract for %q", command)
		}
		if !slices.Contains([]string{"native", "delegated", "omitted", "rejected"}, c.Status) {
			t.Errorf("%s has no final command contract: %+v", command, c)
		}
		if !slices.IsSorted(names) {
			t.Errorf("%s flags are not sorted", command)
		}
		for _, name := range names {
			record := compat.Flag(command, name)
			if record.Status == "" || record.Owner == "" || record.Note == "" {
				t.Errorf("missing %s -%s contract", command, name)
			}
			if record.Status == "ignored" || record.Status == "loader-only" {
				t.Errorf("%s -%s has no truthful consumer", command, name)
			}
		}
	}
	for _, cmd := range []*base.Command{buildcmd.Cmd, install.Cmd, clean.Cmd, run.Cmd, run.CmpTestCmd, testcmd.Cmd, monitor.Cmd} {
		name := cmd.Name()
		goName := name
		if name == "cmptest" {
			goName = "run"
		}
		cmd.Flag.VisitAll(func(f *flag.Flag) {
			if strings.TrimSpace(f.Usage) == "" {
				t.Errorf("%s -%s has empty help", name, f.Name)
			}
			flagName := f.Name
			if name == "test" && strings.HasPrefix(flagName, surface.TestFlagPrefix) {
				flagName = strings.TrimPrefix(flagName, surface.TestFlagPrefix)
				if !slices.Contains(strings.Fields(compat.TestBinaryFlags), flagName) {
					t.Errorf("%s -%s is not a test-binary alias", name, f.Name)
				}
			}
			if !slices.Contains(surface.Commands[goName], flagName) && compat.Extensions[flagName] == "" {
				t.Errorf("%s -%s has no manifest entry", name, f.Name)
			}
		})
		if name == "monitor" {
			continue
		}
		for _, f := range surface.Commands[goName] {
			record := compat.Flag(name, f)
			if record.Status != "rejected" && f != "args" && cmd.Flag.Lookup(f) == nil {
				t.Errorf("%s -%s is advertised but not registered", name, f)
			}
			if record.Status == "rejected" {
				if registered := cmd.Flag.Lookup(f); registered != nil && !strings.HasPrefix(registered.Usage, "Unsupported:") {
					t.Errorf("%s -%s parses a rejected Go control", name, f)
				}
			}
		}
	}
}
