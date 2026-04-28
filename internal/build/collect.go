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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

// collectFingerprint collects all inputs and generates fingerprint for a package.
func (c *context) collectFingerprint(pkg *aPackage) error {
	if pkg.Manifest != "" && pkg.Fingerprint != "" {
		return nil
	}
	if c.fingerprinting == nil {
		c.fingerprinting = make(map[string]bool)
	}
	if c.fingerprinting[pkg.ID] {
		return fmt.Errorf("fingerprint cycle detected for %s", pkg.ID)
	}
	c.fingerprinting[pkg.ID] = true
	defer delete(c.fingerprinting, pkg.ID)

	m := newManifestBuilder()

	// Env section
	c.collectEnvInputs(m)

	// Common section
	c.collectCommonInputs(m)

	// Package section
	if err := c.collectPackageInputs(m, pkg); err != nil {
		return err
	}

	// Dependency section
	if err := c.collectDependencyInputs(m, pkg); err != nil {
		return err
	}

	pkg.Manifest = m.Build()
	pkg.Fingerprint = fingerprintManifest(pkg.Manifest)
	return nil
}

// collectEnvInputs collects environment-related inputs.
func (c *context) collectEnvInputs(m *manifestBuilder) {
	m.env = c.envInputs()
}

func (c *context) envInputs() envSection {
	if c.envInputsReady {
		return c.envInputsCached
	}
	section := envSection{
		Goos:             c.buildConf.Goos,
		Goarch:           c.buildConf.Goarch,
		LlvmTriple:       c.crossCompile.LLVMTarget,
		LlgoVersion:      env.Version(),
		LlgoCompilerHash: c.buildConf.CompilerHash,
		GoVersion:        runtime.Version(),
		LlvmVersion:      c.getLLVMVersion(),
	}

	// Environment variables that affect build. Treat them as per-build inputs:
	// the build context is created for one invocation, so they only need to be
	// captured once and can be reused for every package manifest.
	envVars := []string{
		llgoDebug,
		llgoDbgSyms,
		llgoTrace,
		llgoOptimize,
		llgoWasmRuntime,
		llgoWasiThreads,
		llgoStdioNobuf,
		llgoFullRpath,
		llgoSSASanity,
	}
	for _, envVar := range envVars {
		if v := os.Getenv(envVar); v != "" {
			section.Vars = section.Vars.Add(envVar, v)
		}
	}
	c.envInputsCached = section
	c.envInputsReady = true
	return section
}

// collectCommonInputs collects common build configuration inputs.
func (c *context) collectCommonInputs(m *manifestBuilder) {
	m.common.AbiMode = fmt.Sprintf("%d", c.buildConf.AbiMode)
	if buildTags := c.buildTags(); len(buildTags) > 0 {
		m.common.BuildTags = buildTags
	}
	m.common.Target = c.buildConf.Target
	m.common.TargetABI = c.crossCompile.TargetABI

	// Compiler configuration
	if c.crossCompile.CC != "" {
		m.common.CC = c.crossCompile.CC
	}
	if len(c.crossCompile.CCFLAGS) > 0 {
		m.common.CCFlags = append([]string(nil), c.crossCompile.CCFLAGS...)
	}
	if len(c.crossCompile.CFLAGS) > 0 {
		m.common.CFlags = append([]string(nil), c.crossCompile.CFLAGS...)
	}
	if len(c.crossCompile.LDFLAGS) > 0 {
		m.common.LDFlags = append([]string(nil), c.crossCompile.LDFLAGS...)
	}
	if c.crossCompile.Linker != "" {
		m.common.Linker = c.crossCompile.Linker
	}

	// Extra files from target configuration
	if len(c.crossCompile.ExtraFiles) > 0 {
		extraList, err := digestFiles(c.crossCompile.ExtraFiles)
		if err == nil && len(extraList) > 0 {
			m.common.ExtraFiles = extraList
		}
	}
}

