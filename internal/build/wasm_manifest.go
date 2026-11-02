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

package build

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
	gllvm "github.com/xgo-dev/llvm"
)

type wasmManifestTool struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type wasmManifestArtifact struct {
	Path      string `json:"path"`
	Role      string `json:"role"`
	Bytes     int64  `json:"bytes"`
	GzipBytes int64  `json:"gzip_bytes"`
	SHA256    string `json:"sha256"`
}

type wasmManifest struct {
	Schema          int                    `json:"schema"`
	BuildID         string                 `json:"build_id"`
	Compiler        string                 `json:"compiler"`
	CompilerHash    string                 `json:"compiler_build_hash,omitempty"`
	CompilerGo      string                 `json:"compiler_go"`
	Go              string                 `json:"go"`
	LLVM            string                 `json:"llvm"`
	LinkDriver      string                 `json:"link_driver"`
	DriverVersion   string                 `json:"link_driver_version"`
	Target          string                 `json:"target"`
	Profile         string                 `json:"profile"`
	Provider        string                 `json:"provider"`
	PointerBits     int                    `json:"pointer_bits"`
	PCLN            string                 `json:"pclntab"`
	Optimization    WasmOptMode            `json:"optimization"`
	Strip           WasmStripMode          `json:"strip"`
	Features        []string               `json:"features"`
	AllowedFeatures []string               `json:"allowed_features"`
	Tool            wasmManifestTool       `json:"wasm_opt"`
	Args            []string               `json:"wasm_opt_args"`
	LinkFlags       []string               `json:"link_flags"`
	InputBytes      int64                  `json:"post_link_bytes"`
	Payload         wasmManifestSize       `json:"payload"`
	Debug           wasmManifestSize       `json:"debug"`
	Artifacts       []wasmManifestArtifact `json:"artifacts"`
}

// Artifact totals exclude the manifest itself, whose on-disk size is reported
// by CollectArtifacts. Gzip sizes sum independently compressed files at level 9.
type wasmManifestSize struct {
	Bytes     int64 `json:"bytes"`
	GzipBytes int64 `json:"gzip_bytes"`
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type byteCounter int64

func (n *byteCounter) Write(p []byte) (int, error) { *n += byteCounter(len(p)); return len(p), nil }

func wasmArtifact(path string) (wasmManifestArtifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return wasmManifestArtifact{}, err
	}
	defer f.Close()
	return wasmArtifactFrom(path, f)
}

func wasmArtifactFrom(path string, reader io.Reader) (wasmManifestArtifact, error) {
	a := wasmManifestArtifact{Path: filepath.Base(path), Role: "deployment"}
	if strings.HasSuffix(path, ".debug.wasm") || strings.HasSuffix(path, ".wasm.map") {
		a.Role = "debug"
	}
	if strings.HasSuffix(path, ".pclntab") {
		a.Role = "runtime-symbols"
	}
	var compressed byteCounter
	gz, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		return a, err
	}
	h := sha256.New()
	a.Bytes, err = io.Copy(io.MultiWriter(h, gz), reader)
	if err != nil {
		return a, err
	}
	if err := gz.Close(); err != nil {
		return a, err
	}
	a.GzipBytes, a.SHA256 = int64(compressed), hex.EncodeToString(h.Sum(nil))
	return a, nil
}

