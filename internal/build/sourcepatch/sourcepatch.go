// Package sourcepatch constructs load-time Go standard-library overlays from
// LLGo's runtime/_patch directory, handling //llgo:skipall and //llgo:skip
// directives without mutating GOROOT on disk.
package sourcepatch

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
	"strconv"
	"strings"
	"syscall"

	llruntime "github.com/xgo-dev/llgo/runtime"
	"golang.org/x/tools/go/ast/astutil"
)

// CloneOverlay returns a deep copy of an in-memory file overlay map.
func CloneOverlay(src map[string][]byte) map[string][]byte {
	if len(src) == 0 {
		return nil
	}
	dup := make(map[string][]byte, len(src))
	for k, v := range src {
		dup[k] = slices.Clone(v)
	}
	return dup
}

// BuildContext configures Go build-tag and target matching when constructing
// stdlib source-patch overlays.
type BuildContext struct {
	GOOS       string
	GOARCH     string
	GoVersion  string
	ToolTags   []string
	BuildFlags []string
	TargetTags []string
	CgoEnabled *bool
}

// BuildOverlayForGOROOT applies all configured runtime/_patch overlays against
// goroot, returning the updated overlay and the list of injected patch files
// keyed by standard-library package path.
func BuildOverlayForGOROOT(base map[string][]byte, runtimeDir, goroot string, ctx BuildContext) (map[string][]byte, map[string][]string, error) {
	var out map[string][]byte
	pkgFiles := make(map[string][]string)
	for _, pkgPath := range llruntime.SourcePatchPkgPaths() {
		changed, next, files, err := ApplyForPkg(base, out, runtimeDir, goroot, pkgPath, ctx)
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
		return base, nil, nil
	}
	return out, pkgFiles, nil
}

