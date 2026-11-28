package build

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/xgo-dev/llgo/directive"
	llruntime "github.com/xgo-dev/llgo/runtime"
	"golang.org/x/tools/go/ast/astutil"
)

const sourcePatchCacheMagic = "LLGOSP01"

type sourcePatchFileStat struct {
	Path    string
	Size    int64
	ModTime int64
}

type sourcePatchCacheRecord struct {
	Dirs     []depfileDirStat
	Files    []sourcePatchFileStat
	Overlay  map[string][]byte
	PkgFiles map[string][]string
}

func sourcePatchCacheKey(runtimeDir, goroot string, ctx sourcePatchBuildContext) string {
	var b strings.Builder
	b.WriteString(sourcePatchCacheMagic)
	b.WriteByte('\x00')
	if exe, err := os.Executable(); err == nil {
		if fi, err := os.Stat(exe); err == nil {
			b.WriteString(exe)
			b.WriteByte(':')
			b.WriteString(strconv.FormatInt(fi.Size(), 10))
			b.WriteByte(':')
			b.WriteString(strconv.FormatInt(fi.ModTime().UnixNano(), 10))
		}
	}
	b.WriteByte('\x00')
	b.WriteString(runtimeDir)
	b.WriteByte('\x00')
	b.WriteString(goroot)
	b.WriteByte('\x00')
	b.WriteString(ctx.goos)
	b.WriteByte('\x00')
	b.WriteString(ctx.goarch)
	b.WriteByte('\x00')
	b.WriteString(ctx.goversion)
	b.WriteByte('\x00')
	b.WriteString(strings.Join(ctx.toolTags, ","))
	b.WriteByte('\x00')
	b.WriteString(strings.Join(ctx.buildFlags, "\x1f"))
	b.WriteByte('\x00')
	b.WriteString(strings.Join(ctx.targetTags, ","))
	b.WriteByte('\x00')
	if ctx.cgoEnabled != nil {
		if *ctx.cgoEnabled {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	} else {
		b.WriteByte('-')
	}
	b.WriteByte('\x00')
	b.WriteString(strings.Join(llruntime.SourcePatchPkgPaths(), ","))
	return digestBytes([]byte(b.String()))
}

func marshalSourcePatchCache(rec *sourcePatchCacheRecord) []byte {
	var buf bytes.Buffer
	buf.WriteString(sourcePatchCacheMagic)
	writeBinUvarint(&buf, uint64(len(rec.Dirs)))
	for _, d := range rec.Dirs {
		writeBinStr(&buf, d.Path)
		writeBinBool(&buf, d.Exists)
		writeBinStr(&buf, d.Hash)
	}
	writeBinUvarint(&buf, uint64(len(rec.Files)))
	for _, f := range rec.Files {
		writeBinStr(&buf, f.Path)
		writeBinVarint(&buf, f.Size)
		writeBinVarint(&buf, f.ModTime)
	}
	ovKeys := make([]string, 0, len(rec.Overlay))
	for k := range rec.Overlay {
		ovKeys = append(ovKeys, k)
	}
	sort.Strings(ovKeys)
	writeBinUvarint(&buf, uint64(len(ovKeys)))
	for _, k := range ovKeys {
		writeBinStr(&buf, k)
		writeBinBytes(&buf, rec.Overlay[k])
	}
	pfKeys := make([]string, 0, len(rec.PkgFiles))
	for k := range rec.PkgFiles {
		pfKeys = append(pfKeys, k)
	}
	sort.Strings(pfKeys)
	writeBinUvarint(&buf, uint64(len(pfKeys)))
	for _, k := range pfKeys {
		writeBinStr(&buf, k)
		writeBinStrs(&buf, rec.PkgFiles[k])
	}
	return buf.Bytes()
}

func unmarshalSourcePatchCache(raw []byte) (*sourcePatchCacheRecord, error) {
	if len(raw) < len(sourcePatchCacheMagic) || string(raw[:len(sourcePatchCacheMagic)]) != sourcePatchCacheMagic {
		return nil, errors.New("invalid source patch cache magic")
	}
	r := &binReader{data: raw, pos: len(sourcePatchCacheMagic)}
	nd, err := r.readUvarint()
	if err != nil || nd > uint64(len(raw)-r.pos) {
		return nil, errors.New("corrupt source patch dirs")
	}
	rec := &sourcePatchCacheRecord{}
	if nd > 0 {
		rec.Dirs = make([]depfileDirStat, nd)
		for i := uint64(0); i < nd; i++ {
			if rec.Dirs[i].Path, err = r.readStr(); err != nil {
				return nil, err
			}
			if rec.Dirs[i].Exists, err = r.readBool(); err != nil {
				return nil, err
			}
			if rec.Dirs[i].Hash, err = r.readStr(); err != nil {
				return nil, err
			}
		}
	}
	nf, err := r.readUvarint()
	if err != nil || nf > uint64(len(raw)-r.pos) {
		return nil, errors.New("corrupt source patch files")
	}
	if nf > 0 {
		rec.Files = make([]sourcePatchFileStat, nf)
		for i := uint64(0); i < nf; i++ {
			if rec.Files[i].Path, err = r.readStr(); err != nil {
				return nil, err
			}
			if rec.Files[i].Size, err = r.readVarint(); err != nil {
				return nil, err
			}
			if rec.Files[i].ModTime, err = r.readVarint(); err != nil {
				return nil, err
			}
		}
	}
	no, err := r.readUvarint()
	if err != nil || no > uint64(len(raw)-r.pos) {
		return nil, errors.New("corrupt source patch overlay")
	}
	if no > 0 {
		rec.Overlay = make(map[string][]byte, no)
		for i := uint64(0); i < no; i++ {
			k, err := r.readStr()
			if err != nil {
				return nil, err
			}
			v, err := r.readBytes()
			if err != nil {
				return nil, err
			}
			if v == nil {
				v = []byte{}
			}
			rec.Overlay[k] = v
		}
	}
	np, err := r.readUvarint()
	if err != nil || np > uint64(len(raw)-r.pos) {
		return nil, errors.New("corrupt source patch pkg files")
	}
	if np > 0 {
		rec.PkgFiles = make(map[string][]string, np)
		for i := uint64(0); i < np; i++ {
			k, err := r.readStr()
			if err != nil {
				return nil, err
			}
			v, err := r.readStrs()
			if err != nil {
				return nil, err
			}
			rec.PkgFiles[k] = v
		}
	}
	if r.pos != len(raw) {
		return nil, errors.New("trailing bytes in source patch cache")
	}
	return rec, nil
}

func (r *binReader) readStrs() ([]string, error) {
	return readBinStrs(r)
}

func validateSourcePatchCache(rec *sourcePatchCacheRecord) bool {
	if rec == nil {
		return false
	}
	for _, d := range rec.Dirs {
		entries, err := os.ReadDir(d.Path)
		if !d.Exists {
			if !os.IsNotExist(err) {
				return false
			}
			continue
		}
		if err != nil || hashDirEntries(entries) != d.Hash {
			return false
		}
	}
	for _, f := range rec.Files {
		fi, err := os.Stat(f.Path)
		if err != nil || fi.Size() != f.Size || fi.ModTime().UnixNano() != f.ModTime {
			return false
		}
	}
	return true
}

func collectSourcePatchCacheInputs(runtimeDir, goroot string) ([]depfileDirStat, []sourcePatchFileStat, bool) {
	var dirs []string
	var filePaths []string
	seenFiles := make(map[string]bool)
	addFile := func(p string) {
		if !seenFiles[p] {
			seenFiles[p] = true
			filePaths = append(filePaths, p)
		}
	}
	for _, pkgPath := range llruntime.SourcePatchPkgPaths() {
		patchPath := pkgPath
		switch pkgPath {
		case "runtime/internal/atomic":
			patchPath = "internal/runtime/atomic"
		case "runtime/internal/syscall":
			patchPath = "internal/runtime/syscall"
		}
		patchDir := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(patchPath))
		dirs = append(dirs, patchDir)
		entries, err := os.ReadDir(patchDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, false
		}
		hasPatchGo := false
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				hasPatchGo = true
				addFile(filepath.Join(patchDir, e.Name()))
			}
		}
		if !hasPatchGo {
			continue
		}
		srcDir := filepath.Join(goroot, "src", filepath.FromSlash(pkgPath))
		dirs = append(dirs, srcDir)
		srcEntries, err := os.ReadDir(srcDir)
		if err != nil {
			if os.IsNotExist(err) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
				continue
			}
			return nil, nil, false
		}
		for _, e := range srcEntries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if (strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")) ||
				(strings.HasSuffix(name, ".s") && !strings.HasSuffix(name, "_test.s")) {
				addFile(filepath.Join(srcDir, name))
			}
		}
	}
	dirStats, err := statDepfileDirs(dirs)
	if err != nil {
		return nil, nil, false
	}
	sort.Strings(filePaths)
	fileStats := make([]sourcePatchFileStat, 0, len(filePaths))
	for _, p := range filePaths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, nil, false
		}
		fileStats = append(fileStats, sourcePatchFileStat{
			Path:    p,
			Size:    fi.Size(),
			ModTime: fi.ModTime().UnixNano(),
		})
	}
	return dirStats, fileStats, true
}

