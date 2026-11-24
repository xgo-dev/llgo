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

package target

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents a complete target configuration after inheritance resolution.
type Config struct {
	// Target identification
	Name string `json:"-"`

	// LLVM configuration
	LLVMTarget string `json:"llvm-target"`
	CPU        string `json:"cpu"`
	Features   string `json:"features"`

	// Build configuration
	BuildTags []string `json:"build-tags"`
	GOOS      string   `json:"goos"`
	GOARCH    string   `json:"goarch"`
	// WasmProfile identifies the memory/host ABI pair. WasmProvider selects the
	// concrete host implementation without creating another source profile.
	WasmProfile  string `json:"wasm-profile"`
	WasmProvider string `json:"wasm-provider"`
	// WASI linear-memory and process-stack budgets, in bytes. Zero inherits
	// the parent's value or keeps the toolchain default. These are link-only
	// settings; WasmStackSize does not change goroutine/pthread stack sizes.
	WasmInitialMemory uint64 `json:"wasm-initial-memory"`
	WasmMaxMemory     uint64 `json:"wasm-max-memory"`
	WasmStackSize     uint64 `json:"wasm-stack-size"`

	// Compiler and linker configuration
	Libc         string   `json:"libc"`
	RTLib        string   `json:"rtlib"`
	Linker       string   `json:"linker"`
	LinkerScript string   `json:"linkerscript"`
	CFlags       []string `json:"cflags"`
	LDFlags      []string `json:"ldflags"`
	ExtraFiles   []string `json:"extra-files"`

	// Code generation configuration
	CodeModel       string `json:"code-model"`
	TargetABI       string `json:"target-abi"`
	RelocationModel string `json:"relocation-model"`

	// Binary and firmware configuration
	BinaryFormat string `json:"binary-format"`
	// UF2 configuration
	UF2FamilyID string `json:"uf2-family-id"`

	// Flash and deployment configuration
	FlashMethod       string `json:"flash-method"`  // values: command, openocd, msd
	FlashCommand      string `json:"flash-command"` // used when FlashMethod == "command"
	Flash1200BpsReset string `json:"flash-1200-bps-reset"`

	// Serial configuration
	Serial     string   `json:"serial"`      // Serial communication type (e.g., "usb")
	SerialPort []string `json:"serial-port"` // Serial port identifiers (e.g., vendor:product IDs)

	// Mass storage device configuration
	MSDVolumeName   []string `json:"msd-volume-name"`
	MSDFirmwareName string   `json:"msd-firmware-name"`

	// Device-specific configuration
	RP2040BootPatch bool `json:"rp2040-boot-patch"`

	// Debug and emulation configuration
	Emulator    string   `json:"emulator"`
	DebugServer string   `json:"debug-server"`
	GDB         []string `json:"gdb"`

	// OpenOCD configuration
	OpenOCDInterface string `json:"openocd-interface"`
	OpenOCDTransport string `json:"openocd-transport"`
	OpenOCDTarget    string `json:"openocd-target"`
}

// RawConfig represents the raw JSON configuration before inheritance resolution.
type RawConfig struct {
	Inherits []string `json:"inherits"`
	Config
}

// IsEmpty reports whether the config appears to be uninitialized.
func (c *Config) IsEmpty() bool {
	return c.Name == "" && c.LLVMTarget == "" && c.GOOS == "" && c.GOARCH == ""
}

// FormatDetail returns format-specific metadata such as the UF2 family ID.
func (c *Config) FormatDetail() string {
	if c.BinaryFormat == "uf2" {
		return c.UF2FamilyID
	}
	return ""
}

