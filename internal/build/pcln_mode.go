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

import "fmt"

// PCLNMode controls how Go program counter and line metadata is packaged.
type PCLNMode uint8

const (
	// PCLNEmbedded keeps the metadata in the executable. It is the default.
	PCLNEmbedded PCLNMode = iota
	// PCLNExternal writes the metadata to an optional sidecar file.
	PCLNExternal
	// PCLNNone omits the metadata and its loading support.
	PCLNNone
)

func (m PCLNMode) String() string {
	switch m {
	case PCLNEmbedded:
		return "embedded"
	case PCLNExternal:
		return "external"
	case PCLNNone:
		return "none"
	default:
		return fmt.Sprintf("PCLNMode(%d)", uint8(m))
	}
}

// IsValid reports whether m is a recognized PCLN mode.
func (m PCLNMode) IsValid() bool {
	switch m {
	case PCLNEmbedded, PCLNExternal, PCLNNone:
		return true
	default:
		return false
	}
}

func (m PCLNMode) validate() error {
	if !m.IsValid() {
		return fmt.Errorf("invalid PCLN mode %d", m)
	}
	return nil
}

// effectivePCLNMode translates the legacy environment escape hatch once at
// the configuration boundary. An explicit -pclntab value always wins.
func effectivePCLNMode(conf *Config) PCLNMode {
	if !conf.PCLNModeSet && conf.PCLNMode == PCLNEmbedded && !IsFuncInfoEnabled() {
		return PCLNNone
	}
	return conf.PCLNMode
}

// shouldEnablePCLNSites reports whether compiler-emitted PC anchor records are
// globally enabled for this build. Target-specific filtering of address-site
// categories happens when the runtime tables are emitted.
func shouldEnablePCLNSites(conf *Config, funcInfo bool) bool {
	return conf != nil && funcInfo && IsFuncInfoSitesEnabled()
}

// validatePCLNMode checks whether the selected build can produce the requested
// PCLN artifact. External sidecars support native ELF/Mach-O executables and
// the WASI and Emscripten Wasm hosts.
func validatePCLNMode(conf *Config) error {
	if err := conf.PCLNMode.validate(); err != nil {
		return err
	}
	if conf.PCLNMode != PCLNExternal {
		return nil
	}
	if conf.Mode == ModeGen {
		return fmt.Errorf("external PCLN metadata is not supported in generation mode")
	}
	if conf.BuildMode != BuildModeExe {
		return fmt.Errorf("external PCLN metadata is not supported for -buildmode=%s", conf.BuildMode)
	}
	if !IsFuncInfoSitesEnabled() {
		return fmt.Errorf("external PCLN metadata requires LLGO_FUNCINFO_SITES to be enabled")
	}
	if conf.Goarch == "wasm" {
		if conf.Goos == "js" || conf.Goos == "wasip1" {
			return nil
		}
		return fmt.Errorf("external PCLN metadata is not supported for GOOS=%s", conf.Goos)
	}
	if conf.Target != "" {
		return fmt.Errorf("external PCLN metadata is not supported for target %q", conf.Target)
	}
	switch conf.Goos {
	case "darwin", "linux":
	default:
		return fmt.Errorf("external PCLN metadata is not supported for GOOS=%s", conf.Goos)
	}
	switch conf.Goarch {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("external PCLN metadata is not supported for GOARCH=%s", conf.Goarch)
	}
	return nil
}