func cloneOverlay(src map[string][]byte) map[string][]byte {
	if len(src) == 0 {
		return nil
	}
	dup := make(map[string][]byte, len(src))
	for k, v := range src {
		dup[k] = slices.Clone(v)
	}
	return dup
}

type sourcePatchBuildContext struct {
	goos       string
	goarch     string
	goversion  string
	toolTags   []string
	buildFlags []string
	targetTags []string
	cgoEnabled *bool
}

func buildSourcePatchOverlayForGOROOT(base map[string][]byte, runtimeDir, goroot string, ctx sourcePatchBuildContext) (map[string][]byte, map[string][]string, error) {
	var cachePath string
	if cacheEnabled() && len(base) == 0 && runtimeDir != "" && goroot != "" {
		if root := cacheRootFunc(); root != "" {
			cachePath = filepath.Join(root, "meta", "srcpatch-"+sourcePatchCacheKey(runtimeDir, goroot, ctx)+".bin")
			if raw, err := os.ReadFile(cachePath); err == nil {
				if rec, err := unmarshalSourcePatchCache(raw); err == nil && validateSourcePatchCache(rec) {
					return rec.Overlay, rec.PkgFiles, nil
				}
			}
		}
	}
	var out map[string][]byte
	pkgFiles := make(map[string][]string)
	for _, pkgPath := range llruntime.SourcePatchPkgPaths() {
		changed, next, files, err := applySourcePatchForPkg(base, out, runtimeDir, goroot, pkgPath, ctx)
		if err != nil {
			return nil, nil, err
		}
		if !changed {
			continue
		}
		out = next
		pkgFiles[pkgPath] = files
	}
	if out == nil {
		if cachePath != "" {
			if dirs, files, ok := collectSourcePatchCacheInputs(runtimeDir, goroot); ok {
				_ = writeFileAtomicBytes(cachePath, marshalSourcePatchCache(&sourcePatchCacheRecord{
					Dirs:  dirs,
					Files: files,
				}))
			}
		}
		return base, nil, nil
	}
	if cachePath != "" {
		if dirs, files, ok := collectSourcePatchCacheInputs(runtimeDir, goroot); ok {
			_ = writeFileAtomicBytes(cachePath, marshalSourcePatchCache(&sourcePatchCacheRecord{
				Dirs:     dirs,
				Files:    files,
				Overlay:  out,
				PkgFiles: pkgFiles,
			}))
		}
	}
	return out, pkgFiles, nil
}

