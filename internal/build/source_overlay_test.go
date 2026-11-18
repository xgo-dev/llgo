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

package build

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSourceOverlayWriteFailuresAreAtomic(t *testing.T) {
	closeFailure := errors.New("overlay close failed")
	for _, failWrite := range []bool{true, false} {
		t.Run(map[bool]string{true: "write", false: "close"}[failWrite], func(t *testing.T) {
			dir := t.TempDir()
			conf := &Config{
				Overlay: map[string][]byte{"original.go": []byte("package original")},
				SourceOverlay: map[string]SourceOverlayFile{
					"selected.go": {BackingFile: "replacement.go", Contents: []byte("package replacement")},
				},
			}
			before := conf.clone()
			var created string
			createTemp := func(_, pattern string) (*os.File, error) {
				file, err := os.CreateTemp(dir, pattern)
				if err == nil {
					created = file.Name()
					if failWrite {
						err = file.Close()
					}
				}
				return file, err
			}
			closed := false
			closeFile := func(file *os.File) error {
				closed = true
				file.Close()
				return closeFailure
			}
			name, err := prepareSourceOverlayWithIO(conf, createTemp, closeFile)
			wantErr := closeFailure
			if failWrite {
				wantErr = os.ErrClosed
			}
			if name != "" || !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "write source overlay:") {
				t.Fatalf("overlay failure = (%q, %v), want %v", name, err, wantErr)
			}
			if !closed {
				t.Fatal("did not close the temporary overlay after writing")
			}
			if _, err := os.Stat(created); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed overlay file survived: %v", err)
			}
			if !reflect.DeepEqual(conf, before) {
				t.Fatal("changed the configuration after overlay failure")
			}
		})
	}
}

func TestSourceOverlayViewsAndClone(t *testing.T) {
	dir := t.TempDir()
	selected := filepath.Join(dir, "virtual.go")
	backing := filepath.Join(dir, "backing.go")
	deleted := filepath.Join(dir, "deleted.go")
	conf := &Config{Overlay: map[string][]byte{"base.go": []byte("package base")}, SourceOverlay: map[string]SourceOverlayFile{
		selected: {BackingFile: backing, Contents: []byte("package overlay")}, deleted: {},
	}}
	copy := conf.clone()
	record := copy.SourceOverlay[selected]
	record.Contents[0] = 'P'
	copy.SourceOverlay[selected] = record
	delete(copy.SourceOverlay, deleted)
	if string(conf.SourceOverlay[selected].Contents) != "package overlay" || len(conf.SourceOverlay) != 2 {
		t.Fatal("source overlay clone aliases caller")
	}
	file, err := prepareSourceOverlay(conf)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var overlay struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &overlay); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(overlay.Replace, map[string]string{selected: backing, deleted: ""}) {
		t.Fatalf("loader overlay=%v", overlay.Replace)
	}
	if string(conf.Overlay[selected]) != "package overlay" || string(conf.Overlay["base.go"]) != "package base" {
		t.Fatalf("parser overlay=%v", conf.Overlay)
	}
	if _, ok := conf.Overlay[deleted]; !ok {
		t.Fatal("parser lost deletion view")
	}
	fresh := &Config{SourceOverlay: conf.SourceOverlay}
	name, err := prepareSourceOverlay(fresh)
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(name)
	if len(fresh.Overlay) != 2 {
		t.Fatal("failed to create parser view")
	}
	if name, err := prepareSourceOverlay(&Config{}); name != "" || err != nil {
		t.Fatalf("empty overlay: %q %v", name, err)
	}
	// An unavailable temporary directory must fail without mutating the parser view.
	for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(variable, filepath.Join(dir, "missing"))
	}
	if reused, err := prepareSourceOverlay(conf.clone()); err != nil || reused != file {
		t.Fatalf("child did not reuse the live overlay: %q %v", reused, err)
	}
	unprepared := &Config{Overlay: conf.Overlay, SourceOverlay: conf.SourceOverlay}
	before := unprepared.Overlay
	if name, err := prepareSourceOverlay(unprepared); err == nil || name != "" {
		t.Fatalf("accepted unavailable temp directory: %q %v", name, err)
	}
	if !reflect.DeepEqual(unprepared.Overlay, before) {
		t.Fatal("changed parser view on error")
	}
	buildConf := NewDefaultConf(ModeBuild)
	buildConf.SourceOverlay = conf.SourceOverlay
	if _, err := Do([]string{"../../benchmark/binary_size/println"}, buildConf); err == nil || !strings.Contains(err.Error(), "create source overlay:") {
		t.Fatalf("Build did not report overlay preparation failure: %v", err)
	}
}

func TestSourceOverlayEmptyRelativePathBuild(t *testing.T) {
	dir := t.TempDir()
	for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(variable, dir)
	}
	file, err := os.CreateTemp(".", "llgo-empty-overlay-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(file.Name()) })
	if _, err := file.WriteString(`{"Replace":{}}`); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "program"+defaultAppExt(conf))
	conf.GoBuildFlags = []string{"-overlay=" + file.Name()}
	conf.SourceOverlay = map[string]SourceOverlayFile{}
	if _, err := Do([]string{"../../benchmark/binary_size/println"}, conf); err != nil {
		t.Fatal(err)
	}
	assertBuiltProgram(t, conf.OutFile, "Hello, world")
	if leftovers, err := filepath.Glob(filepath.Join(dir, "llgo-overlay-*.json")); err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary overlays survived Build: %v %v", leftovers, err)
	}
}

func TestSourceOverlaySurvivesInitialGroupsAndIsRemoved(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "success", false: "error"}[valid], func(t *testing.T) {
			dir := t.TempDir()
			for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(variable, dir)
			}
			logical, err := filepath.Abs("../../benchmark/binary_size/println/main.go")
			if err != nil {
				t.Fatal(err)
			}
			source := "package main\nfunc main(){println(42)}\n"
			if !valid {
				source = "package main\nfunc main(){undefined()}\n"
			}
			backing := filepath.Join(dir, "replacement.go")
			if err := os.WriteFile(backing, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			conf := NewDefaultConf(ModeBuild)
			conf.OutFile = t.TempDir() + string(os.PathSeparator)
			conf.AppExt = defaultAppExt(conf)
			conf.SourceOverlay = map[string]SourceOverlayFile{logical: {BackingFile: backing, Contents: []byte(source)}}
			_, err = Do([]string{"../../benchmark/binary_size/cprintf", "../../benchmark/binary_size/println", "../../benchmark/binary_size/fmtprintf"}, conf)
			if valid {
				if err != nil {
					t.Fatal(err)
				}
				assertBuiltProgram(t, filepath.Join(conf.OutFile, "println"+conf.AppExt), "42")
				for _, name := range []string{"cprintf", "fmtprintf"} {
					assertBuiltProgram(t, filepath.Join(conf.OutFile, name+conf.AppExt), "Hello, world")
				}
			} else if err == nil {
				t.Fatal("invalid overlay source built successfully")
			}
			leftovers, err := filepath.Glob(filepath.Join(dir, "llgo-overlay-*.json"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("overlay files survived Build: %v %v", leftovers, err)
			}
		})
	}
}

func TestCleanReportsRemovalErrors(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "program")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "keep"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeExecutableArtifacts(blocked, true); err == nil {
		t.Fatal("silently ignored a removal error")
	}
	if err := removeFile(filepath.Join(dir, "absent"), false); err != nil {
		t.Fatal(err)
	}
}
