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
	"os"
	"sort"
	"strconv"
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
	return buildManifestYAMLFast(data), nil
}

func buildManifestYAMLFast(data manifestData) string {
	var b strings.Builder
	if data.Env != nil && !data.Env.empty() {
		b.WriteString("env:\n")
		writeStringField(&b, 4, "GOOS", data.Env.Goos)
		writeStringField(&b, 4, "GOARCH", data.Env.Goarch)
		writeStringField(&b, 4, "GO_VERSION", data.Env.GoVersion)
		writeStringField(&b, 4, "LLGO_VERSION", data.Env.LlgoVersion)
		writeStringField(&b, 4, "LLGO_COMPILER_HASH", data.Env.LlgoCompilerHash)
		writeStringField(&b, 4, "LLVM_TRIPLE", data.Env.LlvmTriple)
		writeStringField(&b, 4, "LLVM_VERSION", data.Env.LlvmVersion)
		writeStringMap(&b, 4, "VARS", data.Env.Vars)
	}
	if data.Common != nil && !data.Common.empty() {
		b.WriteString("common:\n")
		writeStringField(&b, 4, "ABI_MODE", data.Common.AbiMode)
		writeStringList(&b, 4, "BUILD_TAGS", data.Common.BuildTags)
		writeStringField(&b, 4, "TARGET", data.Common.Target)
		writeStringField(&b, 4, "TARGET_ABI", data.Common.TargetABI)
		writeStringField(&b, 4, "CC", data.Common.CC)
		writeStringList(&b, 4, "CCFLAGS", data.Common.CCFlags)
		writeStringList(&b, 4, "CFLAGS", data.Common.CFlags)
		writeStringList(&b, 4, "LDFLAGS", data.Common.LDFlags)
		writeStringField(&b, 4, "LINKER", data.Common.Linker)
		writeFileDigestList(&b, 4, "EXTRA_FILES", data.Common.ExtraFiles)
	}
	if data.Package != nil && !data.Package.empty() {
		b.WriteString("package:\n")
		writeStringField(&b, 4, "pkg_path", data.Package.PkgPath)
		writeStringField(&b, 4, "pkg_id", data.Package.PkgID)
		writeFileDigestList(&b, 4, "go_files", data.Package.GoFiles)
		writeFileDigestList(&b, 4, "alt_go_files", data.Package.AltGoFiles)
		writeFileDigestList(&b, 4, "other_files", data.Package.OtherFiles)
		writeStringMap(&b, 4, "rewrite_vars", data.Package.RewriteVars)
	}
	if data.Metadata != nil {
		b.WriteString("metadata:\n")
		writeStringList(&b, 4, "link_args", data.Metadata.LinkArgs)
		writeBoolField(&b, 4, "need_rt", data.Metadata.NeedRt)
		writeBoolField(&b, 4, "need_py_init", data.Metadata.NeedPyInit)
	}
	writeDepList(&b, data.Deps)
	return b.String()
}

func writeIndent(b *strings.Builder, n int) {
	for ; n > 0; n-- {
		b.WriteByte(' ')
	}
}

func writeYAMLString(b *strings.Builder, s string) {
	if isPlainYAMLString(s) {
		b.WriteString(s)
		return
	}
	b.WriteString(strconv.Quote(s))
}

func isPlainYAMLString(s string) bool {
	if s == "" || s == "true" || s == "false" || s == "null" || s == "~" {
		return false
	}
	hasNonDigit := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
			hasNonDigit = true
		case '0' <= c && c <= '9':
		case c == '/', c == '.', c == '_', c == '-', c == '+':
			hasNonDigit = true
		default:
			return false
		}
	}
	return hasNonDigit
}

func writeStringField(b *strings.Builder, indent int, key, val string) {
	if val == "" {
		return
	}
	writeIndent(b, indent)
	b.WriteString(key)
	b.WriteString(": ")
	writeYAMLString(b, val)
	b.WriteByte('\n')
}

func writeBoolField(b *strings.Builder, indent int, key string, val bool) {
	if !val {
		return
	}
	writeIndent(b, indent)
	b.WriteString(key)
	b.WriteString(": true\n")
}

func writeStringList(b *strings.Builder, indent int, key string, vals []string) {
	if len(vals) == 0 {
		return
	}
	writeIndent(b, indent)
	b.WriteString(key)
	b.WriteString(":\n")
	for _, val := range vals {
		writeIndent(b, indent+4)
		b.WriteString("- ")
		writeYAMLString(b, val)
		b.WriteByte('\n')
	}
}

func writeStringMap(b *strings.Builder, indent int, key string, vals orderedStringMap) {
	if len(vals) == 0 {
		return
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	writeIndent(b, indent)
	b.WriteString(key)
	b.WriteString(":\n")
	for _, k := range keys {
		writeIndent(b, indent+4)
		writeYAMLString(b, k)
		b.WriteString(": ")
		writeYAMLString(b, vals[k])
		b.WriteByte('\n')
	}
}

func writeFileDigestList(b *strings.Builder, indent int, key string, vals []fileDigest) {
	if len(vals) == 0 {
		return
	}
	writeIndent(b, indent)
	b.WriteString(key)
	b.WriteString(":\n")
	for _, val := range vals {
		writeIndent(b, indent+4)
		b.WriteString("- path: ")
		writeYAMLString(b, val.Path)
		b.WriteByte('\n')
		writeIndent(b, indent+6)
		b.WriteString("size: ")
		b.WriteString(strconv.FormatInt(val.Size, 10))
		b.WriteByte('\n')
		writeIndent(b, indent+6)
		b.WriteString("mtime: ")
		b.WriteString(strconv.FormatInt(val.ModTime, 10))
		b.WriteByte('\n')
		writeStringField(b, indent+6, "overlay_hash", val.OverlayHash)
	}
}

func writeDepList(b *strings.Builder, vals []depEntry) {
	if len(vals) == 0 {
		return
	}
	b.WriteString("deps:\n")
	for _, val := range vals {
		b.WriteString("    - id: ")
		writeYAMLString(b, val.ID)
		b.WriteByte('\n')
		writeStringField(b, 6, "version", val.Version)
		writeStringField(b, 6, "fingerprint", val.Fingerprint)
	}
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