func applySourcePatchForPkg(base, current map[string][]byte, runtimeDir, goroot, pkgPath string, ctx sourcePatchBuildContext) (bool, map[string][]byte, []string, error) {
	patchPath := pkgPath
	// Go 1.23 moved these packages out of runtime without changing the
	// interfaces used by the bare-metal source patches.
	switch pkgPath {
	case "runtime/internal/atomic":
		patchPath = "internal/runtime/atomic"
	case "runtime/internal/syscall":
		patchPath = "internal/runtime/syscall"
	}
	patchDir := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(patchPath))
	entries, err := os.ReadDir(patchDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, current, nil, nil
		}
		return false, nil, nil, fmt.Errorf("read source patch dir %s: %w", pkgPath, err)
	}

	srcDir := filepath.Join(goroot, "src", filepath.FromSlash(pkgPath))
	srcEntries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, current, nil, nil
		}
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return false, current, nil, nil
		}
		return false, nil, nil, fmt.Errorf("read stdlib dir %s: %w", pkgPath, err)
	}

	var (
		out       = current
		changed   bool
		patchSrcs = make(map[string]struct {
			filename string
			data     []byte
		})
		skipAll bool
		skips   = make(map[string]struct{})
	)
	readOverlay := func(filename string) ([]byte, error) {
		if out != nil {
			if src, ok := out[filename]; ok {
				return src, nil
			}
		}
		if src, ok := base[filename]; ok {
			return src, nil
		}
		return os.ReadFile(filename)
	}
	buildCtx, err := newSourcePatchMatchContext(goroot, ctx)
	if err != nil {
		return false, nil, nil, err
	}
	// LLGo may be built with a different Go toolchain from the GOROOT it is
	// compiling. Keep GOOS/GOARCH filename filtering, but do not use the host
	// toolchain's goexperiment and architecture feature tags to discard source
	// files whose declarations still need to be replaced.
	filenameCtx := buildCtx
	filenameCtx.OpenFile = func(string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("package sourcepatch\n")), nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		match, err := buildCtx.MatchFile(patchDir, name)
		if err != nil {
			return false, nil, nil, fmt.Errorf("match source patch file %s: %w", filepath.Join(patchDir, name), err)
		}
		if !match {
			continue
		}
		filename := filepath.Join(patchDir, name)
		src, err := readOverlay(filename)
		if err != nil {
			return false, nil, nil, fmt.Errorf("read source patch file %s: %w", filename, err)
		}
		directives, err := collectSourcePatchDirectives(src)
		if err != nil {
			return false, nil, nil, fmt.Errorf("parse source patch directives %s: %w", filename, err)
		}
		patchSrcs[name] = struct {
			filename string
			data     []byte
		}{
			filename,
			buildInjectedSourcePatchFile(filename, src),
		}
		if directives.skipAll {
			skipAll = true
		}
		for name := range directives.skips {
			skips[name] = struct{}{}
		}
	}
	if len(patchSrcs) == 0 {
		return false, current, nil, nil
	}

	ensureOverlay := func() {
		if out == nil {
			out = cloneOverlay(base)
			if out == nil {
				out = make(map[string][]byte)
			}
		}
	}

	if llruntime.SourcePatchReplacesAsm(pkgPath, buildCtx.GOARCH, buildCtx.BuildTags) {
		for _, entry := range srcEntries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".s") || strings.HasSuffix(name, "_test.s") {
				continue
			}
			match, err := matchSourcePatchTargetFile(buildCtx, filenameCtx, srcDir, name)
			if err != nil {
				return false, nil, nil, fmt.Errorf("match stdlib assembly file %s: %w", filepath.Join(srcDir, name), err)
			}
			if !match {
				continue
			}
			ensureOverlay()
			out[filepath.Join(srcDir, name)] = []byte("// replaced by LLGo source patch\n")
			changed = true
		}
	}

	var files []string
	if skipAll {
		for _, entry := range srcEntries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			match, err := matchSourcePatchTargetFile(buildCtx, filenameCtx, srcDir, name)
			if err != nil {
				return false, nil, nil, fmt.Errorf("match stdlib source file %s: %w", filepath.Join(srcDir, name), err)
			}
			if !match {
				continue
			}
			filename := filepath.Join(srcDir, name)
			src, err := readOverlay(filename)
			if err != nil {
				return false, nil, nil, fmt.Errorf("read stdlib source file %s: %w", filename, err)
			}
			stub, err := packageStubSource(src)
			if err != nil {
				return false, nil, nil, fmt.Errorf("build stdlib stub %s: %w", filename, err)
			}
			ensureOverlay()
			out[filename] = stub
			changed = true
		}
	} else if len(skips) != 0 {
		for _, entry := range srcEntries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			match, err := matchSourcePatchTargetFile(buildCtx, filenameCtx, srcDir, name)
			if err != nil {
				return false, nil, nil, fmt.Errorf("match stdlib source file %s: %w", filepath.Join(srcDir, name), err)
			}
			if !match {
				continue
			}
			filename := filepath.Join(srcDir, name)
			src, err := readOverlay(filename)
			if err != nil {
				return false, nil, nil, fmt.Errorf("read stdlib source file %s: %w", filename, err)
			}
			if !sourcePatchMayContainSkip(src, skips) {
				continue
			}
			filtered, changedFile, err := filterSourcePatchFile(src, skips)
			if err != nil {
				return false, nil, nil, fmt.Errorf("filter stdlib source file %s: %w", filename, err)
			}
			if !changedFile {
				continue
			}
			ensureOverlay()
			out[filename] = filtered
			changed = true
		}
	}

	for name, src := range patchSrcs {
		target := filepath.Join(srcDir, "z_llgo_patch_"+name)
		ensureOverlay()
		out[target] = src.data
		files = append(files, src.filename)
		changed = true
	}
	return changed, out, files, nil
}

