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
	"path/filepath"
	"runtime"
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
	DispatchInlineBudget bool `yaml:"DISPATCH_INLINE_BUDGET,omitempty"`

	EscapeAnalysisVersion int `yaml:"GO_ESCAPE_VERSION,omitempty"`

	BuildTags            []string     `yaml:"BUILD_TAGS,omitempty"`
	Target               string       `yaml:"TARGET,omitempty"`
	TargetABI            string       `yaml:"TARGET_ABI,omitempty"`
	WasmProfile          string       `yaml:"WASM_PROFILE,omitempty"`
	WasmProvider         string       `yaml:"WASM_PROVIDER,omitempty"`
	WasmReflectBridges   bool         `yaml:"WASM_REFLECT_BRIDGES,omitempty"`
	WasmFuncInfoEntries  bool         `yaml:"WASM_FUNCINFO_ENTRIES,omitempty"`
	WasmRuntimeFree      bool         `yaml:"WASM_RUNTIME_FREE,omitempty"`
	PlatformABI          string       `yaml:"PLATFORM_ABI,omitempty"`
	ObjectFormat         string       `yaml:"OBJECT_FORMAT,omitempty"`
	DriverFlavor         string       `yaml:"DRIVER_FLAVOR,omitempty"`
	LinkerFlavor         string       `yaml:"LINKER_FLAVOR,omitempty"`
	TargetTriple         string       `yaml:"TARGET_TRIPLE,omitempty"`
	CRTFlavor            string       `yaml:"CRT_FLAVOR,omitempty"`
	CXXRuntime           string       `yaml:"CXX_RUNTIME,omitempty"`
	SDKVersion           string       `yaml:"SDK_VERSION,omitempty"`
	CRTVersion           string       `yaml:"CRT_VERSION,omitempty"`
	ToolsetVersion       string       `yaml:"TOOLSET_VERSION,omitempty"`
	GoGlobalDCE          bool         `yaml:"GO_GLOBAL_DCE,omitempty"`
	DeadcodeDrop         bool         `yaml:"DEADCODE_DROP,omitempty"`
	EnableLTOPlugin      bool         `yaml:"ENABLE_LTO_PLUGIN,omitempty"`
	EmitDWARF            bool         `yaml:"EMIT_DWARF,omitempty"`
	EmitCodeView         bool         `yaml:"EMIT_CODEVIEW,omitempty"`
	PCLNMode             string       `yaml:"PCLN_MODE,omitempty"`
	DisableBoundsChecks  bool         `yaml:"DISABLE_BOUNDS_CHECKS,omitempty"`
	SaturatingFloatToInt bool         `yaml:"SATURATING_FLOAT_TO_INT,omitempty"`
	LocalContext         bool         `yaml:"LOCAL_CONTEXT,omitempty"`
	CC                   string       `yaml:"CC,omitempty"`
	CCArgs               []string     `yaml:"CC_ARGS,omitempty"`
	CCIdentity           string       `yaml:"CC_IDENTITY,omitempty"`
	CXX                  string       `yaml:"CXX,omitempty"`
	CXXArgs              []string     `yaml:"CXX_ARGS,omitempty"`
	CXXIdentity          string       `yaml:"CXX_IDENTITY,omitempty"`
	CCFlags              []string     `yaml:"CCFLAGS,omitempty"`
	CFlags               []string     `yaml:"CFLAGS,omitempty"`
	LDFlags              []string     `yaml:"LDFLAGS,omitempty"`
	Linker               string       `yaml:"LINKER,omitempty"`
	LinkerArgs           []string     `yaml:"LINKER_ARGS,omitempty"`
	LinkerIdentity       string       `yaml:"LINKER_IDENTITY,omitempty"`
	ExtraFiles           []fileDigest `yaml:"EXTRA_FILES,omitempty"`
}

