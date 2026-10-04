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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactRole distinguishes debugger-owned metadata from bytes delivered to
// a target and from optional runtime symbolization data.
type ArtifactRole string

const (
	ArtifactRoleDebug           ArtifactRole = "debug"
	ArtifactRoleDeployment      ArtifactRole = "deployment"
	ArtifactRoleDebugDeployment ArtifactRole = "debug+deployment"
	ArtifactRoleRuntimeSymbols  ArtifactRole = "runtime-symbols"
)

// Artifact describes one final build output. Size is the on-disk byte size;
// deployment formats therefore remain distinct from their host debug image.
type Artifact struct {
	Role   ArtifactRole
	Format string
	Path   string
	Size   int64
}

// CollectArtifacts returns the final artifacts represented by out. It should
// be called after post-link packaging and target format conversion complete.
func CollectArtifacts(conf *Config, out *OutFmtDetails) ([]Artifact, error) {
	if conf == nil || out == nil {
		return nil, nil
	}
	artifacts := make([]Artifact, 0, 9)
	add := func(role ArtifactRole, format, path string) error {
		if path == "" {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat %s artifact %q: %w", role, path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s artifact %q is not a regular file", role, path)
		}
		artifacts = append(artifacts, Artifact{Role: role, Format: format, Path: path, Size: info.Size()})
		return nil
	}

	var primaryRole ArtifactRole
	switch conf.DebugArtifactMode {
	case DebugArtifactEmbedded:
		primaryRole = ArtifactRoleDebugDeployment
	case DebugArtifactExternal, DebugArtifactNone:
		primaryRole = ArtifactRoleDeployment
	case DebugArtifactHost:
		primaryRole = ArtifactRoleDebug
	default:
		return nil, fmt.Errorf("unresolved debug artifact mode %s", conf.DebugArtifactMode)
	}
	if out.Out == "" {
		return nil, fmt.Errorf("primary artifact path is empty")
	}
	modulePath := debugWasmModulePath(conf, out.Out)
	if modulePath != out.Out {
		if err := add(ArtifactRoleDeployment, primaryArtifactFormat(conf, out.Out), out.Out); err != nil {
			return nil, err
		}
		if err := add(primaryRole, "wasm", modulePath); err != nil {
			return nil, err
		}
		// emcc's HTML shell loads a separate JavaScript module. It is part
		// of this output contract, even when unrelated glue siblings exist.
		if filepath.Ext(out.Out) == ".html" {
			glue := strings.TrimSuffix(out.Out, ".html") + ".js"
			if err := add(ArtifactRoleDeployment, "javascript", glue); err != nil {
				return nil, err
			}
		}
	} else {
		if err := add(primaryRole, primaryArtifactFormat(conf, out.Out), out.Out); err != nil {
			return nil, err
		}
		// An explicit Emscripten .wasm output also owns a sibling ES module.
		// Report it when present; synthetic/tool-produced Wasm artifacts do not
		// necessarily carry JavaScript glue.
		if glue := emscriptenDriverOutput(conf, out.Out); glue != out.Out {
			if _, err := os.Stat(glue); err == nil {
				if err := add(ArtifactRoleDeployment, "javascript", glue); err != nil {
					return nil, err
				}
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	// Use the same ownership rule as publication. Raw GoJS and native outputs
	// must not pick up another build's wasm_fs.js from the output directory.
	if driverOut := emscriptenDriverOutput(conf, out.Out); needsEmscriptenBrowserHost(conf, driverOut) {
		if err := add(ArtifactRoleDeployment, "javascript", emscriptenBrowserHostSidecar(driverOut)); err != nil {
			return nil, err
		}
	}
	if err := add(ArtifactRoleDebug, "wasm-dwarf", out.DWARF); err != nil {
		return nil, err
	}
	if err := add(ArtifactRoleRuntimeSymbols, "pclntab", out.PCLN); err != nil {
		return nil, err
	}
	for _, deployment := range []struct {
		format string
		path   string
	}{
		{"bin", out.Bin},
		{"hex", out.Hex},
		{"img", out.Img},
		{"uf2", out.Uf2},
		{"zip", out.Zip},
	} {
		if err := add(ArtifactRoleDeployment, deployment.format, deployment.path); err != nil {
			return nil, err
		}
	}
	return artifacts, nil
}

func primaryArtifactFormat(conf *Config, path string) string {
	if conf.BuildMode == BuildModeCArchive {
		return "archive"
	}
	if conf.Goarch == "wasm" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".js", ".mjs":
			return "javascript"
		case ".html":
			return "html"
		}
	}
	if conf.Goarch == "wasm" || strings.EqualFold(filepath.Ext(path), ".wasm") {
		return "wasm"
	}
	if conf.Target != "" {
		return "elf"
	}
	switch conf.Goos {
	case "darwin":
		return "macho"
	case "windows":
		return "pe"
	case "linux", "android", "freebsd", "netbsd", "openbsd", "dragonfly", "solaris":
		return "elf"
	default:
		return "executable"
	}
}

// reportBuildOutputs shares one artifact snapshot between the size report and
// the artifact listing after all output transformations have completed.
func reportBuildOutputs(conf *Config, out *OutFmtDetails, pkgs []Package, sizeOutput, artifactOutput io.Writer) error {
	if conf == nil {
		return nil
	}
	wantSize := conf.Mode == ModeBuild && conf.SizeReport
	wantArtifacts := conf.DebugArtifactModeSet || conf.Target != ""
	if !wantSize && !wantArtifacts {
		return nil
	}
	artifacts, err := CollectArtifacts(conf, out)
	if err != nil {
		return err
	}
	if wantSize {
		if err := reportFinalSize(conf, out, pkgs, artifacts, sizeOutput); err != nil {
			return err
		}
	}
	if wantArtifacts {
		return reportBuildArtifacts(artifacts, artifactOutput)
	}
	return nil
}

func reportBuildArtifacts(artifacts []Artifact, w io.Writer) error {
	for _, artifact := range artifacts {
		if _, err := fmt.Fprintf(w, "llgo: artifact role=%s format=%s size=%d path=%q\n",
			artifact.Role, artifact.Format, artifact.Size, artifact.Path); err != nil {
			return err
		}
	}
	return nil
}