func matchSourcePatchTargetFile(buildCtx, filenameCtx build.Context, dir, name string) (bool, error) {
	match, err := buildCtx.MatchFile(dir, name)
	if err != nil || match {
		return match, err
	}
	return filenameCtx.MatchFile(dir, name)
}

func sourcePatchMayContainSkip(src []byte, skips map[string]struct{}) bool {
	for key := range skips {
		if i := strings.LastIndexByte(key, '.'); i >= 0 {
			key = key[i+1:]
		}
		if bytes.Contains(src, []byte(key)) {
			return true
		}
	}
	return false
}

func newSourcePatchMatchContext(goroot string, ctx sourcePatchBuildContext) (build.Context, error) {
	buildCtx := build.Default
	if goroot != "" {
		buildCtx.GOROOT = goroot
	}
	if ctx.goos != "" {
		buildCtx.GOOS = ctx.goos
	}
	if ctx.goarch != "" {
		buildCtx.GOARCH = ctx.goarch
	}
	buildCtx.BuildTags = sourcePatchBuildTags(ctx.buildFlags, ctx.targetTags)
	if ctx.cgoEnabled != nil {
		buildCtx.CgoEnabled = *ctx.cgoEnabled
	}
	if ctx.toolTags != nil {
		buildCtx.ToolTags = slices.Clone(ctx.toolTags)
	}
	if ctx.goversion != "" {
		releaseTags, err := releaseTagsForGoVersion(ctx.goversion)
		if err != nil {
			return build.Context{}, err
		}
		buildCtx.ReleaseTags = releaseTags
	}
	return buildCtx, nil
}