func (s *commonSection) empty() bool {
	return !s.DispatchInlineBudget && s.EscapeAnalysisVersion == 0 && len(s.BuildTags) == 0 && s.Target == "" && s.TargetABI == "" && s.WasmProfile == "" && s.WasmProvider == "" &&
		!s.WasmReflectBridges && !s.WasmFuncInfoEntries && !s.WasmRuntimeFree && s.PlatformABI == "" && s.ObjectFormat == "" && s.DriverFlavor == "" && s.LinkerFlavor == "" &&
		s.TargetTriple == "" && s.CRTFlavor == "" && s.CXXRuntime == "" &&
		s.SDKVersion == "" && s.CRTVersion == "" && s.ToolsetVersion == "" &&
		!s.GoGlobalDCE && !s.DeadcodeDrop && !s.EnableLTOPlugin && !s.EmitDWARF && !s.EmitCodeView && s.PCLNMode == "" &&
		!s.DisableBoundsChecks && !s.SaturatingFloatToInt && !s.LocalContext &&
		s.CC == "" && len(s.CCArgs) == 0 && s.CCIdentity == "" && s.CXX == "" &&
		len(s.CXXArgs) == 0 && s.CXXIdentity == "" && len(s.CCFlags) == 0 &&
		len(s.CFlags) == 0 && len(s.LDFlags) == 0 && s.Linker == "" &&
		len(s.LinkerArgs) == 0 && s.LinkerIdentity == "" && len(s.ExtraFiles) == 0
}

type packageSection struct {
	GoVersion        string            `yaml:"go_language_version,omitempty"`
	FrontendInputs   string            `yaml:"frontend_inputs,omitempty"`
	DeadInits        []string          `yaml:"unused_global_inits,omitempty"`
	PthreadStackSize int64             `yaml:"pthread_stack_size,omitempty"`
	PkgPath          string            `yaml:"pkg_path,omitempty"`
	PkgID            string            `yaml:"pkg_id,omitempty"`
	GoFiles          []fileDigest      `yaml:"go_files,omitempty"`
	AltGoFiles       []fileDigest      `yaml:"alt_go_files,omitempty"`
	EmbedFiles       []embedFileDigest `yaml:"embed_files,omitempty"`
	AltEmbedFiles    []embedFileDigest `yaml:"alt_embed_files,omitempty"`
	OtherFiles       []fileDigest      `yaml:"other_files,omitempty"`
	LLGoFiles        []llgoFileDigest  `yaml:"llgo_files,omitempty"`
	RewriteVars      orderedStringMap  `yaml:"rewrite_vars,omitempty"`
}