// collectPackageInputs collects package-specific inputs.
func (c *context) collectPackageInputs(m *manifestBuilder, pkg *aPackage) error {
	p := pkg.Package

	m.pkg.PkgPath = p.PkgPath
	m.pkg.PkgID = p.ID

	// Go source files
	goFilesList, err := digestFilesWithOverlay(p.GoFiles, c.conf.Overlay)
	if err != nil {
		return fmt.Errorf("digest go files: %w", err)
	}
	m.pkg.GoFiles = goFilesList

	// Alt package files (if any)
	if pkg.AltPkg != nil {
		altList, err := digestFilesWithOverlay(pkg.AltPkg.GoFiles, c.conf.Overlay)
		if err != nil {
			return fmt.Errorf("digest alt go files: %w", err)
		}
		m.pkg.AltGoFiles = altList
	}

	// Other files (C, assembly, etc.)
	otherFiles := append([]string{}, p.OtherFiles...)
	sfiles, err := pkgSFiles(c, p)
	if err != nil {
		return fmt.Errorf("list sfiles: %w", err)
	}
	otherFiles = append(otherFiles, sfiles...)
	if len(otherFiles) > 0 {
		otherList, err := digestFilesWithOverlay(otherFiles, c.conf.Overlay)
		if err != nil {
			return fmt.Errorf("digest other files: %w", err)
		}
		m.pkg.OtherFiles = otherList
	}

	// Rewrite vars
	if len(pkg.rewriteVars) > 0 {
		rewrites := make(map[string]string, len(pkg.rewriteVars))
		for k, v := range pkg.rewriteVars {
			rewrites[k] = v
		}
		m.pkg.RewriteVars = m.pkg.RewriteVars.AddMap(rewrites)
	}

	// Add metadata fields if available (for cache saving)
	// (LINK_ARGS/NEED_RT/NEED_PY_INIT are appended later in saveToCache)

	return nil
}

// collectDependencyInputs adds dependency fingerprints/versions into manifest.
func (c *context) collectDependencyInputs(m *manifestBuilder, pkg *aPackage) error {
	if len(pkg.Imports) == 0 {
		return nil
	}

	deps := make([]*packages.Package, 0, len(pkg.Imports))
	for _, dep := range pkg.Imports {
		if dep == nil || dep.ID == pkg.ID {
			continue
		}
		deps = append(deps, dep)
	}

	sort.Slice(deps, func(i, j int) bool { return deps[i].ID < deps[j].ID })

	for _, dep := range deps {
		depEntry, err := c.dependencyFingerprint(dep)
		if err != nil {
			return err
		}
		m.deps = append(m.deps, depEntry)
	}

	return nil
}

func (c *context) dependencyFingerprint(dep *packages.Package) (depEntry, error) {
	entry := depEntry{ID: dep.ID}
	if v := moduleVersion(dep.Module); v != "" {
		entry.Version = v
		return entry, nil
	}

	if c.pkgByID != nil {
		if aDep, ok := c.pkgByID[dep.ID]; ok {
			if aDep.Fingerprint == "" {
				if err := c.collectFingerprint(aDep); err != nil {
					return entry, fmt.Errorf("collect fingerprint for %s: %w", dep.ID, err)
				}
			}
			entry.Fingerprint = aDep.Fingerprint
			return entry, nil
		}
	}

	temp := &aPackage{Package: dep}
	if err := c.collectFingerprint(temp); err != nil {
		return entry, fmt.Errorf("collect fingerprint for %s: %w", dep.ID, err)
	}
	entry.Fingerprint = temp.Fingerprint
	return entry, nil
}

func (c *context) buildTags() []string {
	tags := c.buildConf.Tags
	if tags == "" {
		return nil
	}
	if c.buildTagsRaw == tags && c.buildTagsCached != nil {
		return c.buildTagsCached
	}
	buildTags := strings.Split(tags, ",")
	sort.Strings(buildTags)
	c.buildTagsRaw = tags
	c.buildTagsCached = buildTags
	return buildTags
}

func moduleVersion(mod *gopackages.Module) string {
	if mod == nil {
		return ""
	}
	if mod.Replace != nil {
		// replace to local path (Version empty) should not use version for fingerprint
		if mod.Replace.Version != "" {
			return mod.Replace.Version
		}
		return ""
	}
	return mod.Version
}

var (
	llvmVersionCache      sync.Map
	llvmCompilerPathCache sync.Map
	detectLLVMVersionFunc = detectLLVMVersion
)

// getLLVMVersion returns the cached LLVM version or detects it.
func (c *context) getLLVMVersion() string {
	if c.llvmVersion != "" {
		return c.llvmVersion
	}
	cc := c.crossCompile.CC
	if cc == "" {
		cc = "clang"
	}
	cacheKey := llvmVersionCacheKey(cc)
	if version, ok := llvmVersionCache.Load(cacheKey); ok {
		c.llvmVersion = version.(string)
		return c.llvmVersion
	}
	c.llvmVersion = detectLLVMVersionFunc(c)
	if c.llvmVersion != "" {
		actual, _ := llvmVersionCache.LoadOrStore(cacheKey, c.llvmVersion)
		c.llvmVersion = actual.(string)
	}
	return c.llvmVersion
}