// sourcePatchBuildTags is shared by the overlay and the assembly planner.
func sourcePatchBuildTags(buildFlags, targetTags []string) []string {
	return append(parseSourcePatchBuildTags(buildFlags), targetTags...)
}

func parseSourcePatchBuildTags(buildFlags []string) []string {
	var tags []string
	for i := 0; i < len(buildFlags); i++ {
		flag := buildFlags[i]
		if flag == "-tags" && i+1 < len(buildFlags) {
			tags = append(tags, splitSourcePatchBuildTags(buildFlags[i+1])...)
			i++
			continue
		}
		if strings.HasPrefix(flag, "-tags=") {
			tags = append(tags, splitSourcePatchBuildTags(strings.TrimPrefix(flag, "-tags="))...)
		}
	}
	return slices.Compact(tags)
}

func splitSourcePatchBuildTags(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
}

func releaseTagsForGoVersion(goVersion string) ([]string, error) {
	goVersion = strings.TrimPrefix(goVersion, "go")
	parts := strings.Split(goVersion, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("unsupported Go version %q", goVersion)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("parse Go major version %q: %w", goVersion, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parse Go minor version %q: %w", goVersion, err)
	}
	if major != 1 || minor < 1 {
		return nil, fmt.Errorf("unsupported Go version %q", goVersion)
	}
	tags := make([]string, 0, minor)
	for v := 1; v <= minor; v++ {
		tags = append(tags, fmt.Sprintf("go1.%d", v))
	}
	return tags, nil
}

func packageStubSource(src []byte) ([]byte, error) {
	lines := strings.SplitAfter(string(src), "\n")
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(line)
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			return []byte(buf.String()), nil
		}
	}
	return nil, fmt.Errorf("package clause not found")
}

type sourcePatchDirectives struct {
	skipAll bool
	skips   map[string]struct{}
}

func collectSourcePatchDirectives(src []byte) (sourcePatchDirectives, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return sourcePatchDirectives{}, err
	}
	d := sourcePatchDirectives{skips: make(map[string]struct{})}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			line := strings.TrimSpace(comment.Text)
			skipAll, names, ok := parseSourcePatchDirective(line)
			if !ok {
				continue
			}
			if skipAll {
				d.skipAll = true
			}
			for _, name := range names {
				d.skips[name] = struct{}{}
			}
		}
	}
	for _, decl := range file.Decls {
		for _, name := range declPatchKeys(decl) {
			d.skips[name] = struct{}{}
		}
	}
	return d, nil
}

