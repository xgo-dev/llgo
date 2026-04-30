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
	"sort"
	"strings"
	"unsafe"

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
	GoVersion        string           `yaml:"GO_VERSION,omitempty"`
	LlgoVersion      string           `yaml:"LLGO_VERSION,omitempty"`
	LlgoCompilerHash string           `yaml:"LLGO_COMPILER_HASH,omitempty"`
	LlvmTriple       string           `yaml:"LLVM_TRIPLE,omitempty"`
	LlvmVersion      string           `yaml:"LLVM_VERSION,omitempty"`
	Vars             orderedStringMap `yaml:"VARS,omitempty"`
}

func (s *envSection) empty() bool {
	return s.Goos == "" && s.Goarch == "" && s.LlvmTriple == "" && s.LlgoVersion == "" && s.LlgoCompilerHash == "" && s.GoVersion == "" && s.LlvmVersion == "" && len(s.Vars) == 0
}

type commonSection struct {
	AbiMode    string       `yaml:"ABI_MODE,omitempty"`
	BuildTags  []string     `yaml:"BUILD_TAGS,omitempty"`
	Target     string       `yaml:"TARGET,omitempty"`
	TargetABI  string       `yaml:"TARGET_ABI,omitempty"`
	CC         string       `yaml:"CC,omitempty"`
	CCFlags    []string     `yaml:"CCFLAGS,omitempty"`
	CFlags     []string     `yaml:"CFLAGS,omitempty"`
	LDFlags    []string     `yaml:"LDFLAGS,omitempty"`
	Linker     string       `yaml:"LINKER,omitempty"`
	ExtraFiles []fileDigest `yaml:"EXTRA_FILES,omitempty"`
}

func (s *commonSection) empty() bool {
	return s.AbiMode == "" && len(s.BuildTags) == 0 && s.Target == "" && s.TargetABI == "" &&
		s.CC == "" && len(s.CCFlags) == 0 && len(s.CFlags) == 0 && len(s.LDFlags) == 0 && s.Linker == "" && len(s.ExtraFiles) == 0
}

type packageSection struct {
	PkgPath     string           `yaml:"pkg_path,omitempty"`
	PkgID       string           `yaml:"pkg_id,omitempty"`
	GoFiles     []fileDigest     `yaml:"go_files,omitempty"`
	AltGoFiles  []fileDigest     `yaml:"alt_go_files,omitempty"`
	OtherFiles  []fileDigest     `yaml:"other_files,omitempty"`
	RewriteVars orderedStringMap `yaml:"rewrite_vars,omitempty"`
}

func (s *packageSection) empty() bool {
	return s.PkgPath == "" && s.PkgID == "" && len(s.GoFiles) == 0 && len(s.AltGoFiles) == 0 && len(s.OtherFiles) == 0 && len(s.RewriteVars) == 0
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
	return fingerprintManifest(m.Build())
}

func fingerprintManifest(content string) string {
	hash := sha256.Sum256(readOnlyStringBytes(content))
	return hex.EncodeToString(hash[:])
}

func readOnlyStringBytes(s string) []byte {
	if s == "" {
		return nil
	}
	// sha256.Sum256 only reads its input. Avoid copying large manifest strings
	// solely to satisfy the []byte API.
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func readOnlyBytesString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// The caller must not mutate b after this conversion. This is intended for
	// freshly allocated read buffers whose contents become immutable manifest text.
	return unsafe.String(unsafe.SliceData(b), len(b))
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
	return readOnlyBytesString(out), err
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

	if len(digests) > 1 {
		sort.Slice(digests, func(i, j int) bool { return digests[i].Path < digests[j].Path })
	}

	return digests, nil
}
