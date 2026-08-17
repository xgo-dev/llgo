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

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

// DWARFMode records whether the command layer requested that the backend
// preserve or omit DWARF. DWARFDefault leaves the decision to other link
// options.
type DWARFMode uint8

const (
	DWARFDefault DWARFMode = iota
	DWARFPreserve
	DWARFOmit
)

// LinkOptions is the typed linker intent consumed by the build backend.
// Command packages are responsible for parsing user-facing flag syntax into
// this representation.
type LinkOptions struct {
	// OmitSymbolTable records the requested -s intent. Phase one uses it for
	// Go's implied -w rule; native symbol-table omission is implemented later.
	OmitSymbolTable bool
	DWARF           DWARFMode
}

func (o LinkOptions) validate() error {
	switch o.DWARF {
	case DWARFDefault, DWARFPreserve, DWARFOmit:
		return nil
	default:
		return fmt.Errorf("invalid DWARF mode %d", o.DWARF)
	}
}

// EffectiveOmitDWARF reports whether the backend should omit DWARF. As in
// cmd/link, an explicit -w value takes precedence; otherwise -s implies -w.
func (o LinkOptions) EffectiveOmitDWARF() bool {
	switch o.DWARF {
	case DWARFPreserve:
		return false
	case DWARFOmit:
		return true
	default:
		return o.OmitSymbolTable
	}
}

// omitDWARFRequested combines explicit Go linker flags with LLGo's typed
// default. The default never overrides an explicit -w value.
func omitDWARFRequested(conf *Config) bool {
	if conf.LinkOptions.DWARF == DWARFDefault && conf.OmitDWARFByDefault {
		return true
	}
	return conf.LinkOptions.EffectiveOmitDWARF()
}

// effectiveOmitDWARF combines command intent with the selected toolchain's
// baseline behavior. Some fixed-target linkers always omit DWARF, so LLGo
// should avoid generating debug metadata that cannot reach the artifact.
func effectiveOmitDWARF(conf *Config, target *crosscompile.Export) bool {
	return omitDWARFRequested(conf) || target.DebugInfo.AlwaysOmit
}

// shouldEmitDebugInfo reports whether this compilation should produce DWARF.
// Linked modes use the typed LLGo default and target/linker constraints, with
// an explicit -w value taking precedence. ModeGen has no linker, so it emits
// only on an explicit preserve request.
func shouldEmitDebugInfo(conf *Config, target *crosscompile.Export) bool {
	if effectiveOmitDWARF(conf, target) {
		return false
	}
	return conf.Mode != ModeGen || conf.LinkOptions.DWARF == DWARFPreserve
}

// validateLinkOptions checks whether the typed linker intent can be honored
// by the selected backend. User-facing Go flag syntax is parsed by
// internal/goflags.
func validateLinkOptions(conf *Config, target *crosscompile.Export) error {
	if err := conf.LinkOptions.validate(); err != nil {
		return err
	}
	if conf.LinkOptions.DWARF == DWARFPreserve && target.DebugInfo.AlwaysOmit {
		return fmt.Errorf("preserving DWARF is not supported by the selected target linker")
	}
	if !omitDWARFRequested(conf) {
		return nil
	}
	if target.DebugInfo.AlwaysOmit {
		return nil
	}
	if len(target.DebugInfo.OmitLinkFlags) == 0 {
		return fmt.Errorf("DWARF omission is not supported for GOOS=%s", conf.Goos)
	}
	return nil
}

// dwarfLinkerArgs asks the native linker not to copy debug information from
// input objects into the final artifact. LLGo-generated DWARF is disabled
// earlier; this handles debug sections in native and prebuilt inputs without
// rewriting the linked binary afterward.
func dwarfLinkerArgs(conf *Config, target *crosscompile.Export) []string {
	if target.DebugInfo.AlwaysOmit || !effectiveOmitDWARF(conf, target) {
		return nil
	}
	// c-archive has no final native link step. Omitting generated DWARF is
	// sufficient; consumers decide how to link the archive later.
	if conf.BuildMode == BuildModeCArchive {
		return nil
	}
	return slices.Clone(target.DebugInfo.OmitLinkFlags)
}