// parseSourcePatchDirective recognizes build-time source-patch directives.
//
// Supported directives:
//   - //llgo:skipall: clear every stdlib .go file in the patched package to a package stub
//   - //llgo:skip A B: comment the named declarations from the stdlib package view
//
// Unlike cl/import.go directives, these are consumed only while constructing the
// load-time overlay and are rewritten to plain comments before type checking.
func parseSourcePatchDirective(line string) (skipAll bool, names []string, ok bool) {
	d, parsed := directive.Parse(&ast.Comment{Text: line})
	if !parsed {
		return false, nil, false
	}
	switch d.Name {
	case "llgo:skipall":
		if d.Args == "" {
			return true, nil, true
		}
	case "llgo:skip":
		if d.Args != "" {
			return false, strings.Fields(d.Args), true
		}
	}
	return false, nil, false
}

func sanitizeSourcePatchDirectiveLines(src []byte) []byte {
	out := slices.Clone(src)
	lines := bytes.SplitAfter(out, []byte{'\n'})
	changed := false
	markDirective := func(line []byte, prefix []byte) bool {
		idx := bytes.Index(line, prefix)
		if idx < 0 {
			return false
		}
		line[idx+len(prefix)-1] = '_'
		return true
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if _, _, ok := parseSourcePatchDirective(trimmed); !ok {
			continue
		}
		switch {
		case markDirective(line, []byte("//llgo:")):
			changed = true
		case markDirective(line, []byte("// llgo:")):
			changed = true
		}
	}
	if !changed {
		return src
	}
	return out
}

func buildInjectedSourcePatchFile(filename string, src []byte) []byte {
	sanitized := sanitizeSourcePatchDirectiveLines(src)
	newline := "\n"
	if first := bytes.IndexByte(src, '\n'); first > 0 && src[first-1] == '\r' {
		newline = "\r\n"
	}
	var out bytes.Buffer
	out.WriteString("//line ")
	out.WriteString(filepath.ToSlash(filename))
	out.WriteString(":1")
	out.WriteString(newline)
	out.Write(sanitized)
	return out.Bytes()
}

func filterSourcePatchFile(src []byte, skips map[string]struct{}) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}
	tokFile := fset.File(file.Pos())
	if tokFile == nil {
		return nil, false, fmt.Errorf("missing file positions")
	}
	changed := false
	var spans []sourcePatchSpan
	decls := file.Decls[:0]
	for _, decl := range file.Decls {
		filteredDecl, removed, rmSpans, err := filterSourcePatchDecl(tokFile, decl, skips)
		if err != nil {
			return nil, false, err
		}
		if removed {
			changed = true
		}
		spans = append(spans, rmSpans...)
		if filteredDecl != nil {
			decls = append(decls, filteredDecl)
		}
	}
	if !changed {
		return src, false, nil
	}
	file.Decls = decls
	spans = append(spans, collectUnusedImportSpans(tokFile, file)...)
	if len(spans) == 0 {
		return src, false, nil
	}
	return commentSourcePatchSpans(src, spans), true, nil
}

type sourcePatchSpan struct {
	start int
	end   int
}

func filterSourcePatchDecl(tokFile *token.File, decl ast.Decl, skips map[string]struct{}) (ast.Decl, bool, []sourcePatchSpan, error) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		if _, ok := skips[declPatchKeyFunc(decl)]; ok {
			return nil, true, nodeAndCommentsSpans(tokFile, decl), nil
		}
		return decl, false, nil, nil
	case *ast.GenDecl:
		switch decl.Tok {
		case token.TYPE, token.VAR, token.CONST:
		default:
			return decl, false, nil, nil
		}
		specs := decl.Specs[:0]
		removedAny := false
		var removedSpans []sourcePatchSpan
		for _, spec := range decl.Specs {
			filteredSpec, removed, spans := filterSourcePatchSpec(tokFile, spec, skips)
			if removed {
				removedAny = true
			}
			removedSpans = append(removedSpans, spans...)
			if filteredSpec != nil {
				specs = append(specs, filteredSpec)
			}
		}
		if len(specs) == 0 {
			return nil, removedAny, nodeAndCommentsSpans(tokFile, decl), nil
		}
		if !removedAny {
			return decl, false, nil, nil
		}
		out := *decl
		out.Specs = specs
		return &out, true, removedSpans, nil
	default:
		return decl, false, nil, nil
	}
}

