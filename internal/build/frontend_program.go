// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/goescape"
	"golang.org/x/tools/go/ssa"
)

type frontendProgramFeatures struct{ ReflectBridges, FuncInfoEntries bool }

// These decisions depend on the complete program, unlike generic function
// effects. Keep their identity and storage separate from package contracts.
type frontendProgramExport struct {
	Version             int
	Input               string
	Features            frontendProgramFeatures
	Initials            map[string]frontendProgramFeatures
	RuntimeFree, Closed bool
	Inits               map[string]cl.GlobalInitExport
}

func (f *frontendCache) programKey(mode ssa.BuilderMode) string {
	var inputs []string
	for _, pkg := range f.orderedPackages() {
		key, err := f.inputKey(pkg)
		if err != nil || key == "" {
			return ""
		}
		inputs = append(inputs, pkg.ID+":"+key)
	}
	for _, entry := range f.entries {
		if f.owners[entry.pkg] != nil {
			continue
		}
		for _, file := range entry.syntax {
			filename := f.ctx.progSSA.Fset.Position(file.Pos()).Filename
			data, ok := f.ctx.buildConf.Overlay[filename]
			if !ok {
				var err error
				data, err = os.ReadFile(filename)
				if err != nil {
					return ""
				}
			}
			inputs = append(inputs, entry.id+":"+filename+":"+frontendDigest(data))
		}
	}
	sort.Strings(inputs)
	roots := make([]string, 0, len(f.ctx.initial))
	for _, root := range f.ctx.initial {
		roots = append(roots, root.ID)
	}
	sort.Strings(roots)
	data, err := json.Marshal(struct {
		Version                               int
		Mode                                  ssa.BuilderMode
		Inputs, Roots                         []string
		Link, EnvironmentLink, CommandLink    string
		Asyncify, DebuggerFrames, ShadowStack bool
	}{frontendExportVersion, mode, inputs, roots, f.ctx.buildConf.LinkOptions.ExternalLinkerFlags, os.Getenv("LDFLAGS"), f.ctx.commands.lookup("LDFLAGS"), f.ctx.crossCompile.WasmPostLink.Asyncify, f.ctx.frontendOptions.DebuggerFrames, f.ctx.frontendOptions.ShadowStack})
	if err != nil {
		return ""
	}
	return frontendDigest(data)
}

func (f *frontendCache) programPath() string {
	return filepath.Join(f.ctx.ensureCacheManager().root, "frontend", "program-"+f.programInput+".json")
}

func (f *frontendCache) loadProgram() *frontendProgramExport {
	if f.programInput == "" {
		return nil
	}
	filename := f.programPath()
	info, err := os.Stat(filename)
	if err != nil || info.Size() > goescape.MaxCacheBytes {
		return nil
	}
	data, err := os.ReadFile(filename)
	var envelope frontendEnvelope
	if err != nil || json.Unmarshal(data, &envelope) != nil || frontendDigest(envelope.Payload) != envelope.Digest {
		return nil
	}
	var record frontendProgramExport
	if json.Unmarshal(envelope.Payload, &record) != nil || record.Version != frontendExportVersion || record.Input != f.programInput {
		return nil
	}
	for _, root := range f.ctx.initial {
		if _, ok := record.Initials[root.ID]; !ok {
			return nil
		}
	}
	for _, entry := range f.entries {
		if _, ok := record.Inits[entry.id]; !ok {
			return nil
		}
	}
	return &record
}

func (f *frontendCache) restoreInitializers() error {
	if f.program == nil {
		return fmt.Errorf("missing program export")
	}
	exports := make(map[*ssa.Package]cl.GlobalInitExport)
	for _, entry := range f.entries {
		exports[entry.pkg] = f.program.Inits[entry.id]
	}
	plan, err := cl.RestoreUnusedGlobalInits(exports)
	if err != nil {
		return err
	}
	f.ctx.frontendOptions.UnusedGlobalInits = plan
	f.ctx.wasmClosedProgram = f.program.Closed
	return nil
}

func (f *frontendCache) captureProgram() {
	if f.ctx.buildConf.Goarch != "wasm" || f.programInput == "" {
		return
	}
	record := &frontendProgramExport{Version: frontendExportVersion, Input: f.programInput,
		Features: frontendProgramFeatures{f.ctx.prog.Target().WasmReflectBridges, f.ctx.prog.Target().WasmFuncInfoEntries},
		Initials: make(map[string]frontendProgramFeatures), RuntimeFree: f.ctx.wasmRuntimeFree, Closed: f.ctx.wasmClosedProgram, Inits: make(map[string]cl.GlobalInitExport)}
	if f.program != nil {
		record.Initials = f.program.Initials
	} else {
		for _, root := range f.ctx.initial {
			use := analyzeWasmInitialUse(f.ctx.progSSA, root.Types)
			record.Initials[root.ID] = frontendProgramFeatures{wasmReflectBridgeProvider(f.ctx) && use.usesWasmReflectBridges(), use.usesRuntimeFuncForPC()}
		}
	}
	for _, entry := range f.entries {
		record.Inits[entry.id] = f.ctx.frontendOptions.UnusedGlobalInits.Export(entry.pkg)
	}
	// Preserve removals for opaque packages, whose initializer has no body here.
	if f.program != nil {
		for _, entry := range f.entries {
			if !f.built[entry.pkg] {
				record.Inits[entry.id] = f.program.Inits[entry.id]
			}
		}
	}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	wrapped, err := json.Marshal(frontendEnvelope{Digest: frontendDigest(data), Payload: data})
	if err == nil && len(wrapped) <= goescape.MaxCacheBytes {
		_ = writeManifest(f.programPath(), string(wrapped))
	}
}
