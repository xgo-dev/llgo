/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// depEntry captures dependency identity plus either version or fingerprint.
type depEntry struct {
	ID          string `yaml:"id"`
	Version     string `yaml:"version,omitempty"`
	Fingerprint string `yaml:"fingerprint,omitempty"`
}

// manifestMetadata stores metadata produced during build but not part of the fingerprint.
type manifestMetadata struct {
	LinkArgs   []string `yaml:"link_args,omitempty"`
	NeedRt     bool     `yaml:"need_rt,omitempty"`
	NeedPyInit bool     `yaml:"need_py_init,omitempty"`
}

// manifestData is the structured representation of manifest content.
type manifestData struct {
	Env      *envSection       `yaml:"env,omitempty"`
	Common   *commonSection    `yaml:"common,omitempty"`
	Package  *packageSection   `yaml:"package,omitempty"`
	Metadata *manifestMetadata `yaml:"metadata,omitempty"`
	Deps     []depEntry        `yaml:"deps,omitempty"`
}

// orderedStringMap keeps deterministic order for map[string]string when marshaling.
type orderedStringMap map[string]string

func (m orderedStringMap) Add(key, val string) orderedStringMap {
	if m == nil {
		m = make(map[string]string)
	}
	m[key] = val
	return m
}

func (m orderedStringMap) AddMap(src map[string]string) orderedStringMap {
	if len(src) == 0 {
		return m
	}
	if m == nil {
		m = make(map[string]string, len(src))
	}
	for k, v := range src {
		m[k] = v
	}
	return m
}

func (m orderedStringMap) MarshalYAML() (interface{}, error) {
	if len(m) == 0 {
		return nil, nil
	}
	type kv struct{ K, V string }
	list := make([]kv, 0, len(m))
	for k, v := range m {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].K < list[j].K })
	out := &yaml.Node{Kind: yaml.MappingNode}
	for _, item := range list {
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: item.K},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: item.V},
		)
	}
	return out, nil
}

// envSection holds fixed environment fields and optional vars.
type envSection struct {
	Goos             string           `yaml:"GOOS,omitempty"`
	Goarch           string           `yaml:"GOARCH,omitempty"`
	Go386            string           `yaml:"GO386,omitempty"`
	Goamd64          string           `yaml:"GOAMD64,omitempty"`
	Goarm            string           `yaml:"GOARM,omitempty"`
	Goarm64          string           `yaml:"GOARM64,omitempty"`
	Goexperiment     string           `yaml:"GOEXPERIMENT,omitempty"`
	SourceGoVersion  string           `yaml:"SOURCE_GO_VERSION,omitempty"`
	ToolTags         []string         `yaml:"TOOL_TAGS,omitempty"`
	GoVersion        string           `yaml:"GO_VERSION,omitempty"`
	LlgoVersion      string           `yaml:"LLGO_VERSION,omitempty"`
	LlgoCompilerHash string           `yaml:"LLGO_COMPILER_HASH,omitempty"`
	LlvmTriple       string           `yaml:"LLVM_TRIPLE,omitempty"`
	LlvmVersion      string           `yaml:"LLVM_VERSION,omitempty"`
	Vars             orderedStringMap `yaml:"VARS,omitempty"`
}

func (s *envSection) empty() bool {
	return s.Goos == "" && s.Goarch == "" && s.Go386 == "" && s.Goamd64 == "" && s.Goarm == "" && s.Goarm64 == "" && s.Goexperiment == "" && s.SourceGoVersion == "" && len(s.ToolTags) == 0 && s.LlvmTriple == "" && s.LlgoVersion == "" && s.LlgoCompilerHash == "" && s.GoVersion == "" && s.LlvmVersion == "" && len(s.Vars) == 0
}

