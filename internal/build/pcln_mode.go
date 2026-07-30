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
	if !conf.PCLNModeSet && conf.PCLNMode == PCLNEmbedded && !isEnvOnConfig(conf, llgoFuncInfo, true) {
		return PCLNNone
	}
	return conf.PCLNMode
}

// shouldEnablePCLNSites reports whether compiler-emitted PC anchor records are
// required for this build. Darwin DWARF builds keep the historical site-free
// path because inline anchors disturb LLDB lexical scopes there. ELF cannot
// reconstruct all Go entry PCs with dlsym: most Go symbols are intentionally
// absent from .dynsym, so Linux keeps sites even when it emits DWARF.
func shouldEnablePCLNSites(conf *Config, funcInfo, emitDebugInfo bool) bool {
	if conf == nil || !funcInfo || !isEnvOnConfig(conf, llgoFuncInfoSites, true) {
		return false
	}
	return !emitDebugInfo || conf.Goos == "linux" || conf.PCLNMode == PCLNExternal
}

// validatePCLNMode checks whether the selected build can produce the requested
// PCLN artifact. External sidecars initially support native ELF and Mach-O
// executables only.
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
	if !isEnvOnConfig(conf, llgoFuncInfoSites, true) {
		return fmt.Errorf("external PCLN metadata requires LLGO_FUNCINFO_SITES to be enabled")
	}
	return nil
}
