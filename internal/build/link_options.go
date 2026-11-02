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
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/quoted"
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

// DWARFCompressionMode distinguishes platform defaults from an explicit
// -compressdwarf request. Mach-O/PE compression is experimental and opt-in.
type DWARFCompressionMode uint8

const (
	DWARFCompressionDefault DWARFCompressionMode = iota
	DWARFCompressionEnabled
	DWARFCompressionDisabled
)

// LinkOptions is the typed linker intent consumed by the build backend.
// Command packages are responsible for parsing user-facing flag syntax into
// this representation.
type LinkOptions struct {
	// OmitSymbolTable implements -s for native ELF/Mach-O linked outputs,
	// after runtime PCLN has consumed the linker's ordinary symbols.
	OmitSymbolTable bool
	DWARF           DWARFMode
	// Native Linux links compress retained DWARF by default. Other native
	// formats require an explicit -compressdwarf=true request.
	DWARFCompression DWARFCompressionMode
	// ExternalLinker and ExternalLinkerFlags retain the Go linker's -extld
	// command and -extldflags argument-list spelling. The native toolchain
	// resolver applies Go's quoted command parsing after target selection.
	ExternalLinker      string
	ExternalLinkerFlags string
}

func (o LinkOptions) validate() error {
	switch o.DWARFCompression {
	case DWARFCompressionDefault, DWARFCompressionEnabled, DWARFCompressionDisabled:
	default:
		return fmt.Errorf("invalid DWARF compression mode %d", o.DWARFCompression)
	}
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
	if conf.LinkOptions.DWARF != DWARFDefault {
		return conf.LinkOptions.EffectiveOmitDWARF()
	}
	if conf.OmitDWARFByDefault {
		return true
	}
	if conf.Mode != ModeGen {
		// Keep fixed-target and Wasm artifacts opt-in for DWARF.
		if conf.Target != "" || conf.Goarch == "wasm" {
			return true
		}
		// Match cmd/link's Darwin c-shared default.
		if conf.Goos == "darwin" && conf.BuildMode == BuildModeCShared {
			return true
		}
	}
	return conf.LinkOptions.EffectiveOmitDWARF()
}