type commonSection struct {
	BuildTags               []string     `yaml:"BUILD_TAGS,omitempty"`
	Target                  string       `yaml:"TARGET,omitempty"`
	TargetABI               string       `yaml:"TARGET_ABI,omitempty"`
	WasmABI                 string       `yaml:"WASM_ABI,omitempty"`
	PlatformABI             string       `yaml:"PLATFORM_ABI,omitempty"`
	ObjectFormat            string       `yaml:"OBJECT_FORMAT,omitempty"`
	DriverFlavor            string       `yaml:"DRIVER_FLAVOR,omitempty"`
	LinkerFlavor            string       `yaml:"LINKER_FLAVOR,omitempty"`
	TargetTriple            string       `yaml:"TARGET_TRIPLE,omitempty"`
	CRTFlavor               string       `yaml:"CRT_FLAVOR,omitempty"`
	CXXRuntime              string       `yaml:"CXX_RUNTIME,omitempty"`
	SDKVersion              string       `yaml:"SDK_VERSION,omitempty"`
	CRTVersion              string       `yaml:"CRT_VERSION,omitempty"`
	ToolsetVersion          string       `yaml:"TOOLSET_VERSION,omitempty"`
	GoGlobalDCE             bool         `yaml:"GO_GLOBAL_DCE,omitempty"`
	EnableLTOPlugin         bool         `yaml:"ENABLE_LTO_PLUGIN,omitempty"`
	EmitDWARF               bool         `yaml:"EMIT_DWARF,omitempty"`
	EmitCodeView            bool         `yaml:"EMIT_CODEVIEW,omitempty"`
	PCLNMode                string       `yaml:"PCLN_MODE,omitempty"`
	DisableBoundsChecks     bool         `yaml:"DISABLE_BOUNDS_CHECKS,omitempty"`
	SaturatingFloatToUint32 bool         `yaml:"SATURATING_FLOAT_TO_UINT32,omitempty"`
	LocalContext            bool         `yaml:"LOCAL_CONTEXT,omitempty"`
	CC                      string       `yaml:"CC,omitempty"`
	CCArgs                  []string     `yaml:"CC_ARGS,omitempty"`
	CCIdentity              string       `yaml:"CC_IDENTITY,omitempty"`
	CXX                     string       `yaml:"CXX,omitempty"`
	CXXArgs                 []string     `yaml:"CXX_ARGS,omitempty"`
	CXXIdentity             string       `yaml:"CXX_IDENTITY,omitempty"`
	CCFlags                 []string     `yaml:"CCFLAGS,omitempty"`
	CFlags                  []string     `yaml:"CFLAGS,omitempty"`
	LDFlags                 []string     `yaml:"LDFLAGS,omitempty"`
	Linker                  string       `yaml:"LINKER,omitempty"`
	LinkerArgs              []string     `yaml:"LINKER_ARGS,omitempty"`
	LinkerIdentity          string       `yaml:"LINKER_IDENTITY,omitempty"`
	ExtraFiles              []fileDigest `yaml:"EXTRA_FILES,omitempty"`
}

func (s *commonSection) empty() bool {
	return len(s.BuildTags) == 0 && s.Target == "" && s.TargetABI == "" && s.WasmABI == "" &&
		s.PlatformABI == "" && s.ObjectFormat == "" && s.DriverFlavor == "" && s.LinkerFlavor == "" &&
		s.TargetTriple == "" && s.CRTFlavor == "" && s.CXXRuntime == "" &&
		s.SDKVersion == "" && s.CRTVersion == "" && s.ToolsetVersion == "" &&
		!s.GoGlobalDCE && !s.EnableLTOPlugin && !s.EmitDWARF && !s.EmitCodeView && s.PCLNMode == "" &&
		!s.DisableBoundsChecks && !s.SaturatingFloatToUint32 && !s.LocalContext &&
		s.CC == "" && len(s.CCArgs) == 0 && s.CCIdentity == "" && s.CXX == "" &&
		len(s.CXXArgs) == 0 && s.CXXIdentity == "" && len(s.CCFlags) == 0 &&
		len(s.CFlags) == 0 && len(s.LDFlags) == 0 && s.Linker == "" &&
		len(s.LinkerArgs) == 0 && s.LinkerIdentity == "" && len(s.ExtraFiles) == 0
}

type packageSection struct {
	PthreadStackSize int64            `yaml:"pthread_stack_size,omitempty"`
	PkgPath          string           `yaml:"pkg_path,omitempty"`
	PkgID            string           `yaml:"pkg_id,omitempty"`
	GoFiles          []fileDigest     `yaml:"go_files,omitempty"`
	AltGoFiles       []fileDigest     `yaml:"alt_go_files,omitempty"`
	OtherFiles       []fileDigest     `yaml:"other_files,omitempty"`
	LLGoFiles        []llgoFileDigest `yaml:"llgo_files,omitempty"`
	RewriteVars      orderedStringMap `yaml:"rewrite_vars,omitempty"`
}

func (s *packageSection) empty() bool {
	return s.PkgPath == "" && s.PkgID == "" && len(s.GoFiles) == 0 && len(s.AltGoFiles) == 0 &&
		len(s.OtherFiles) == 0 && len(s.LLGoFiles) == 0 && len(s.RewriteVars) == 0 && s.PthreadStackSize == 0
}

// manifestBuilder builds manifest text with sorted sections.
type manifestBuilder struct {
	env    envSection
	common commonSection
	pkg    packageSection
	deps   []depEntry
	meta   *manifestMetadata
}

// newManifestBuilder creates a new manifestBuilder.
func newManifestBuilder() *manifestBuilder {
	return &manifestBuilder{}
}

// Build generates the sorted manifest text in INI format.
func (m *manifestBuilder) Build() string {
	env := m.env
	common := m.common
	pkg := m.pkg

	sort.Strings(common.BuildTags)

	data := manifestData{
		Env:      &env,
		Common:   &common,
		Package:  &pkg,
		Deps:     sortDeps(m.deps),
		Metadata: m.meta,
	}
	content, _ := buildManifestYAML(data)
	return content
}