func (s *packageSection) empty() bool {
	return s.GoVersion == "" && s.FrontendInputs == "" && len(s.DeadInits) == 0 && s.PkgPath == "" && s.PkgID == "" && len(s.GoFiles) == 0 && len(s.AltGoFiles) == 0 &&
		len(s.EmbedFiles) == 0 && len(s.AltEmbedFiles) == 0 &&
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

// embedFileDigest records the path and SHA-256 hash of an embedded resource.
// Content hashes detect edits preserving the size and timestamp used by fileDigest.
type embedFileDigest struct {
	Path        string `yaml:"path"`
	ContentHash string `yaml:"content_hash"`
}

// digestEmbedFiles hashes embedded file contents so edits preserving size and
// timestamps still invalidate the package cache.
func digestEmbedFiles(paths []string) ([]embedFileDigest, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	digests := make([]embedFileDigest, 0, len(paths))
	for _, path := range paths {
		hash, err := digestFile(path)
		if err != nil {
			return nil, fmt.Errorf("read embedded file %q: %w", path, err)
		}
		digests = append(digests, embedFileDigest{Path: path, ContentHash: hash})
	}
	sort.Slice(digests, func(i, j int) bool { return digests[i].Path < digests[j].Path })
	return digests, nil
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

// depfilePreprocessEnvVars lists environment variables that influence C/C++
// preprocessor include-path resolution, SDK/sysroot selection, or expansion.
var depfilePreprocessEnvVars = []string{
	"CPATH",
	"C_INCLUDE_PATH",
	"CPLUS_INCLUDE_PATH",
	"OBJC_INCLUDE_PATH",
	"OBJCPLUS_INCLUDE_PATH",
	"INCLUDE",
	"SDKROOT",
	"SYSROOT",
	"MACOSX_DEPLOYMENT_TARGET",
	"IPHONEOS_DEPLOYMENT_TARGET",
	"COMPILER_PATH",
	"GCC_EXEC_PREFIX",
	"CC",
	"CXX",
	"CCFLAGS",
	"CFLAGS",
	"CXXFLAGS",
	"CPPFLAGS",
	"EMCC_CFLAGS",
	"SOURCE_DATE_EPOCH",
}

var depfileIncludeSearchEnvVars = []string{
	"CPATH",
	"C_INCLUDE_PATH",
	"CPLUS_INCLUDE_PATH",
	"OBJC_INCLUDE_PATH",
	"OBJCPLUS_INCLUDE_PATH",
	"INCLUDE",
}

func lookupPreprocessEnv(environ []string, name string) string {
	if len(environ) == 0 {
		return os.Getenv(name)
	}
	return (commandEnv{environ: environ}).lookup(name)
}

func depfileEnvIdentity(ctx *context, workDir string, environ []string, effectiveArgs []string) string {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	var b strings.Builder
	b.WriteString(ctx.targetTriple())
	b.WriteByte('\x00')
	b.WriteString(ctx.llvmVersion)
	b.WriteByte('\x00')
	b.WriteString(ctx.crossCompile.CC)
	b.WriteByte('\x00')
	b.WriteString(ctx.crossCompile.CXX)
	b.WriteByte('\x00')
	b.WriteString(strings.Join(effectiveArgs, "\x1f"))
	b.WriteByte('\x00')
	b.WriteString(workDir)
	for _, name := range depfilePreprocessEnvVars {
		val := lookupPreprocessEnv(environ, name)
		b.WriteByte('\x00')
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(val)
	}
	return b.String()
}

func resolveSearchDir(workDir, dir string) string {
	if dir == "" {
		dir = workDir
	}
	if dir == "" {
		return ""
	}
	if !filepath.IsAbs(dir) && workDir != "" {
		dir = filepath.Join(workDir, dir)
	}
	return filepath.Clean(dir)
}

// collectIncludeSearchDirs extracts every directory that clang searches when
// resolving #include directives for inputPath, from the source directory,
// effective compiler flags, and preprocessor environment variables.
func collectIncludeSearchDirs(inputPath, workDir string, environ, effectiveArgs []string) []string {
	var dirs []string
	seen := make(map[string]bool)
	addDir := func(d string) {
		d = resolveSearchDir(workDir, d)
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}
	if inputPath != "" {
		srcPath := inputPath
		if !filepath.IsAbs(srcPath) && workDir != "" {
			srcPath = filepath.Join(workDir, srcPath)
		}
		addDir(filepath.Dir(srcPath))
	}
	twoArgFlags := map[string]bool{
		"-I": true, "-isystem": true, "-iquote": true, "-idirafter": true,
		"-F": true, "-iframework": true,
	}
	joinedPrefixes := []string{
		"--include-directory=",
		"-isystem", "-iquote", "-idirafter", "-iframework",
		"-I", "-F",
	}
	for i := 0; i < len(effectiveArgs); i++ {
		arg := effectiveArgs[i]
		if twoArgFlags[arg] {
			if i+1 < len(effectiveArgs) {
				i++
				addDir(effectiveArgs[i])
			}
			continue
		}
		for _, pfx := range joinedPrefixes {
			if strings.HasPrefix(arg, pfx) && len(arg) > len(pfx) {
				addDir(arg[len(pfx):])
				break
			}
		}
	}
	sep := string(os.PathListSeparator)
	for _, envName := range depfileIncludeSearchEnvVars {
		val := lookupPreprocessEnv(environ, envName)
		if val == "" {
			continue
		}
		for _, seg := range strings.Split(val, sep) {
			// Per GCC/Clang rules, an empty segment in CPATH or *_INCLUDE_PATH
			// searches the current working directory.
			if seg == "" && envName == "INCLUDE" {
				continue
			}
			addDir(seg)
		}
	}
	return dirs
}

// extractIncludeTargets scans C/C++/ObjC preprocessor directives in content for
// quoted ("...") and angled (<...>) header paths (#include, #import,
// #include_next, __has_include).
func extractIncludeTargets(content []byte) (quoted, angled []string) {
	s := string(content)
	addTarget := func(target string, isQuoted bool) {
		if target == "" {
			return
		}
		if isQuoted {
			quoted = append(quoted, target)
		} else {
			angled = append(angled, target)
		}
	}
	for len(s) > 0 {
		var line string
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			line = s[:nl]
			s = s[nl+1:]
		} else {
			line = s
			s = ""
		}
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) == 0 || trimmed[0] != '#' {
			continue
		}
		rest := strings.TrimLeft(trimmed[1:], " \t")
		for _, kw := range []string{"include_next", "include", "import"} {
			if strings.HasPrefix(rest, kw) {
				after := rest[len(kw):]
				if len(after) > 0 && (after[0] == ' ' || after[0] == '\t' || after[0] == '"' || after[0] == '<') {
					after = strings.TrimLeft(after, " \t")
					target, isQuoted := parseIncludeToken(after)
					addTarget(target, isQuoted)
				}
				break
			}
		}
		for _, fn := range []string{"__has_include_next(", "__has_include("} {
			cur := rest
			for {
				idx := strings.Index(cur, fn)
				if idx < 0 {
					break
				}
				cur = strings.TrimLeft(cur[idx+len(fn):], " \t")
				target, isQuoted := parseIncludeToken(cur)
				addTarget(target, isQuoted)
			}
		}
	}
	return quoted, angled
}