func filterSourcePatchSpec(tokFile *token.File, spec ast.Spec, skips map[string]struct{}) (ast.Spec, bool, []sourcePatchSpan) {
	switch spec := spec.(type) {
	case *ast.TypeSpec:
		if _, ok := skips[spec.Name.Name]; ok {
			return nil, true, nodeAndCommentsSpans(tokFile, spec)
		}
		return spec, false, nil
	case *ast.ValueSpec:
		keep := make([]*ast.Ident, 0, len(spec.Names))
		removed := false
		for _, name := range spec.Names {
			if _, ok := skips[name.Name]; ok {
				removed = true
				continue
			}
			keep = append(keep, name)
		}
		if !removed {
			return spec, false, nil
		}
		if len(keep) == 0 {
			return nil, true, nodeAndCommentsSpans(tokFile, spec)
		}
		if len(keep) != len(spec.Names) {
			// Keep the transformation simple and deterministic. Mixed multi-name specs
			// can be split later if we need them in real patches.
			return nil, true, nodeAndCommentsSpans(tokFile, spec)
		}
		out := *spec
		out.Names = keep
		return &out, true, nil
	default:
		return spec, false, nil
	}
}

func collectUnusedImportSpans(tokFile *token.File, file *ast.File) []sourcePatchSpan {
	var spans []sourcePatchSpan
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		removable := make([]*ast.ImportSpec, 0, len(gen.Specs))
		keptCount := 0
		for _, spec := range gen.Specs {
			imp := spec.(*ast.ImportSpec)
			if imp.Name != nil && (imp.Name.Name == "_" || imp.Name.Name == ".") {
				keptCount++
				continue
			}
			if sourcePatchUsesImport(file, imp) {
				keptCount++
				continue
			}
			removable = append(removable, imp)
		}
		if len(removable) == 0 {
			continue
		}
		if keptCount == 0 || !gen.Lparen.IsValid() || len(gen.Specs) == 1 {
			spans = append(spans, nodeAndCommentsSpans(tokFile, gen)...)
			continue
		}
		for _, imp := range removable {
			spans = append(spans, nodeAndCommentsSpans(tokFile, imp)...)
		}
	}
	return spans
}

func sourcePatchUsesImport(file *ast.File, imp *ast.ImportSpec) bool {
	path := strings.Trim(imp.Path.Value, `"`)
	if imp.Name == nil {
		if altName, ok := versionedImportDefaultName(path); ok && fileUsesPackageIdent(file, altName) {
			return true
		}
	} else if fileUsesPackageIdent(file, imp.Name.Name) {
		return true
	}
	return astutil.UsesImport(file, path)
}

func versionedImportDefaultName(path string) (string, bool) {
	slash := strings.LastIndexByte(path, '/')
	if slash <= 0 || slash+2 > len(path) || path[slash+1] != 'v' {
		return "", false
	}
	for i := slash + 2; i < len(path); i++ {
		if path[i] < '0' || path[i] > '9' {
			return "", false
		}
	}
	prefix := path[:slash]
	if prev := strings.LastIndexByte(prefix, '/'); prev >= 0 {
		prefix = prefix[prev+1:]
	}
	return prefix, prefix != ""
}