// ToSpec derives a resolved pure-Go Spec from this JSON target Config.
//
// Matching ssa.Target.Spec, Spec.Triple keeps the GOOS/GOARCH-derived backend
// triple for non-Wasm named embedded targets (because external cross-compilers
// may target architectures such as Xtensa that the host LLVM library does not
// register), while Spec.PhysicalTriple preserves Config.LLVMTarget when set.
func (c *Config) ToSpec() Spec {
	spec := ResolveSpec(Spec{
		Name:         c.Name,
		GOOS:         c.GOOS,
		GOARCH:       c.GOARCH,
		LLVMTarget:   c.LLVMTarget,
		WasmProfile:  c.WasmProfile,
		WasmProvider: c.WasmProvider,
		TargetABI:    c.TargetABI,
	})
	if c.CPU != "" {
		spec.CPU = c.CPU
	}
	if c.Features != "" {
		spec.Features = c.Features
	}
	return spec
}

// HasInheritance reports whether this config inherits from parent configs.
func (rc *RawConfig) HasInheritance() bool {
	return len(rc.Inherits) > 0
}

// GetInherits returns the list of parent target names.
func (rc *RawConfig) GetInherits() []string {
	return rc.Inherits
}

// Loader handles loading and resolving target JSON files from an explicit directory.
type Loader struct {
	targetsDir string
	cache      map[string]*RawConfig
}

// NewLoader creates a target configuration loader rooted at targetsDir.
func NewLoader(targetsDir string) *Loader {
	return &Loader{
		targetsDir: targetsDir,
		cache:      make(map[string]*RawConfig),
	}
}

