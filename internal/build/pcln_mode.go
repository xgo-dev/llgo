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
	"slices"

	"github.com/xgo-dev/llgo/internal/targets"
)

// PCLNMode controls how Go program counter and line metadata is packaged.
type PCLNMode uint8

// PCLNLoadPolicy controls when a hosted Wasm executable loads its sidecar.
type PCLNLoadPolicy string

const (
	PCLNLoadLazy       PCLNLoadPolicy = ""
	PCLNLoadRequired   PCLNLoadPolicy = "required"
	PCLNLoadBackground PCLNLoadPolicy = "background"
	PCLNLoadNone       PCLNLoadPolicy = "none"
)

func (p PCLNLoadPolicy) startsAtEntry() bool {
	return p == PCLNLoadRequired || p == PCLNLoadBackground
}

func (p PCLNLoadPolicy) buildTags() string {
	if p == PCLNLoadLazy {
		return ""
	}
	return ",llgo_pclntab_explicit,llgo_pclntab_" + string(p)
}

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
	if conf.PCLNModeSet || conf.PCLNMode != PCLNEmbedded {
		return conf.PCLNMode
	}
	if !IsFuncInfoEnabled() {
		return PCLNNone
	}
	// Baremetal runtimes cannot consume embedded funcinfo tables: the
	// baremetal site object format is unsupported, so entry-site records
	// never resolve and traceback lookups always fail. Emitting the tables
	// is pure dead weight there, so default baremetal builds to no
	// metadata; an explicit LLGO_FUNCINFO=1 (1, true or on) opts back in.
	// Hosted builds and non-baremetal targets keep the embedded default:
	// their runtimes link the tables (e.g. wasi via runtime/internal/pcln's
	// wasm ABI).
	if conf.Target != "" && isBaremetalTarget(conf.Target) && !isEnvExplicitlyOn(llgoFuncInfo) {
		return PCLNNone
	}
	return PCLNEmbedded
}

// isBaremetalTarget reports whether the named target declares the baremetal
// build tag. Targets that fail to resolve keep the baremetal assumption;
// they are rejected by target resolution later regardless.
func isBaremetalTarget(name string) bool {
	config, err := targets.NewDefaultResolver().Resolve(name)
	if err != nil {
		return true
	}
	return slices.Contains(config.BuildTags, "baremetal")
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
	switch conf.PCLNLoad {
	case PCLNLoadLazy:
	case PCLNLoadRequired, PCLNLoadBackground, PCLNLoadNone:
		if conf.PCLNMode != PCLNExternal || conf.Goarch != "wasm" {
			return fmt.Errorf("-pclntab-load=%s requires external metadata on a hosted Wasm target", conf.PCLNLoad)
		}
	default:
		return fmt.Errorf("invalid PCLN loading policy %q", conf.PCLNLoad)
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