func fileUsesPackageIdent(file *ast.File, name string) bool {
	used := false
	ast.Inspect(file, func(n ast.Node) bool {
		if used {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == name && id.Obj == nil {
			used = true
			return false
		}
		return true
	})
	return used
}

func nodeAndCommentsSpans(tokFile *token.File, node ast.Node) []sourcePatchSpan {
	spans := []sourcePatchSpan{nodeSpan(tokFile, node)}
	switch node := node.(type) {
	case *ast.FuncDecl:
		spans = append(spans, commentGroupsToSpans(tokFile, compactCommentGroups(node.Doc))...)
	case *ast.GenDecl:
		spans = append(spans, commentGroupsToSpans(tokFile, compactCommentGroups(node.Doc))...)
		for _, spec := range node.Specs {
			spans = append(spans, commentGroupsToSpans(tokFile, collectSpecComments(spec))...)
		}
	case *ast.ImportSpec:
		spans = append(spans, commentGroupsToSpans(tokFile, compactCommentGroups(node.Doc, node.Comment))...)
	default:
		if spec, ok := node.(ast.Spec); ok {
			spans = append(spans, commentGroupsToSpans(tokFile, collectSpecComments(spec))...)
		}
	}
	return spans
}

func commentGroupsToSpans(tokFile *token.File, groups []*ast.CommentGroup) []sourcePatchSpan {
	out := make([]sourcePatchSpan, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		out = append(out, nodeSpan(tokFile, group))
	}
	return out
}

func nodeSpan(tokFile *token.File, node ast.Node) sourcePatchSpan {
	return sourcePatchSpan{
		start: tokFile.Offset(node.Pos()),
		end:   tokFile.Offset(node.End()),
	}
}

func commentSourcePatchSpans(src []byte, spans []sourcePatchSpan) []byte {
	if len(spans) == 0 {
		return slices.Clone(src)
	}
	out := make([]byte, 0, len(src)+len(spans)*4)
	slices.SortFunc(spans, func(a, b sourcePatchSpan) int {
		switch {
		case a.start < b.start:
			return -1
		case a.start > b.start:
			return 1
		case a.end < b.end:
			return -1
		case a.end > b.end:
			return 1
		default:
			return 0
		}
	})
	merged := make([]sourcePatchSpan, 0, len(spans))
	for _, span := range spans {
		if span.start < 0 {
			span.start = 0
		}
		if span.end > len(src) {
			span.end = len(src)
		}
		if span.start >= span.end {
			continue
		}
		if len(merged) == 0 || span.start > merged[len(merged)-1].end {
			merged = append(merged, span)
			continue
		}
		if span.end > merged[len(merged)-1].end {
			merged[len(merged)-1].end = span.end
		}
	}
	cursor := 0
	for _, span := range merged {
		if span.start > cursor {
			out = append(out, src[cursor:span.start]...)
		}
		out = appendCommentedSourcePatchSpan(out, src[span.start:span.end])
		cursor = span.end
	}
	out = append(out, src[cursor:]...)
	return out
}

func appendCommentedSourcePatchSpan(dst, src []byte) []byte {
	lines := bytes.SplitAfter(src, []byte{'\n'})
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		newline := len(line)
		if line[newline-1] == '\n' {
			newline--
		}
		body := line[:newline]
		suffix := line[newline:]
		if len(bytes.TrimSpace(body)) == 0 {
			dst = append(dst, line...)
			continue
		}
		indent := 0
		for indent < len(body) && (body[indent] == ' ' || body[indent] == '\t') {
			indent++
		}
		dst = append(dst, body[:indent]...)
		dst = append(dst, '/', '/')
		if indent < len(body) {
			dst = append(dst, ' ')
			dst = append(dst, body[indent:]...)
		}
		dst = append(dst, suffix...)
	}
	return dst
}

func collectSpecComments(spec ast.Spec) []*ast.CommentGroup {
	switch spec := spec.(type) {
	case *ast.TypeSpec:
		return compactCommentGroups(spec.Doc, spec.Comment)
	case *ast.ValueSpec:
		return compactCommentGroups(spec.Doc, spec.Comment)
	default:
		return nil
	}
}

func compactCommentGroups(groups ...*ast.CommentGroup) []*ast.CommentGroup {
	var out []*ast.CommentGroup
	for _, group := range groups {
		if group != nil {
			out = append(out, group)
		}
	}
	return out
}

func declPatchKeys(decl ast.Decl) []string {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		// Package init() and blank functions can be declared multiple times in
		// Go without colliding with stdlib declarations. Only an explicit
		// //llgo:skip init directive should suppress original init functions.
		if decl.Recv == nil && (decl.Name.Name == "init" || decl.Name.Name == "_") {
			return nil
		}
		return []string{declPatchKeyFunc(decl)}
	case *ast.GenDecl:
		var out []string
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				if spec.Name.Name != "_" {
					out = append(out, spec.Name.Name)
				}
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					if name.Name != "_" {
						out = append(out, name.Name)
					}
				}
			}
		}
		return out
	default:
		return nil
	}
}

func declPatchKeyFunc(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return decl.Name.Name
	}
	return recvPatchKey(decl.Recv.List[0].Type) + "." + decl.Name.Name
}

func recvPatchKey(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.ParenExpr:
		return recvPatchKey(expr.X)
	case *ast.StarExpr:
		return "(*" + recvPatchKey(expr.X) + ")"
	case *ast.IndexExpr:
		return recvPatchKey(expr.X)
	case *ast.IndexListExpr:
		return recvPatchKey(expr.X)
	case *ast.SelectorExpr:
		return expr.Sel.Name
	default:
		// Source patches are expected to use ordinary named receiver types.
		// Treat unsupported receiver forms as a fail-fast patch authoring error
		// instead of silently leaving the original declaration active.
		panic(fmt.Sprintf("unhandled expression type in recvPatchKey: %T", expr))
	}
}