// ApplyForPkg applies source patches for a single standard-library package.
func ApplyForPkg(base, current map[string][]byte, runtimeDir, goroot, pkgPath string, ctx BuildContext) (bool, map[string][]byte, []string, error) {
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
	buildCtx, err := NewMatchContext(goroot, ctx)
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
		directives, err := collectDirectives(src)
		if err != nil {
			return false, nil, nil, fmt.Errorf("parse source patch directives %s: %w", filename, err)
		}
		patchSrcs[name] = struct {
			filename string
			data     []byte
		}{
			filename,
			BuildInjectedFile(filename, src),
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
			out = CloneOverlay(base)
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
			match, err := matchTargetFile(buildCtx, filenameCtx, srcDir, name)
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
			match, err := matchTargetFile(buildCtx, filenameCtx, srcDir, name)
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
			match, err := matchTargetFile(buildCtx, filenameCtx, srcDir, name)
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
			if !MayContainSkip(src, skips) {
				continue
			}
			filtered, changedFile, err := FilterFile(src, skips)
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

func matchTargetFile(buildCtx, filenameCtx build.Context, dir, name string) (bool, error) {
	match, err := buildCtx.MatchFile(dir, name)
	if err != nil || match {
		return match, err
	}
	return filenameCtx.MatchFile(dir, name)
}

// MayContainSkip performs a fast byte-level prefilter before parsing src.
func MayContainSkip(src []byte, skips map[string]struct{}) bool {
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

// NewMatchContext constructs a go/build.Context configured for ctx.
func NewMatchContext(goroot string, ctx BuildContext) (build.Context, error) {
	buildCtx := build.Default
	if goroot != "" {
		buildCtx.GOROOT = goroot
	}
	if ctx.GOOS != "" {
		buildCtx.GOOS = ctx.GOOS
	}
	if ctx.GOARCH != "" {
		buildCtx.GOARCH = ctx.GOARCH
	}
	buildCtx.BuildTags = BuildTags(ctx.BuildFlags, ctx.TargetTags)
	if ctx.CgoEnabled != nil {
		buildCtx.CgoEnabled = *ctx.CgoEnabled
	}
	if ctx.ToolTags != nil {
		buildCtx.ToolTags = slices.Clone(ctx.ToolTags)
	}
	if ctx.GoVersion != "" {
		releaseTags, err := ReleaseTagsForGoVersion(ctx.GoVersion)
		if err != nil {
			return build.Context{}, err
		}
		buildCtx.ReleaseTags = releaseTags
	}
	return buildCtx, nil
}

// BuildTags combines build tags parsed from -tags flags with targetTags.
func BuildTags(buildFlags, targetTags []string) []string {
	return append(ParseBuildTags(buildFlags), targetTags...)
}

// ParseBuildTags parses -tags flags from buildFlags.
func ParseBuildTags(buildFlags []string) []string {
	var tags []string
	for i := 0; i < len(buildFlags); i++ {
		flag := buildFlags[i]
		if flag == "-tags" && i+1 < len(buildFlags) {
			tags = append(tags, SplitBuildTags(buildFlags[i+1])...)
			i++
			continue
		}
		if strings.HasPrefix(flag, "-tags=") {
			tags = append(tags, SplitBuildTags(strings.TrimPrefix(flag, "-tags="))...)
		}
	}
	return slices.Compact(tags)
}

// SplitBuildTags splits a Go -tags argument on commas and spaces.
func SplitBuildTags(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
}

// ReleaseTagsForGoVersion returns the Go release tags (go1.1..go1.N) for goVersion.
func ReleaseTagsForGoVersion(goVersion string) ([]string, error) {
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

type patchDirectives struct {
	skipAll bool
	skips   map[string]struct{}
}

func collectDirectives(src []byte) (patchDirectives, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return patchDirectives{}, err
	}
	d := patchDirectives{skips: make(map[string]struct{})}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			line := strings.TrimSpace(comment.Text)
			skipAll, names, ok := ParseDirective(line)
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
		for _, name := range DeclKeys(decl) {
			d.skips[name] = struct{}{}
		}
	}
	return d, nil
}

// ParseDirective recognizes build-time source-patch directives.
//
// Supported directives:
//   - //llgo:skipall: clear every stdlib .go file in the patched package to a package stub
//   - //llgo:skip A B: comment the named declarations from the stdlib package view
//
// Unlike cl/import.go directives, these are consumed only while constructing the
// load-time overlay and are rewritten to plain comments before type checking.
func ParseDirective(line string) (skipAll bool, names []string, ok bool) {
	const (
		llgo1 = "//llgo:"
		llgo2 = "// llgo:"
		go1   = "//go:"
	)
	if strings.HasPrefix(line, go1) {
		return false, nil, false
	}
	var tail string
	switch {
	case strings.HasPrefix(line, llgo1):
		tail = line[len(llgo1):]
	case strings.HasPrefix(line, llgo2):
		tail = line[len(llgo2):]
	default:
		return false, nil, false
	}
	switch {
	case tail == "skipall":
		return true, nil, true
	case strings.HasPrefix(tail, "skip "):
		return false, strings.Fields(tail[len("skip "):]), true
	default:
		return false, nil, false
	}
}

func sanitizeDirectiveLines(src []byte) []byte {
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
		if _, _, ok := ParseDirective(trimmed); !ok {
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

// BuildInjectedFile prefixes an injected patch file with a //line directive
// preserving original positions and neutralizes consumed //llgo:skip* lines.
func BuildInjectedFile(filename string, src []byte) []byte {
	sanitized := sanitizeDirectiveLines(src)
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

// FilterFile comments out declarations listed in skips and any imports that
// become unused as a result.
func FilterFile(src []byte, skips map[string]struct{}) ([]byte, bool, error) {
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
	var spans []byteSpan
	decls := file.Decls[:0]
	for _, decl := range file.Decls {
		filteredDecl, removed, rmSpans, err := filterDecl(tokFile, decl, skips)
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
	return commentSpans(src, spans), true, nil
}

type byteSpan struct {
	start int
	end   int
}

func filterDecl(tokFile *token.File, decl ast.Decl, skips map[string]struct{}) (ast.Decl, bool, []byteSpan, error) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		if _, ok := skips[declKeyFunc(decl)]; ok {
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
		var removedSpans []byteSpan
		for _, spec := range decl.Specs {
			filteredSpec, removed, spans := filterSpec(tokFile, spec, skips)
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

func filterSpec(tokFile *token.File, spec ast.Spec, skips map[string]struct{}) (ast.Spec, bool, []byteSpan) {
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

func collectUnusedImportSpans(tokFile *token.File, file *ast.File) []byteSpan {
	var spans []byteSpan
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		removable := make([]*ast.ImportSpec, 0, len(gen.Specs))
		keptCount := 0
		for _, spec := range gen.Specs {
			imp := spec.(*ast.ImportSpec)
			if imp.Name != nil && imp.Name.Name == "_" {
				keptCount++
				continue
			}
			if astutil.UsesImport(file, strings.Trim(imp.Path.Value, `"`)) {
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

func nodeAndCommentsSpans(tokFile *token.File, node ast.Node) []byteSpan {
	spans := []byteSpan{nodeSpan(tokFile, node)}
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

func commentGroupsToSpans(tokFile *token.File, groups []*ast.CommentGroup) []byteSpan {
	out := make([]byteSpan, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		out = append(out, nodeSpan(tokFile, group))
	}
	return out
}

func nodeSpan(tokFile *token.File, node ast.Node) byteSpan {
	return byteSpan{
		start: tokFile.Offset(node.Pos()),
		end:   tokFile.Offset(node.End()),
	}
}

func commentSpans(src []byte, spans []byteSpan) []byte {
	if len(spans) == 0 {
		return slices.Clone(src)
	}
	out := make([]byte, 0, len(src)+len(spans)*4)
	slices.SortFunc(spans, func(a, b byteSpan) int {
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
	merged := make([]byteSpan, 0, len(spans))
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
		out = appendCommentedSpan(out, src[span.start:span.end])
		cursor = span.end
	}
	out = append(out, src[cursor:]...)
	return out
}

func appendCommentedSpan(dst, src []byte) []byte {
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

// DeclKeys returns the patch keys defined by decl (function names, receiver
// method keys like "(*T).M", or type/var/const identifier names).
func DeclKeys(decl ast.Decl) []string {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		return []string{declKeyFunc(decl)}
	case *ast.GenDecl:
		var out []string
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				out = append(out, spec.Name.Name)
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					out = append(out, name.Name)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func declKeyFunc(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return decl.Name.Name
	}
	return RecvKey(decl.Recv.List[0].Type) + "." + decl.Name.Name
}

// RecvKey returns the normalized receiver type key for a method declaration.
func RecvKey(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.ParenExpr:
		return RecvKey(expr.X)
	case *ast.StarExpr:
		return "(*" + RecvKey(expr.X) + ")"
	case *ast.IndexExpr:
		return RecvKey(expr.X)
	case *ast.IndexListExpr:
		return RecvKey(expr.X)
	case *ast.SelectorExpr:
		return expr.Sel.Name
	default:
		// Source patches are expected to use ordinary named receiver types.
		// Treat unsupported receiver forms as a fail-fast patch authoring error
		// instead of silently leaving the original declaration active.
		panic(fmt.Sprintf("unhandled expression type in RecvKey: %T", expr))
	}
}