// Fingerprint returns the sha256 hash of the manifest content.
func (m *manifestBuilder) Fingerprint() string {
	content := m.Build()
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func sortDeps(deps []depEntry) []depEntry {
	if len(deps) == 0 {
		return nil
	}
	sorted := append([]depEntry(nil), deps...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ID == sorted[j].ID {
			if sorted[i].Version == sorted[j].Version {
				return sorted[i].Fingerprint < sorted[j].Fingerprint
			}
			return sorted[i].Version < sorted[j].Version
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func (d manifestData) isEmpty() bool {
	return (d.Env == nil || d.Env.empty()) &&
		(d.Common == nil || d.Common.empty()) &&
		(d.Package == nil || d.Package.empty()) &&
		len(d.Deps) == 0 && d.Metadata == nil
}

func buildManifestYAML(data manifestData) (string, error) {
	if data.isEmpty() {
		return "", nil
	}
	out, err := yaml.Marshal(data)
	return string(out), err
}

const maxManifestSize = 10 * 1024 * 1024 // 10MB safety bound

func decodeManifest(content string) (manifestData, error) {
	var data manifestData
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return data, nil
	}
	if len(trimmed) > maxManifestSize {
		return manifestData{}, fmt.Errorf("manifest too large: %d bytes", len(trimmed))
	}
	if err := yaml.Unmarshal([]byte(trimmed), &data); err != nil {
		return manifestData{}, err
	}
	return data, nil
}

// digestFile calculates the sha256 hash of a file.
func digestFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// digestBytes calculates the sha256 hash of bytes.
func digestBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// fileDigest represents a file with its path and metadata.
type fileDigest struct {
	Path        string `yaml:"path"`
	Size        int64  `yaml:"size"`
	ModTime     int64  `yaml:"mtime"`
	OverlayHash string `yaml:"overlay_hash,omitempty"`
}

// llgoFileDigest records the preprocessed content and per-file flags of an
// LLGoFiles input. These files are declared through a Go constant, so go list
// does not include either them or their local includes in OtherFiles.
type llgoFileDigest struct {
	Path             string   `yaml:"path"`
	PreprocessedHash string   `yaml:"preprocessed_hash"`
	OverlayHash      string   `yaml:"overlay_hash,omitempty"`
	CompilerArgs     []string `yaml:"compiler_args,omitempty"`
}

func digestLLGoFileInputs(ctx *context, inputs []llgoFileInput, overlay map[string][]byte) ([]llgoFileDigest, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if ctx.llgoFileHashCache == nil {
		ctx.llgoFileHashCache = make(map[llgoFileHashKey]string)
	}
	digests := make([]llgoFileDigest, 0, len(inputs))
	for _, input := range inputs {
		compilerArgs := llgoFileCompilerArgs(ctx, input.compilerArgs, input.path)
		cacheKey := llgoFileHashKey{
			path:         input.path,
			compilerArgs: strings.Join(compilerArgs, "\x00"),
		}
		preprocessedHash, ok := ctx.llgoFileHashCache[cacheKey]
		if !ok {
			hash := sha256.New()
			cmd := ctx.compilerForSource(input.path)
			cmd.Stdout = hash
			preprocessArgs := append(slices.Clone(compilerArgs), "-E", input.path)
			if err := cmd.Compile(preprocessArgs...); err != nil {
				return nil, fmt.Errorf("preprocess file %q: %w", input.path, err)
			}
			preprocessedHash = hex.EncodeToString(hash.Sum(nil))
			ctx.llgoFileHashCache[cacheKey] = preprocessedHash
		}
		overlayHash := ""
		if content, ok := overlay[input.path]; ok {
			overlayHash = digestBytes(content)
		}
		digests = append(digests, llgoFileDigest{
			Path:             input.path,
			PreprocessedHash: preprocessedHash,
			OverlayHash:      overlayHash,
			CompilerArgs:     append([]string(nil), input.compilerArgs...),
		})
	}
	sort.Slice(digests, func(i, j int) bool {
		if digests[i].Path != digests[j].Path {
			return digests[i].Path < digests[j].Path
		}
		return strings.Join(digests[i].CompilerArgs, "\x00") < strings.Join(digests[j].CompilerArgs, "\x00")
	})
	return digests, nil
}

// digestFiles calculates digests for multiple files.
func digestFiles(paths []string) ([]fileDigest, error) {
	return digestFilesWithOverlay(paths, nil)
}

// digestFilesWithOverlay calculates digests for files, using overlay content when available.
func digestFilesWithOverlay(paths []string, overlay map[string][]byte) ([]fileDigest, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	digests := make([]fileDigest, 0, len(paths))
	for _, path := range paths {
		if content, ok := overlay[path]; ok {
			fd := fileDigest{
				Path:    path,
				Size:    int64(len(content)),
				ModTime: 0,
			}
			fd.OverlayHash = digestBytes(content)
			digests = append(digests, fd)
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat file %q: %w", path, err)
		}
		digests = append(digests, fileDigest{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
		})
	}

	sort.Slice(digests, func(i, j int) bool { return digests[i].Path < digests[j].Path })

	return digests, nil
}