// effectiveOmitDWARF combines command intent with the selected toolchain's
// capability. LLGo avoids generating debug metadata when the linked output
// format cannot retain it.
func effectiveOmitDWARF(conf *Config, target *crosscompile.Export) bool {
	return omitDWARFRequested(conf) || !target.DebugInfo.CanRetain()
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

// shouldEmitCodeView reports whether an MSVC COFF link explicitly requests a
// PDB. LLGo keeps its Go-compatible DWARF by default; CodeView is additional
// metadata for an explicitly requested native Windows debugger artifact.
func shouldEmitCodeView(conf *Config, target *crosscompile.Export) bool {
	return shouldEmitDebugInfo(conf, target) && conf.Goos == "windows" &&
		target.Toolchain.ABI == crosscompile.PlatformABIMsvc &&
		hasCOFFPDBFlag(conf.LinkOptions.ExternalLinkerFlags)
}

// validateLinkOptions checks whether the typed linker intent can be honored
// by the selected backend. User-facing Go flag syntax is parsed by
// internal/goflags.
func validateLinkOptions(conf *Config, target *crosscompile.Export) error {
	if err := conf.LinkOptions.validate(); err != nil {
		return err
	}
	if conf.LinkOptions.DWARF == DWARFPreserve && !target.DebugInfo.CanRetain() {
		return fmt.Errorf("preserving DWARF is not supported by the selected target linker")
	}
	if !omitDWARFRequested(conf) {
		return nil
	}
	if !target.DebugInfo.CanRetain() {
		return nil
	}
	if len(target.DebugInfo.OmitLinkFlags) == 0 {
		return fmt.Errorf("DWARF omission is not supported for GOOS=%s", conf.Goos)
	}
	return nil
}

// debugInfoLinkerArgs asks the native linker to preserve or omit debug
// information consistently with the compile-time policy. Some linkers, such
// as lld-link, discard DWARF unless preservation is requested explicitly.
func debugInfoLinkerArgs(conf *Config, target *crosscompile.Export) []string {
	if !target.DebugInfo.CanRetain() {
		return nil
	}
	// c-archive has no final native link step. Consumers decide how to link
	// debug information from archive members later.
	if conf.BuildMode == BuildModeCArchive {
		return nil
	}
	if effectiveOmitDWARF(conf, target) {
		return slices.Clone(target.DebugInfo.OmitLinkFlags)
	}
	// The default COFF policy keeps Go-compatible DWARF in the PE image. An
	// explicit external-linker /debug option may instead request a PDB (for
	// example /debug:full). Do not append /debug:dwarf after that user choice:
	// lld-link applies the last /debug option and would silently suppress the
	// requested PDB. DWARF omission above still wins when Go's -w is active.
	if conf.Goos == "windows" && hasCOFFDebugFlag(conf.LinkOptions.ExternalLinkerFlags) {
		return nil
	}
	args := slices.Clone(target.DebugInfo.PreserveLinkFlags)
	if conf.Target == "" && conf.Goos == "linux" &&
		target.Toolchain.ObjectFormat == crosscompile.ObjectFormatELF &&
		conf.LinkOptions.DWARFCompression != DWARFCompressionDisabled {
		args = append(args, "-Wl,--compress-debug-sections=zlib")
	}
	return args
}

// debugInfoCompilerArgs keeps package C/C++ sources on the same typed debug
// policy as generated Go code. COFF PDB builds add CodeView while retaining
// DWARF, so LLDB, Go tracebacks, and native Windows debuggers share the same
// source inputs.
func debugInfoCompilerArgs(conf *Config, target *crosscompile.Export) []string {
	if shouldEmitDebugInfo(conf, target) {
		args := []string{"-gdwarf-4"}
		if shouldEmitCodeView(conf, target) {
			args = append(args, "-gcodeview")
		}
		return args
	}
	return nil
}

func hasCOFFPDBFlag(value string) bool {
	pdb := false
	hasCOFFDebugFlagMatching(value, func(arg string) bool {
		if isCOFFDebugFlag(arg) {
			// lld-link applies the last /debug option. Preserve that ordering so
			// "/debug:full /debug:none" does not add unused CodeView records.
			pdb = isCOFFPDBFlag(arg)
		}
		// Visit every argument so the last /debug option determines the result.
		return false
	})
	return pdb
}

func hasCOFFDebugFlag(value string) bool {
	return hasCOFFDebugFlagMatching(value, isCOFFDebugFlag)
}

func hasCOFFDebugFlagMatching(value string, match func(string) bool) bool {
	args, err := quoted.Split(value)
	if err != nil {
		// Native toolchain input validation reports the malformed value before a
		// real link. Keep this helper side-effect-free for configuration tests.
		return false
	}
	for i, arg := range args {
		lower := strings.ToLower(arg)
		if match(lower) {
			return true
		}
		if strings.HasPrefix(lower, "-wl,") {
			for _, linkerArg := range strings.Split(strings.TrimPrefix(lower, "-wl,"), ",") {
				if match(linkerArg) {
					return true
				}
			}
		}
		if lower == "-xlinker" && i+1 < len(args) && match(strings.ToLower(args[i+1])) {
			return true
		}
	}
	return false
}

func isCOFFPDBFlag(arg string) bool {
	if arg == "/debug" || arg == "-debug" {
		return true
	}
	for _, prefix := range []string{"/debug:", "-debug:"} {
		if value, ok := strings.CutPrefix(arg, prefix); ok {
			return value != "none" && value != "dwarf"
		}
	}
	return false
}

func isCOFFDebugFlag(arg string) bool {
	return arg == "/debug" || strings.HasPrefix(arg, "/debug:") ||
		arg == "-debug" || strings.HasPrefix(arg, "-debug:")
}