func writeWasmManifest(ctx *context, dir, module string, moduleData []byte, name string, tool wasmManifestTool, args, linkFlags []string, before int64) error {
	conf := ctx.buildConf
	if tool.Path != "" {
		tool.Path = filepath.Base(tool.Path)
	}
	linkFlags = slices.Clone(linkFlags)
	for i, arg := range linkFlags {
		linkFlags[i] = wasmManifestArg(arg)
	}
	m := wasmManifest{Schema: 1, Compiler: env.Version(), CompilerHash: conf.CompilerHash,
		CompilerGo: runtime.Version(), Go: ctx.sourceGoVersion(), LLVM: gllvm.Version,
		LinkDriver: ctx.crossCompile.CC, DriverVersion: ctx.getLLVMVersion(),
		Target: conf.Target, Profile: string(ctx.crossCompile.WasmProfile), Provider: string(ctx.crossCompile.WasmProvider),
		PointerBits: int(conf.debugPointerSize) * 8, PCLN: conf.PCLNMode.String(), Optimization: conf.WasmOpt,
		Strip: conf.WasmStrip, Tool: tool, Args: args, LinkFlags: linkFlags, InputBytes: before,
		AllowedFeatures: wasmFinalFeatures(&ctx.crossCompile)}
	if m.LinkDriver != "" {
		m.LinkDriver = filepath.Base(m.LinkDriver)
	}
	if m.Optimization == "" {
		m.Optimization = WasmOptOff
	}
	if m.Strip == "" {
		m.Strip = WasmStripNone
	}
	var err error
	if m.Features, err = wasmdebug.TargetFeatures(moduleData); err != nil {
		return err
	}
	if m.Features == nil {
		m.Features = []string{}
	}
	slices.Sort(m.Features)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	identity := sha256.New()
	for _, entry := range entries {
		if entry.Name() == name {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unexpected non-file Wasm artifact %q", entry.Name())
		}
		var a wasmManifestArtifact
		if entry.Name() == module {
			// Optimization/debug packaging can change features and bytes. Reuse
			// that final image for metadata, hashing and compression; other
			// artifacts are streamed without loading the whole set into memory.
			a, err = wasmArtifactFrom(module, bytes.NewReader(moduleData))
			a.Role = "deployment"
		} else {
			a, err = wasmArtifact(filepath.Join(dir, entry.Name()))
		}
		if err != nil {
			return err
		}
		m.Artifacts = append(m.Artifacts, a)
		total := &m.Payload
		if a.Role == "debug" {
			total = &m.Debug
		}
		total.Bytes += a.Bytes
		total.GzipBytes += a.GzipBytes
		fmt.Fprintf(identity, "%s\x00%s\n", a.Path, a.SHA256)
	}
	m.BuildID = hex.EncodeToString(identity.Sum(nil))
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0o644)
}

// Keep flag intent without publishing machine-specific absolute directories.
// A manifest describes the result; it is not a portable linker command line.
func wasmManifestArg(arg string) string {
	if filepath.IsAbs(arg) {
		return "<path>/" + filepath.Base(arg)
	}
	if i := strings.IndexAny(arg, "=,"); i >= 0 {
		return wasmManifestArg(arg[:i]) + arg[i:i+1] + wasmManifestArg(arg[i+1:])
	}
	for _, prefix := range []string{"-L", "-I", "@"} {
		if strings.HasPrefix(arg, prefix) && filepath.IsAbs(arg[len(prefix):]) {
			return prefix + wasmManifestArg(arg[len(prefix):])
		}
	}
	return arg
}

// Publish with rollback on ordinary filesystem failures, then publish the
// manifest as the completion marker. This is not a cross-file atomic rename;
// deployers should verify the manifest or publish into a versioned directory.
func publishWasmArtifacts(stage, destination, manifest string, remove []string, rename func(string, string) error) error {
	entries, err := os.ReadDir(stage)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if entry.Name() != manifest {
			names = append(names, entry.Name())
		}
	}
	names = append(names, remove...)
	names = append(names, manifest)
	backup, err := os.MkdirTemp(destination, ".llgo-wasm-backup-*")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			os.RemoveAll(backup)
		}
	}()
	var saved, installed []string
	rollback := func(cause error) error {
		for i := len(installed) - 1; i >= 0; i-- {
			if err := os.Remove(filepath.Join(destination, installed[i])); err != nil {
				cause = errors.Join(cause, err)
			}
		}
		for i := len(saved) - 1; i >= 0; i-- {
			if err := rename(filepath.Join(backup, saved[i]), filepath.Join(destination, saved[i])); err != nil {
				keepBackup = true
				cause = errors.Join(cause, fmt.Errorf("restore %s (backup retained at %s): %w", saved[i], backup, err))
			}
		}
		return cause
	}
	for _, name := range names {
		dst := filepath.Join(destination, name)
		if info, err := os.Lstat(dst); err == nil {
			if !info.Mode().IsRegular() {
				return rollback(fmt.Errorf("refusing to replace non-file artifact %s", dst))
			}
			if err := rename(dst, filepath.Join(backup, name)); err != nil {
				return rollback(err)
			}
			saved = append(saved, name)
		} else if !os.IsNotExist(err) {
			return rollback(err)
		}
		if slices.Contains(remove, name) {
			continue
		}
		if err := rename(filepath.Join(stage, name), dst); err != nil {
			return rollback(err)
		}
		installed = append(installed, name)
	}
	return nil
}