func llvmVersionCacheKey(cc string) string {
	if filepath.IsAbs(cc) {
		return cc
	}
	pathKey := cc + "\x00" + os.Getenv("PATH")
	if cacheKey, ok := llvmCompilerPathCache.Load(pathKey); ok {
		return cacheKey.(string)
	}
	cacheKey := cc
	if path, err := exec.LookPath(cc); err == nil {
		cacheKey = path
	}
	actual, _ := llvmCompilerPathCache.LoadOrStore(pathKey, cacheKey)
	return actual.(string)
}

// detectLLVMVersion detects LLVM version from clang --version.
func detectLLVMVersion(ctx *context) string {
	// Get compiler path from cross compile config
	cc := ctx.crossCompile.CC
	if cc == "" {
		cc = "clang"
	}
	versionCmd := exec.Command(cc, "--version")
	output, err := versionCmd.Output()
	if err != nil {
		return ""
	}
	line := string(output)
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

// targetTriple returns the target triple for cache directory.
func (c *context) targetTriple() string {
	if c.targetTripleCached != "" {
		return c.targetTripleCached
	}
	c.targetTripleCached = targetTriple(
		c.buildConf.Goos,
		c.buildConf.Goarch,
		c.crossCompile.LLVMTarget,
		c.crossCompile.TargetABI,
	)
	return c.targetTripleCached
}

// targetTriple returns the target triple string for cache directory
func targetTriple(goos, goarch, llvmTarget, targetABI string) string {
	triple := llvmTarget
	if triple == "" {
		triple = goarch + "-" + goos
	}
	if targetABI != "" {
		triple = triple + "-" + targetABI
	}
	return triple
}

// ensureCacheManager creates cacheManager if not exists.
func (c *context) ensureCacheManager() *cacheManager {
	if c.cacheManager == nil {
		c.cacheManager = newCacheManager()
	}
	return c.cacheManager
}

// tryLoadFromCache attempts to load a package from cache.
// Returns true if cache hit, false otherwise.
func (c *context) tryLoadFromCache(pkg *aPackage) bool {
	if !cacheEnabled() {
		return false
	}

	// Skip cache when force rebuild is enabled
	if c.buildConf.ForceRebuild {
		return false
	}

	if pkg.Fingerprint == "" {
		return false
	}

	cm := c.ensureCacheManager()
	paths := cm.PackagePaths(c.targetTriple(), pkg.PkgPath, pkg.Fingerprint)

	// Check if archive file exists
	if _, err := os.Stat(paths.Archive); err != nil {
		return false
	}

	// Read metadata from manifest
	content, err := readManifest(paths.Manifest)
	if err != nil {
		return false
	}

	// Parse metadata from manifest [Package] section (INI format)
	meta, err := parseManifestMetadata(content)
	if err != nil {
		return false
	}

	// Use the .a archive directly for linking (no extraction needed)
	pkg.ArchiveFile = paths.Archive
	pkg.LinkArgs = meta.LinkArgs
	pkg.NeedRt = meta.NeedRt
	pkg.NeedPyInit = meta.NeedPyInit
	pkg.CacheHit = true

	return true
}

// parseManifestMetadata extracts metadata from manifest content.
// It supports the new YAML format and falls back to the legacy INI layout for
// backward compatibility with existing cache entries.
func parseManifestMetadata(content string) (*cacheArchiveMetadata, error) {
	meta := &cacheArchiveMetadata{}
	if metadataContent, ok := yamlMetadataSection(content); ok {
		if simpleMeta, ok := parseSimpleManifestMetadata(metadataContent); ok {
			return simpleMeta, nil
		}
		if data, err := decodeManifest(metadataContent); err == nil {
			if data.Metadata != nil {
				meta.LinkArgs = append([]string(nil), data.Metadata.LinkArgs...)
				meta.NeedRt = data.Metadata.NeedRt
				meta.NeedPyInit = data.Metadata.NeedPyInit
			}
			return meta, nil
		}
	} else if !strings.Contains(content, "[Package]\n") {
		return meta, nil
	}
	if data, err := decodeManifest(content); err == nil {
		if data.Metadata != nil {
			meta.LinkArgs = append([]string(nil), data.Metadata.LinkArgs...)
			meta.NeedRt = data.Metadata.NeedRt
			meta.NeedPyInit = data.Metadata.NeedPyInit
		}
		return meta, nil
	}

	return parseManifestMetadataLegacy(content, meta)
}

func parseSimpleManifestMetadata(content string) (*cacheArchiveMetadata, bool) {
	meta := &cacheArchiveMetadata{}
	inLinkArgs := false
	for start := 0; start <= len(content); {
		end := strings.IndexByte(content[start:], '\n')
		if end < 0 {
			end = len(content)
		} else {
			end += start
		}
		field := strings.TrimSpace(content[start:end])
		switch field {
		case "", "metadata:":
			// Skip.
		case "link_args:":
			inLinkArgs = true
			if meta.LinkArgs == nil {
				meta.LinkArgs = make([]string, 0, 4)
			}
		case "need_rt: true":
			inLinkArgs = false
			meta.NeedRt = true
		case "need_rt: false":
			inLinkArgs = false
			meta.NeedRt = false
		case "need_py_init: true":
			inLinkArgs = false
			meta.NeedPyInit = true
		case "need_py_init: false":
			inLinkArgs = false
			meta.NeedPyInit = false
		default:
			if !inLinkArgs || !strings.HasPrefix(field, "- ") {
				return nil, false
			}
			arg := strings.TrimSpace(field[2:])
			if !isSimpleMetadataScalar(arg) {
				return nil, false
			}
			meta.LinkArgs = append(meta.LinkArgs, arg)
		}
		if end == len(content) {
			break
		}
		start = end + 1
	}
	return meta, true
}

func isSimpleMetadataScalar(s string) bool {
	return s != "" && !strings.ContainsAny(s, "\"'{}[]&*!|>@`#\t\r\n")
}

func yamlMetadataSection(content string) (string, bool) {
	start := 0
	if !strings.HasPrefix(content, "metadata:") {
		idx := strings.Index(content, "\nmetadata:")
		if idx < 0 {
			return "", false
		}
		start = idx + 1
	}
	end := len(content)
	searchFrom := start + len("metadata:")
	for _, marker := range [...]string{"\nenv:", "\ncommon:", "\npackage:", "\ndeps:"} {
		if idx := strings.Index(content[searchFrom:], marker); idx >= 0 {
			pos := searchFrom + idx
			if pos+1 < end {
				end = pos + 1
			}
		}
	}
	return content[start:end], true
}

func parseManifestMetadataLegacy(content string, meta *cacheArchiveMetadata) (*cacheArchiveMetadata, error) {
	// Find Package section
	idx := strings.Index(content, "[Package]\n")
	if idx == -1 {
		return meta, nil
	}

	// Extract Package section content (until next section or end)
	pkgSection := content[idx+len("[Package]\n"):]
	if nextIdx := strings.Index(pkgSection, "\n["); nextIdx != -1 {
		pkgSection = pkgSection[:nextIdx]
	}

	// Parse key-value pairs in Package section
	lines := strings.Split(pkgSection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " = ", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		switch key {
		case "LINK_ARGS":
			if value != "" {
				meta.LinkArgs = strings.Fields(value)
			}
		case "NEED_RT":
			meta.NeedRt = value == "true"
		case "NEED_PY_INIT":
			meta.NeedPyInit = value == "true"
		}
	}

	return meta, nil
}

// cacheArchiveMetadata holds metadata about a cached archive.
type cacheArchiveMetadata struct {
	LinkArgs   []string
	NeedRt     bool
	NeedPyInit bool
}

// saveToCache saves a built package to cache.
func (c *context) saveToCache(pkg *aPackage) error {
	if !cacheEnabled() {
		return nil
	}

	if pkg.Fingerprint == "" || pkg.Manifest == "" {
		return nil
	}

	// Don't cache main packages
	if pkg.Name == "main" {
		return nil
	}

	cm := c.ensureCacheManager()
	paths := cm.PackagePaths(c.targetTriple(), pkg.PkgPath, pkg.Fingerprint)

	// Ensure directory exists
	if err := cm.EnsureDir(paths); err != nil {
		return err
	}

	// Publish the package archive directly at its cache path. Cache misses can
	// then link this archive immediately instead of first creating a temporary
	// archive and copying it into the cache.
	if pkg.ArchiveFile != "" {
		if pkg.ArchiveFile != paths.Archive {
			if err := copyFileAtomic(pkg.ArchiveFile, paths.Archive); err != nil {
				return err
			}
		}
	} else if len(pkg.ObjFiles) > 0 {
		if err := c.createArchiveFile(paths.Archive, pkg.ObjFiles); err != nil {
			return err
		}
	} else {
		return nil
	}
	pkg.ArchiveFile = paths.Archive
	pkg.ObjFiles = nil

	// Append metadata to existing manifest (pkg.Manifest was built in collectFingerprint).
	manifestContent := pkg.Manifest
	if manifestContent == "" {
		return fmt.Errorf("package %s missing manifest for fingerprint %s", pkg.PkgPath, pkg.Fingerprint)
	}

	meta := &manifestMetadata{
		LinkArgs:   append([]string(nil), pkg.LinkArgs...),
		NeedRt:     pkg.NeedRt,
		NeedPyInit: pkg.NeedPyInit,
	}

	manifestWithMeta := manifestContent
	if len(meta.LinkArgs) > 0 || meta.NeedRt || meta.NeedPyInit {
		var err error
		manifestWithMeta, err = appendManifestMetadata(manifestContent, meta)
		if err != nil {
			return err
		}
	}

	// Write manifest with metadata
	if err := writeManifest(paths.Manifest, manifestWithMeta); err != nil {
		return err
	}

	return nil
}

func appendManifestMetadata(manifestContent string, meta *manifestMetadata) (string, error) {
	if metaLen, ok := simpleManifestMetadataLen(meta); ok {
		return appendSimpleManifestMetadata(manifestContent, meta, metaLen), nil
	}
	metaContent, err := buildManifestYAML(manifestData{Metadata: meta})
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(manifestContent, "\n") {
		manifestContent += "\n"
	}
	if idx := strings.Index(manifestContent, "\ndeps:"); idx >= 0 {
		idx++
		return manifestContent[:idx] + metaContent + manifestContent[idx:], nil
	}
	return manifestContent + metaContent, nil
}

func appendSimpleManifestMetadata(manifestContent string, meta *manifestMetadata, metaLen int) string {
	missingNewline := !strings.HasSuffix(manifestContent, "\n")
	idx := strings.Index(manifestContent, "\ndeps:")
	if idx >= 0 {
		idx++
	}

	finalLen := len(manifestContent) + metaLen
	if missingNewline {
		finalLen++
	}
	var b strings.Builder
	b.Grow(finalLen)
	if idx >= 0 {
		b.WriteString(manifestContent[:idx])
		writeSimpleManifestMetadata(&b, meta)
		b.WriteString(manifestContent[idx:])
		if missingNewline {
			b.WriteByte('\n')
		}
		return b.String()
	}
	b.WriteString(manifestContent)
	if missingNewline {
		b.WriteByte('\n')
	}
	writeSimpleManifestMetadata(&b, meta)
	return b.String()
}

func buildSimpleManifestMetadata(meta *manifestMetadata) (string, bool) {
	metaLen, ok := simpleManifestMetadataLen(meta)
	if !ok {
		return "", false
	}
	var b strings.Builder
	b.Grow(metaLen)
	writeSimpleManifestMetadata(&b, meta)
	return b.String(), true
}

func simpleManifestMetadataLen(meta *manifestMetadata) (int, bool) {
	metaLen := len("metadata:\n")
	if len(meta.LinkArgs) > 0 {
		metaLen += len("    link_args:\n")
		for _, arg := range meta.LinkArgs {
			if !isSimpleMetadataScalar(arg) {
				return 0, false
			}
			metaLen += len("        - ") + len(arg) + 1
		}
	}
	if meta.NeedRt {
		metaLen += len("    need_rt: true\n")
	}
	if meta.NeedPyInit {
		metaLen += len("    need_py_init: true\n")
	}
	return metaLen, true
}

func writeSimpleManifestMetadata(b *strings.Builder, meta *manifestMetadata) {
	b.WriteString("metadata:\n")
	if len(meta.LinkArgs) > 0 {
		b.WriteString("    link_args:\n")
		for _, arg := range meta.LinkArgs {
			b.WriteString("        - ")
			b.WriteString(arg)
			b.WriteByte('\n')
		}
	}
	if meta.NeedRt {
		b.WriteString("    need_rt: true\n")
	}
	if meta.NeedPyInit {
		b.WriteString("    need_py_init: true\n")
	}
}

// copyFileAtomic copies src to dst using a temp file for atomicity.
func copyFileAtomic(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmp, in); err != nil {
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}

	cleanup = false
	return nil
}
