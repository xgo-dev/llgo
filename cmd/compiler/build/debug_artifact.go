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
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
)

// DebugArtifactMode controls where debugger-owned DWARF is packaged. It is
// independent of runtime pclntab packaging.
type DebugArtifactMode uint8

const (
	// DebugArtifactDefault derives the effective mode from -w and the current
	// build default. It is valid only before build configuration is resolved.
	DebugArtifactDefault DebugArtifactMode = iota
	// DebugArtifactEmbedded retains DWARF in a Wasm module. Native builds only
	// request DWARF preservation and use the platform toolchain's default
	// packaging; a self-contained executable is not guaranteed.
	DebugArtifactEmbedded
	// DebugArtifactExternal writes a debugger-owned sidecar referenced by the
	// executable or module.
	DebugArtifactExternal
	// DebugArtifactHost retains a full host-side debug executable and derives a
	// separate deployment image from it.
	DebugArtifactHost
	// DebugArtifactNone omits LLGo DWARF, equivalent to an effective -w.
	DebugArtifactNone
)

func (m DebugArtifactMode) String() string {
	switch m {
	case DebugArtifactDefault:
		return "default"
	case DebugArtifactEmbedded:
		return "embedded"
	case DebugArtifactExternal:
		return "external"
	case DebugArtifactHost:
		return "host"
	case DebugArtifactNone:
		return "none"
	default:
		return fmt.Sprintf("DebugArtifactMode(%d)", uint8(m))
	}
}

// IsValid reports whether m is a recognized debug-artifact mode.
func (m DebugArtifactMode) IsValid() bool {
	switch m {
	case DebugArtifactDefault, DebugArtifactEmbedded, DebugArtifactExternal, DebugArtifactHost, DebugArtifactNone:
		return true
	default:
		return false
	}
}

func isWasmDebugTarget(conf *Config, target *crosscompile.Export) bool {
	return conf.Goarch == "wasm" || strings.HasPrefix(target.LLVMTarget, "wasm")
}

// resolveDebugArtifactMode validates an explicit artifact request, translates
// it into typed -w intent, and records the effective packaging mode. The
// current build/target DWARF policy remains authoritative when no artifact mode
// is supplied.
func resolveDebugArtifactMode(conf *Config, target *crosscompile.Export) error {
	conf.debugPointerSize = 4
	if target.WasmProfile == crosscompile.WasmProfileJ64 || strings.HasPrefix(target.LLVMTarget, "wasm64") {
		conf.debugPointerSize = 8
	}
	if !conf.DebugArtifactMode.IsValid() {
		return fmt.Errorf("invalid debug artifact mode %d", conf.DebugArtifactMode)
	}
	if conf.DebugArtifactModeSet && conf.DebugArtifactMode == DebugArtifactDefault {
		return fmt.Errorf("debug artifact mode default cannot be selected explicitly")
	}

	if conf.DebugArtifactModeSet {
		switch conf.DebugArtifactMode {
		case DebugArtifactNone:
			if conf.LinkOptions.DWARF == DWARFPreserve {
				return fmt.Errorf("debug artifact mode none conflicts with -w=false")
			}
			conf.LinkOptions.DWARF = DWARFOmit
		case DebugArtifactExternal:
			if !isWasmDebugTarget(conf, target) {
				return fmt.Errorf("external debug artifacts are currently supported only for WebAssembly")
			}
			if conf.Mode == ModeGen || conf.BuildMode != BuildModeExe {
				return fmt.Errorf("external debug artifacts require an executable build")
			}
			if conf.LinkOptions.DWARF == DWARFOmit {
				return fmt.Errorf("debug artifact mode external conflicts with -w")
			}
			conf.LinkOptions.DWARF = DWARFPreserve
		case DebugArtifactHost:
			if conf.Target == "" || isWasmDebugTarget(conf, target) {
				return fmt.Errorf("host debug artifacts require a non-WebAssembly target build")
			}
			if conf.Mode == ModeGen || conf.BuildMode != BuildModeExe {
				return fmt.Errorf("host debug artifacts require an executable build")
			}
			if conf.LinkOptions.DWARF == DWARFOmit {
				return fmt.Errorf("debug artifact mode host conflicts with -w")
			}
			conf.LinkOptions.DWARF = DWARFPreserve
		case DebugArtifactEmbedded:
			if conf.Mode == ModeGen {
				return fmt.Errorf("embedded debug artifacts are not supported in generation mode")
			}
			if conf.Target != "" && !isWasmDebugTarget(conf, target) {
				return fmt.Errorf("non-WebAssembly target builds use host debug artifacts")
			}
			if conf.LinkOptions.DWARF == DWARFOmit {
				return fmt.Errorf("debug artifact mode embedded conflicts with -w")
			}
			conf.LinkOptions.DWARF = DWARFPreserve
		}
	}

	if conf.DebugArtifactMode == DebugArtifactDefault {
		if effectiveOmitDWARF(conf, target) {
			conf.DebugArtifactMode = DebugArtifactNone
		} else if conf.Target != "" && !isWasmDebugTarget(conf, target) {
			conf.DebugArtifactMode = DebugArtifactHost
		} else {
			conf.DebugArtifactMode = DebugArtifactEmbedded
		}
	}
	return nil
}