func validateTargetName(name string) error {
	if name == "" || name == "." || name == ".." ||
		filepath.IsAbs(name) ||
		strings.ContainsAny(name, `/\`) ||
		strings.Contains(name, "..") {
		return fmt.Errorf("invalid target name %q", name)
	}
	return nil
}

// LoadRaw loads a raw configuration without resolving inheritance.
func (l *Loader) LoadRaw(name string) (*RawConfig, error) {
	if err := validateTargetName(name); err != nil {
		return nil, err
	}
	if config, exists := l.cache[name]; exists {
		return config, nil
	}
	configPath := filepath.Join(l.targetsDir, name+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read target config %s: %w", name, err)
	}
	var config RawConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse target config %s: %w", name, err)
	}
	config.Name = name
	l.cache[name] = &config
	return &config, nil
}

// Load loads a target configuration with inheritance resolved and cycle detection.
func (l *Loader) Load(name string) (*Config, error) {
	return l.loadWithStack(name, nil)
}

func (l *Loader) loadWithStack(name string, stack []string) (*Config, error) {
	for _, seen := range stack {
		if seen == name {
			return nil, fmt.Errorf("target inheritance cycle detected: %s -> %s", strings.Join(stack, " -> "), name)
		}
	}
	raw, err := l.LoadRaw(name)
	if err != nil {
		return nil, err
	}
	return l.resolveInheritance(raw, append(stack, name))
}

// LoadAll loads all target configurations in the loader's targets directory.
func (l *Loader) LoadAll() (map[string]*Config, error) {
	entries, err := os.ReadDir(l.targetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read targets directory: %w", err)
	}
	configs := make(map[string]*Config)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		config, err := l.Load(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load target %s: %w", name, err)
		}
		configs[name] = config
	}
	return configs, nil
}

func (l *Loader) resolveInheritance(raw *RawConfig, stack []string) (*Config, error) {
	if !raw.HasInheritance() {
		return &raw.Config, nil
	}
	result := &Config{Name: raw.Name}
	for _, parentName := range raw.GetInherits() {
		parent, err := l.loadWithStack(parentName, stack)
		if err != nil {
			return nil, fmt.Errorf("failed to load parent config %s: %w", parentName, err)
		}
		mergeConfig(result, parent)
	}
	mergeConfig(result, &raw.Config)
	return result, nil
}

func mergeConfig(dst, src *Config) {
	if src.LLVMTarget != "" {
		dst.LLVMTarget = src.LLVMTarget
	}
	if src.CPU != "" {
		dst.CPU = src.CPU
	}
	if src.Features != "" {
		dst.Features = src.Features
	}
	if src.GOOS != "" {
		dst.GOOS = src.GOOS
	}
	if src.GOARCH != "" {
		dst.GOARCH = src.GOARCH
	}
	if src.WasmProfile != "" {
		dst.WasmProfile = src.WasmProfile
	}
	if src.WasmProvider != "" {
		dst.WasmProvider = src.WasmProvider
	}
	if src.WasmInitialMemory != 0 {
		dst.WasmInitialMemory = src.WasmInitialMemory
	}
	if src.WasmMaxMemory != 0 {
		dst.WasmMaxMemory = src.WasmMaxMemory
	}
	if src.WasmStackSize != 0 {
		dst.WasmStackSize = src.WasmStackSize
	}
	if src.Libc != "" {
		dst.Libc = src.Libc
	}
	if src.RTLib != "" {
		dst.RTLib = src.RTLib
	}
	if src.Linker != "" {
		dst.Linker = src.Linker
	}
	if src.LinkerScript != "" {
		dst.LinkerScript = src.LinkerScript
	}
	if src.CodeModel != "" {
		dst.CodeModel = src.CodeModel
	}
	if src.TargetABI != "" {
		dst.TargetABI = src.TargetABI
	}
	if src.RelocationModel != "" {
		dst.RelocationModel = src.RelocationModel
	}
	if src.BinaryFormat != "" {
		dst.BinaryFormat = src.BinaryFormat
	}
	if src.FlashCommand != "" {
		dst.FlashCommand = src.FlashCommand
	}
	if src.FlashMethod != "" {
		dst.FlashMethod = src.FlashMethod
	}
	if src.Flash1200BpsReset != "" {
		dst.Flash1200BpsReset = src.Flash1200BpsReset
	}
	if src.Serial != "" {
		dst.Serial = src.Serial
	}
	if src.MSDFirmwareName != "" {
		dst.MSDFirmwareName = src.MSDFirmwareName
	}
	if src.UF2FamilyID != "" {
		dst.UF2FamilyID = src.UF2FamilyID
	}
	if src.RP2040BootPatch {
		dst.RP2040BootPatch = src.RP2040BootPatch
	}
	if src.Emulator != "" {
		dst.Emulator = src.Emulator
	}
	if src.DebugServer != "" {
		dst.DebugServer = src.DebugServer
	}
	if src.OpenOCDInterface != "" {
		dst.OpenOCDInterface = src.OpenOCDInterface
	}
	if src.OpenOCDTransport != "" {
		dst.OpenOCDTransport = src.OpenOCDTransport
	}
	if src.OpenOCDTarget != "" {
		dst.OpenOCDTarget = src.OpenOCDTarget
	}
	if len(src.BuildTags) > 0 {
		dst.BuildTags = append(dst.BuildTags, src.BuildTags...)
	}
	if len(src.CFlags) > 0 {
		dst.CFlags = append(dst.CFlags, src.CFlags...)
	}
	if len(src.LDFlags) > 0 {
		dst.LDFlags = append(dst.LDFlags, src.LDFlags...)
	}
	if len(src.ExtraFiles) > 0 {
		dst.ExtraFiles = append(dst.ExtraFiles, src.ExtraFiles...)
	}
	if len(src.SerialPort) > 0 {
		dst.SerialPort = append(dst.SerialPort, src.SerialPort...)
	}
	if len(src.MSDVolumeName) > 0 {
		dst.MSDVolumeName = append(dst.MSDVolumeName, src.MSDVolumeName...)
	}
	if len(src.GDB) > 0 {
		dst.GDB = append(dst.GDB, src.GDB...)
	}
}

// GetTargetsDir returns the targets directory path.
func (l *Loader) GetTargetsDir() string {
	return l.targetsDir
}

// ListTargets returns all available target names in the directory.
func (l *Loader) ListTargets() ([]string, error) {
	entries, err := os.ReadDir(l.targetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read targets directory: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".json"))
	}
	return names, nil
}