func parseIncludeToken(s string) (target string, isQuoted bool) {
	if len(s) < 3 {
		return "", false
	}
	var closeCh byte
	switch s[0] {
	case '"':
		closeCh = '"'
		isQuoted = true
	case '<':
		closeCh = '>'
	default:
		return "", false
	}
	end := strings.IndexByte(s[1:], closeCh)
	if end <= 0 {
		return "", false
	}
	return s[1 : 1+end], isQuoted
}

// collectWatchedDepDirs combines explicit include search directories with the
// containing directories of all resolved dependencies so transitive quoted
// includes (which prioritize the including header's directory) are watched.
func collectWatchedDepDirs(searchDirs, deps []string) []string {
	seen := make(map[string]bool, len(searchDirs)+len(deps))
	dirs := make([]string, 0, len(searchDirs)+len(deps))
	addDir := func(d string) {
		if d == "" {
			return
		}
		d = filepath.Clean(d)
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	for _, sd := range searchDirs {
		addDir(sd)
	}
	for _, dep := range deps {
		if dep != "" {
			addDir(filepath.Dir(dep))
		}
	}
	return dirs
}

// collectShadowingMissingPaths records non-existent candidate header paths across
// searchDirs and including headers' directories that, if created later, could
// shadow a previously resolved header or change __has_include evaluation without
// changing environment variables.
func collectShadowingMissingPaths(searchDirs, deps []string, depContents map[string][]byte) []string {
	watchedDirs := collectWatchedDepDirs(searchDirs, deps)
	if len(watchedDirs) == 0 {
		return nil
	}
	depSet := make(map[string]bool, len(deps))
	depDirSeen := make(map[string]bool, len(deps))
	var depDirs []string
	for _, d := range deps {
		if d == "" {
			continue
		}
		clean := filepath.Clean(d)
		depSet[clean] = true
		dd := filepath.Dir(clean)
		if !depDirSeen[dd] {
			depDirSeen[dd] = true
			depDirs = append(depDirs, dd)
		}
	}
	// Resolve symlinks on watchedDirs once so macOS /var -> /private/var matches.
	evalWatchedDirs := make([]string, len(watchedDirs))
	for i, sd := range watchedDirs {
		if real, err := filepath.EvalSymlinks(sd); err == nil {
			evalWatchedDirs[i] = filepath.Clean(real)
		} else {
			evalWatchedDirs[i] = sd
		}
	}

	missingSeen := make(map[string]bool)
	var missing []string
	addCandidate := func(cand string) {
		if cand == "" {
			return
		}
		cand = filepath.Clean(cand)
		if depSet[cand] || missingSeen[cand] {
			return
		}
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			missingSeen[cand] = true
			missing = append(missing, cand)
		}
	}

	allRelSeen := make(map[string]bool)
	var allRels []string
	quotedRelSeen := make(map[string]bool)
	var quotedRels []string
	addRel := func(rel string, alsoQuoted bool) {
		rel = strings.TrimSpace(rel)
		if rel == "" || filepath.IsAbs(rel) {
			return
		}
		rel = filepath.Clean(rel)
		if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			return
		}
		if !allRelSeen[rel] {
			allRelSeen[rel] = true
			allRels = append(allRels, rel)
		}
		if alsoQuoted && !quotedRelSeen[rel] {
			quotedRelSeen[rel] = true
			quotedRels = append(quotedRels, rel)
		}
	}

	for depPath, content := range depContents {
		depDir := filepath.Dir(filepath.Clean(depPath))
		quoted, angled := extractIncludeTargets(content)
		for _, q := range quoted {
			// Quoted includes search the including file's own directory first,
			// including relative "../..." paths.
			if q != "" && !filepath.IsAbs(q) {
				addCandidate(filepath.Join(depDir, q))
			}
			addRel(q, true)
		}
		for _, a := range angled {
			addRel(a, false)
		}
	}
	for _, dep := range deps {
		if dep == "" {
			continue
		}
		cleanDep := filepath.Clean(dep)
		realDep := cleanDep
		if r, err := filepath.EvalSymlinks(cleanDep); err == nil {
			realDep = filepath.Clean(r)
		}
		for i, sd := range watchedDirs {
			for _, pair := range [][2]string{{sd, cleanDep}, {evalWatchedDirs[i], realDep}} {
				rel, err := filepath.Rel(pair[0], pair[1])
				if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
					addRel(rel, true)
				}
			}
		}
	}

	for _, sd := range searchDirs {
		for _, rel := range allRels {
			addCandidate(filepath.Join(sd, rel))
		}
	}
	for _, dd := range depDirs {
		for _, rel := range quotedRels {
			addCandidate(filepath.Join(dd, rel))
		}
	}
	sort.Strings(missing)
	return missing
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
		cmd := ctx.compilerForSource(input.path)
		workDir := cmd.Dir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		effectiveArgs := cmd.CompileArguments(compilerArgs...)
		envID := depfileEnvIdentity(ctx, workDir, cmd.Env, effectiveArgs)
		cacheKey := llgoFileHashKey{
			path:         input.path,
			compilerArgs: strings.Join(effectiveArgs, "\x00"),
			envIdentity:  envID,
		}
		preprocessedHash, ok := ctx.llgoFileHashCache[cacheKey]
		if !ok {
			var depPath string
			depKey := ""
			if cacheEnabled() && overlay[input.path] == nil {
				depKey = cacheKey.envIdentity + "\x00" + cacheKey.path + "\x00" + cacheKey.compilerArgs
				depPath = ctx.ensureCacheManager().DepfilePath(depKey)
				if raw, err := os.ReadFile(depPath); err == nil {
					if df, err := unmarshalDepfile(raw); err == nil && df.Key == depKey && validateDepfile(df) {
						preprocessedHash = df.PreprocessedHash
						ok = true
					}
				}
			}
			if !ok {
				hash := sha256.New()
				cmd.Stdout = hash
				var tmpDep string
				if depPath != "" {
					if f, err := os.CreateTemp("", "llgo-dep-*.d"); err == nil {
						tmpDep = f.Name()
						_ = f.Close()
					}
				}
				if tmpDep != "" {
					// Use -MD (not -MMD) so system headers resolved via C_INCLUDE_PATH,
					// CPATH, -isystem, or SDK paths are tracked in the depfile cache.
					preprocessArgs := append(slices.Clone(compilerArgs), "-E", "-MD", "-MF", tmpDep, "-MT", "llgo", input.path)
					if err := cmd.Compile(preprocessArgs...); err != nil {
						// Fallback to plain -E if compiler rejects -MD/-MF for these flags.
						_ = os.Remove(tmpDep)
						tmpDep = ""
						hash.Reset()
						cmd = ctx.compilerForSource(input.path)
						cmd.Stdout = hash
						if err := cmd.Compile(append(slices.Clone(compilerArgs), "-E", input.path)...); err != nil {
							return nil, fmt.Errorf("preprocess file %q: %w", input.path, err)
						}
					}
				} else {
					preprocessArgs := append(slices.Clone(compilerArgs), "-E", input.path)
					if err := cmd.Compile(preprocessArgs...); err != nil {
						return nil, fmt.Errorf("preprocess file %q: %w", input.path, err)
					}
				}
				preprocessedHash = hex.EncodeToString(hash.Sum(nil))
				if tmpDep != "" {
					if depBytes, err := os.ReadFile(tmpDep); err == nil {
						deps := parseMakeDepfile(string(depBytes))
						deps = append(deps, input.path)
						for i, d := range deps {
							if d != "" && !filepath.IsAbs(d) && workDir != "" {
								deps[i] = filepath.Join(workDir, d)
							}
						}
						searchDirs := collectIncludeSearchDirs(input.path, workDir, cmd.Env, effectiveArgs)
						if stats, contents, err := statDepfilesWithContent(deps); err == nil && len(stats) > 0 {
							watchedDirs := collectWatchedDepDirs(searchDirs, deps)
							if dirStats, err := statDepfileDirs(watchedDirs); err == nil {
								missing := collectShadowingMissingPaths(searchDirs, deps, contents)
								if raw, err := marshalDepfile(&depfileRecord{
									Key:              depKey,
									PreprocessedHash: preprocessedHash,
									Files:            stats,
									Dirs:             dirStats,
									Missing:          missing,
								}); err == nil {
									_ = writeFileAtomicBytes(depPath, raw)
								}
							}
						}
					}
					_ = os.Remove(tmpDep)
				}
			}
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

// parseMakeDepfile parses a clang/gcc -MD/-MMD -MF Makefile dependency file and
// returns the list of prerequisite file paths.
func parseMakeDepfile(content string) []string {
	return parseMakeDepfileForOS(content, runtime.GOOS == "windows")
}

func parseMakeDepfileForOS(content string, isWindows bool) []string {
	content = strings.ReplaceAll(content, "\\\r\n", " ")
	content = strings.ReplaceAll(content, "\\\n", " ")
	colon := strings.IndexByte(content, ':')
	if colon < 0 {
		return nil
	}
	rest := content[colon+1:]
	var out []string
	var cur strings.Builder
	for i := 0; i < len(rest); i++ {
		ch := rest[i]
		if ch == '$' && i+1 < len(rest) && rest[i+1] == '$' {
			cur.WriteByte('$')
			i++
			continue
		}
		if ch == '\\' && i+1 < len(rest) {
			next := rest[i+1]
			if !isWindows || next == ' ' || next == '\t' || next == '#' {
				i++
				cur.WriteByte(next)
				continue
			}
			// On Windows, clang/gcc only escapes a run of backslashes when it
			// immediately precedes a space or '#'. Ordinary path separators
			// such as C:\Users\foo must be preserved verbatim.
			j := i
			for j < len(rest) && rest[j] == '\\' {
				j++
			}
			if j < len(rest) && (rest[j] == ' ' || rest[j] == '#') {
				n := j - i
				for k := 0; k < n/2; k++ {
					cur.WriteByte('\\')
				}
				if n%2 == 1 {
					cur.WriteByte(rest[j])
					i = j
				} else {
					i = j - 1
				}
				continue
			}
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if cur.Len() > 0 {
				p := filepath.Clean(cur.String())
				if p != "" && p != "." {
					out = append(out, p)
				}
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(ch)
	}
	if cur.Len() > 0 {
		p := filepath.Clean(cur.String())
		if p != "" && p != "." {
			out = append(out, p)
		}
	}
	return out
}
