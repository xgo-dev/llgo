// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestLinkSnapshotSidecarRoundTripAndErrors(t *testing.T) {
	fset := token.NewFileSet()
	goPkg := types.NewPackage("example.com/lsn", "lsn")
	st := types.NewStruct([]*types.Var{types.NewField(0, goPkg, "X", types.Typ[types.Int], false)}, nil)
	named := types.NewNamed(types.NewTypeName(0, goPkg, "Pair", nil), st, nil)
	goPkg.Scope().Insert(named.Obj())
	cbSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int32])),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int32])), false)

	progSSA := ssa.NewProgram(fset, 0)
	ap := &aPackage{
		Package: &packages.Package{ID: goPkg.Path(), PkgPath: goPkg.Path(), Name: goPkg.Name(), Types: goPkg},
		linkSnapshot: &packageLinkSnapshot{
			needAbiInit:     2,
			hasLocalExports: true,
			methodByIndex:   map[int]none{0: {}, 3: {}},
			methodByName:    map[string]none{"String": {}, "Error": {}},
			abiSymbols:      map[string]none{"_llgo_lsn_Pair": {}},
			abiTypes: []llvmgen.AbiTypeInfo{
				{Name: "_llgo_lsn_Pair", Raw: named},
				{Name: "_llgo_unowned", Raw: types.Typ[types.Int]},
			},
			cExports: []cExport{
				{goName: "example.com/lsn.Add", cName: "Add", sig: cbSig},
				{goName: "example.com/lsn.NoSig", cName: "NoSig", sig: nil},
			},
		},
	}
	ctx := &context{
		progSSA: progSSA,
		pkgByID: map[string]Package{goPkg.Path(): ap},
	}

	// Nil package / nil snapshot are no-ops.
	if err := ctx.writeLinkSnapshotSidecar(filepath.Join(t.TempDir(), "noop.linksnap"), nil); err != nil {
		t.Fatalf("writeLinkSnapshotSidecar(nil) = %v", err)
	}
	if err := ctx.writeLinkSnapshotSidecar(filepath.Join(t.TempDir(), "noop2.linksnap"), &aPackage{}); err != nil {
		t.Fatalf("writeLinkSnapshotSidecar(nil snap) = %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "pkg.linksnap")
	if err := ctx.writeLinkSnapshotSidecar(path, ap); err != nil {
		t.Fatalf("writeLinkSnapshotSidecar: %v", err)
	}
	got, err := ctx.readLinkSnapshotSidecar(path)
	if err != nil {
		t.Fatalf("readLinkSnapshotSidecar: %v", err)
	}
	if got.needAbiInit != 2 || !got.hasLocalExports || len(got.methodByIndex) != 2 ||
		len(got.methodByName) != 2 || len(got.abiSymbols) != 1 || len(got.abiTypes) != 1 || len(got.cExports) != 2 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.cExports[1].sig != nil || got.cExports[0].sig == nil {
		t.Fatalf("unexpected cExports signatures: %+v", got.cExports)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Missing file.
	if _, err := ctx.readLinkSnapshotSidecar(filepath.Join(dir, "missing.linksnap")); err == nil {
		t.Fatal("expected error for missing linksnap")
	}
	// Missing progSSA when type graph blob is present.
	if _, err := (&context{}).readLinkSnapshotSidecar(path); err == nil {
		t.Fatal("expected error when progSSA is nil")
	}
	// Every strict truncation prefix must fail cleanly without panic.
	truncPath := filepath.Join(dir, "trunc.linksnap")
	for i := 0; i < len(raw); i++ {
		if err := os.WriteFile(truncPath, raw[:i], 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ctx.readLinkSnapshotSidecar(truncPath); err == nil {
			t.Fatalf("truncation prefix len=%d unexpectedly succeeded", i)
		}
	}

	// Bad magic, negative method index, huge counts, bad JSON, bad ABI/CExport, and incomplete blob.
	writeAndExpectErr := func(name string, payload []byte) {
		t.Helper()
		p := filepath.Join(dir, name+".linksnap")
		if err := os.WriteFile(p, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ctx.readLinkSnapshotSidecar(p); err == nil {
			t.Fatalf("%s: expected readLinkSnapshotSidecar error", name)
		}
	}
	writeAndExpectErr("bad-magic", []byte("BADMAGIC0000"))

	makeBuf := func(fn func(*bytes.Buffer)) []byte {
		var buf bytes.Buffer
		buf.WriteString(linkSnapMagic)
		fn(&buf)
		return buf.Bytes()
	}
	writeAndExpectErr("huge-idx-count", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 99999)
	}))
	writeAndExpectErr("neg-method-idx", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 1)
		writeBinVarint(b, -5)
	}))
	writeAndExpectErr("huge-names-count", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 99999)
	}))
	writeAndExpectErr("huge-syms-count", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 99999)
	}))
	writeAndExpectErr("huge-abi-count", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 99999)
	}))
	writeAndExpectErr("huge-cexp-count", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 99999)
	}))
	writeAndExpectErr("incomplete-no-blob", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 1)
		writeBinStr(b, "_llgo_a")
		writeBinUvarint(b, 0)
		writeBinBytes(b, nil)
	}))
	writeAndExpectErr("bad-json-blob", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinBytes(b, []byte("{bad json"))
	}))
	badABIBlob, _ := json.Marshal(linkSnapTypePayload{ABI: []frontendABIType{{Name: "x", Type: 99}}})
	writeAndExpectErr("bad-abi-ref", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinBytes(b, badABIBlob)
	}))
	badCExpBlob, _ := json.Marshal(linkSnapTypePayload{CExports: []frontendCExport{{GoName: "g", CName: "c", Signature: 99}}})
	writeAndExpectErr("bad-cexp-ref", makeBuf(func(b *bytes.Buffer) {
		writeBinVarint(b, 0)
		writeBinBool(b, false)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinUvarint(b, 0)
		writeBinBytes(b, badCExpBlob)
	}))

	// writeFileAtomicBytes error when parent is an existing file.
	blocker := filepath.Join(dir, "file-blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomicBytes(filepath.Join(blocker, "child"), []byte("y")); err == nil {
		t.Fatal("expected writeFileAtomicBytes error when parent is a file")
	}
}

func TestDepfileCacheRoundTripValidationAndErrors(t *testing.T) {
	if stats, err := statDepfiles(nil); err != nil || stats != nil {
		t.Fatalf("statDepfiles(nil) = %v, %v", stats, err)
	}
	if dirs, err := statDepfileDirs(nil); err != nil || dirs != nil {
		t.Fatalf("statDepfileDirs(nil) = %v, %v", dirs, err)
	}
	if _, err := marshalDepfile(nil); err == nil {
		t.Fatal("marshalDepfile(nil) expected error")
	}

	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.h")
	if err := os.WriteFile(f1, []byte("#define A 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	missingDir := filepath.Join(dir, "absent-dir")
	missingFile := filepath.Join(dir, "shadow.h")

	// Missing file in statDepfiles returns error.
	if _, err := statDepfiles([]string{missingFile}); err == nil {
		t.Fatal("expected statDepfiles error on missing file")
	}
	// File as directory in statDepfileDirs returns error on POSIX (ENOTDIR).
	// On Windows, os.ReadDir on a regular file fails with ERROR_PATH_NOT_FOUND,
	// which os.IsNotExist classifies as an absent directory.
	if runtime.GOOS != "windows" {
		if _, err := statDepfileDirs([]string{f1}); err == nil {
			t.Fatal("expected statDepfileDirs error on regular file")
		}
	}

	fileStats, err := statDepfiles([]string{"", f1, f1})
	if err != nil || len(fileStats) != 1 {
		t.Fatalf("statDepfiles = %v, %v", fileStats, err)
	}
	dirStats, err := statDepfileDirs([]string{"", dir, dir, missingDir})
	if err != nil || len(dirStats) != 2 {
		t.Fatalf("statDepfileDirs = %v, %v", dirStats, err)
	}

	rec := &depfileRecord{
		Key:              "k1",
		PreprocessedHash: "h1",
		Files:            fileStats,
		Dirs:             dirStats,
		Missing:          []string{missingFile},
	}
	if !validateDepfile(rec) {
		t.Fatal("expected valid depfile")
	}
	raw, err := marshalDepfile(rec)
	if err != nil {
		t.Fatalf("marshalDepfile: %v", err)
	}
	decoded, err := unmarshalDepfile(raw)
	if err != nil || decoded.Key != "k1" || decoded.PreprocessedHash != "h1" ||
		len(decoded.Files) != 1 || len(decoded.Dirs) != 2 || len(decoded.Missing) != 1 {
		t.Fatalf("unmarshalDepfile mismatch: %+v err=%v", decoded, err)
	}

	// Bad magic & every truncation prefix must fail.
	if _, err := unmarshalDepfile([]byte("BADMAGIC")); err == nil {
		t.Fatal("expected bad magic error")
	}
	for i := 0; i < len(raw); i++ {
		if _, err := unmarshalDepfile(raw[:i]); err == nil {
			t.Fatalf("unmarshalDepfile truncation prefix len=%d succeeded", i)
		}
	}
	// Overflow counts.
	for _, corrupt := range [][]byte{
		func() []byte {
			var b bytes.Buffer
			b.WriteString(depfileMagic)
			writeBinStr(&b, "k")
			writeBinStr(&b, "h")
			writeBinUvarint(&b, 99999)
			return b.Bytes()
		}(),
		func() []byte {
			var b bytes.Buffer
			b.WriteString(depfileMagic)
			writeBinStr(&b, "k")
			writeBinStr(&b, "h")
			writeBinUvarint(&b, 0)
			writeBinUvarint(&b, 99999)
			return b.Bytes()
		}(),
		func() []byte {
			var b bytes.Buffer
			b.WriteString(depfileMagic)
			writeBinStr(&b, "k")
			writeBinStr(&b, "h")
			writeBinUvarint(&b, 0)
			writeBinUvarint(&b, 0)
			writeBinUvarint(&b, 99999)
			return b.Bytes()
		}(),
	} {
		if _, err := unmarshalDepfile(corrupt); err == nil {
			t.Fatal("expected corrupt count error")
		}
	}

	// Validation failure branches.
	if validateDepfile(nil) || validateDepfile(&depfileRecord{}) {
		t.Fatal("nil/empty depfile should fail validation")
	}
	badHash := *rec
	badHash.Files = []depfileStat{{Path: f1, Size: fileStats[0].Size, Hash: ""}}
	if validateDepfile(&badHash) {
		t.Fatal("empty file hash should fail")
	}
	wrongHash := *rec
	wrongHash.Files = []depfileStat{{Path: f1, Size: fileStats[0].Size, Hash: "0000"}}
	if validateDepfile(&wrongHash) {
		t.Fatal("mismatched file hash should fail")
	}
	missingFileRec := *rec
	missingFileRec.Files = []depfileStat{{Path: missingFile, Size: 1, Hash: "x"}}
	if validateDepfile(&missingFileRec) {
		t.Fatal("missing dep file should fail")
	}
	// Missing path appears -> invalidate.
	if err := os.WriteFile(missingFile, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if validateDepfile(rec) {
		t.Fatal("appearing missing path should fail validation")
	}
	_ = os.Remove(missingFile)
	// Directory content changes -> invalidate.
	if err := os.WriteFile(filepath.Join(dir, "extra.h"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if validateDepfile(rec) {
		t.Fatal("changed directory entries should fail validation")
	}
	_ = os.Remove(filepath.Join(dir, "extra.h"))
	// Previously missing directory appears -> invalidate.
	if err := os.MkdirAll(missingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if validateDepfile(rec) {
		t.Fatal("appearing absent dir should fail validation")
	}
}

func TestExportSidecarLazyDepAndCollectHelpers(t *testing.T) {
	dir := t.TempDir()
	ctx := &context{
		buildConf:    &Config{Goos: "linux", Goarch: "amd64", Overlay: map[string][]byte{"a.go": []byte("package p")}},
		cacheManager: &cacheManager{root: dir},
		pkgByID:      make(map[string]Package),
	}
	if err := ctx.writeExportMetaSidecar(filepath.Join(dir, "nil.llgoex"), nil); err != nil {
		t.Fatalf("writeExportMetaSidecar(nil) = %v", err)
	}
	if _, err := ctx.readExportMetaSidecar(filepath.Join(dir, "missing.llgoex")); err == nil {
		t.Fatal("expected error reading missing export meta")
	}
	if err := ctx.writePackageTypesSidecar(filepath.Join(dir, "nil.llgotypes"), nil); err == nil {
		t.Fatal("expected error writing nil package types")
	}
	if _, err := ctx.readPackageTypesSidecar(filepath.Join(dir, "missing.llgotypes"), "p", nil); err == nil {
		t.Fatal("expected error reading missing types sidecar")
	}

	// Exercise packageHasOverlay across GoFiles, OtherFiles, EmbedFiles, and none.
	if packageHasOverlay(nil, ctx.buildConf) {
		t.Fatal("nil pkg should not have overlay")
	}
	if !packageHasOverlay(&packages.Package{GoFiles: []string{"a.go"}}, ctx.buildConf) {
		t.Fatal("GoFiles overlay not detected")
	}
	if !packageHasOverlay(&packages.Package{OtherFiles: []string{"a.go"}}, ctx.buildConf) {
		t.Fatal("OtherFiles overlay not detected")
	}
	if !packageHasOverlay(&packages.Package{EmbedFiles: []string{"a.go"}}, ctx.buildConf) {
		t.Fatal("EmbedFiles overlay not detected")
	}
	if packageHasOverlay(&packages.Package{GoFiles: []string{"other.go"}}, ctx.buildConf) {
		t.Fatal("unrelated file reported as overlay")
	}

	// Build a minimal dependency + importer SSA pair to exercise lazy .llgoex loading
	// in writeExportMetaSidecar when depPkg.CacheHit == true and depPkg.exportMeta == nil.
	fset := token.NewFileSet()
	depFile, err := parser.ParseFile(fset, "dep.go", "package dep\nfunc Inc(x int) int { return x + 1 }\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	depInfo := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	depTypes, err := (&types.Config{}).Check("example.com/lazy/dep", fset, []*ast.File{depFile}, depInfo)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssa.NewProgram(fset, 0)
	depSSA := prog.CreatePackage(depTypes, []*ast.File{depFile}, depInfo, true)
	depSSA.Build()
	ctx.progSSA = prog

	depPkg := &aPackage{
		Package:     &packages.Package{ID: "example.com/lazy/dep", PkgPath: "example.com/lazy/dep", Name: "dep", Types: depTypes},
		SSA:         depSSA,
		Fingerprint: "deadbeef",
		CacheHit:    true,
	}
	depPaths := ctx.cacheManager.PackagePaths(ctx.targetTriple(), depPkg.PkgPath, depPkg.Fingerprint)
	if err := ctx.writeExportMetaSidecar(depPaths.Export, depPkg); err != nil {
		t.Fatalf("write dep export meta: %v", err)
	}
	// Clear in-memory exportMeta so the dependent's writeExportMetaSidecar must load it lazily.
	depPkg.exportMeta = nil
	ctx.pkgByID[depPkg.ID] = depPkg

	callerFile, err := parser.ParseFile(fset, "caller.go", "package caller\nfunc Use() int { return 1 }\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	callerInfo := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	callerTypes, err := (&types.Config{}).Check("example.com/lazy/caller", fset, []*ast.File{callerFile}, callerInfo)
	if err != nil {
		t.Fatal(err)
	}
	callerSSA := prog.CreatePackage(callerTypes, []*ast.File{callerFile}, callerInfo, true)
	callerSSA.Build()
	callerPkg := &aPackage{
		Package: &packages.Package{
			ID:      "example.com/lazy/caller",
			PkgPath: "example.com/lazy/caller",
			Name:    "caller",
			Types:   callerTypes,
			Imports: map[string]*packages.Package{depPkg.PkgPath: depPkg.Package},
		},
		SSA: callerSSA,
	}
	if err := ctx.writeExportMetaSidecar(filepath.Join(dir, "caller.llgoex"), callerPkg); err != nil {
		t.Fatalf("write caller export meta with lazy dep: %v", err)
	}

	// Exercise loadLinkSnapshotSidecar nil/empty/hit paths.
	ctx.loadLinkSnapshotSidecar(nil)
	ctx.loadLinkSnapshotSidecar(&aPackage{})
	depPkg.linkSnapshot = &packageLinkSnapshot{needAbiInit: 1}
	if err := ctx.writeLinkSnapshotSidecar(depPaths.LinkSnap, depPkg); err != nil {
		t.Fatal(err)
	}
	depPkg.linkSnapshot = nil
	ctx.loadLinkSnapshotSidecar(depPkg)
	if depPkg.linkSnapshot == nil || depPkg.linkSnapshot.needAbiInit != 1 {
		t.Fatalf("loadLinkSnapshotSidecar failed to restore snapshot: %+v", depPkg.linkSnapshot)
	}
}

func TestFrontendInlineHintTargetedRetryAndHelpers(t *testing.T) {
	// Structured error unwrapping.
	reqErr := &frontendSourceRequiredError{pkgIDs: []string{"example.com/a", "example.com/b"}}
	if !errors.Is(reqErr, errFrontendSourceRequired) || reqErr.Error() != errFrontendSourceRequired.Error() {
		t.Fatalf("unexpected error semantics: %v", reqErr)
	}
	var unwrapped *frontendSourceRequiredError
	if !errors.As(reqErr, &unwrapped) || len(unwrapped.pkgIDs) != 2 {
		t.Fatalf("errors.As failed: %v", unwrapped)
	}

	// frontendPosCache edge cases.
	var posCache frontendPosCache
	if posCache.snapshot(nil) != nil || posCache.lookup(nil, "a:1") != token.NoPos {
		t.Fatal("nil fset lookup should return NoPos")
	}
	fset := token.NewFileSet()
	if posCache.lookup(fset, "nocolon") != token.NoPos || posCache.lookup(fset, "a:-1") != token.NoPos || posCache.lookup(fset, "a:bad") != token.NoPos {
		t.Fatal("malformed position keys should return NoPos")
	}
	posFn := frontendPosition(fset)
	if posFn("missing:0") != token.NoPos {
		t.Fatal("missing file should return NoPos")
	}

	// frontendTypeLoader inline hint round-trip and corrupt/wrong-version handling.
	cacheDir := t.TempDir()
	l := &frontendTypeLoader{
		cache:         &cacheManager{root: cacheDir},
		inlineHintKey: "test-hint-key",
	}
	if got := l.loadInlineHint(); got != nil {
		t.Fatalf("missing hint should return nil, got %v", got)
	}
	l.storeInlineHint([]string{"b", "a"})
	if got := l.loadInlineHint(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("hint round-trip = %v, want [a b]", got)
	}
	// Wrong version rejected.
	if err := os.WriteFile(l.inlineHintPath(), []byte(`{"Version":1,"Packages":["a"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := l.loadInlineHint(); got != nil {
		t.Fatalf("wrong version hint = %v, want nil", got)
	}
	// Corrupt JSON rejected.
	if err := os.WriteFile(l.inlineHintPath(), []byte(`{bad`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := l.loadInlineHint(); got != nil {
		t.Fatalf("corrupt hint = %v, want nil", got)
	}
	// Empty hint key is a no-op.
	emptyL := &frontendTypeLoader{}
	emptyL.storeInlineHint([]string{"a"})
	if emptyL.loadInlineHint() != nil {
		t.Fatal("empty hint key should return nil")
	}

	// manifestBuilder.Build must sort BuildTags without mutating the shared backing slice.
	mb := newManifestBuilder()
	origTags := []string{"zeta", "alpha"}
	mb.common.BuildTags = origTags
	_ = mb.Build()
	if origTags[0] != "zeta" || origTags[1] != "alpha" {
		t.Fatalf("manifestBuilder.Build mutated BuildTags in place: %v", origTags)
	}

	// frontendCache helper edge cases: manifestTemplate sorting, empty/single-pkg precomputeLocals,
	// missing-local computeInputKeyReadOnly, single-pkg loadAll, frontendCallerSynthetic, markUncacheable(nil).
	fcCtx := &context{
		buildConf:    &Config{Tags: "ztag,atag", Goos: runtime.GOOS, Goarch: runtime.GOARCH},
		cacheManager: &cacheManager{root: cacheDir},
		pkgByID:      map[string]Package{},
		progSSA:      ssa.NewProgram(fset, 0),
	}
	fc := &frontendCache{
		ctx:    fcCtx,
		locals: map[string]string{},
	}
	fc.markUncacheable(nil)
	if tmpl := fc.manifestTemplate(); len(tmpl.common.BuildTags) >= 2 && tmpl.common.BuildTags[0] > tmpl.common.BuildTags[1] {
		t.Fatalf("manifestTemplate did not sort BuildTags: %v", tmpl.common.BuildTags)
	}
	depPkg := &packages.Package{ID: "example.com/missinglocal", PkgPath: "example.com/missinglocal", Name: "missinglocal"}
	topPkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/top",
		PkgPath: "example.com/top",
		Name:    "top",
		Imports: map[string]*packages.Package{depPkg.PkgPath: depPkg},
	}}
	// Empty precomputeLocals (already cached local) and missing-local read-only key.
	fc.locals[topPkg.ID] = "local-top"
	fc.precomputeLocals(nil)
	if key, ok := fc.computeInputKeyReadOnly(topPkg); ok || key != "" {
		t.Fatalf("computeInputKeyReadOnly with missing dep local = (%q, %v), want empty/false", key, ok)
	}
	delete(fc.locals, topPkg.ID)
	if key, ok := fc.computeInputKeyReadOnly(topPkg); ok || key != "" {
		t.Fatalf("computeInputKeyReadOnly with missing self local = (%q, %v), want empty/false", key, ok)
	}
	// Single-package precomputeLocals / loadAll (workers <= 1 path).
	single := &aPackage{Package: &packages.Package{
		ID:      "example.com/single",
		PkgPath: "example.com/single",
		Name:    "single",
		GoFiles: []string{filepath.Join(cacheDir, "missing.go")},
	}}
	fcCtx.pkgByID[single.ID] = single
	res := fc.loadAll([]Package{single})
	if len(res) != 1 || res[0].record != nil {
		t.Fatalf("loadAll(single missing file) = %+v", res)
	}
	if frontendCallerSynthetic(nil) || frontendCallerSynthetic(&ssa.Function{Synthetic: "other"}) {
		t.Fatal("frontendCallerSynthetic false positive")
	}
	if !frontendCallerSynthetic(&ssa.Function{Synthetic: "bound method wrapper for T.M"}) ||
		!frontendCallerSynthetic(&ssa.Function{Synthetic: "thunk for T.M"}) {
		t.Fatal("frontendCallerSynthetic missed bound/thunk wrapper")
	}

	// End-to-end targeted retry at -Os:
	// Build 1 (cold) uses dep1 only -> inline hint records only dep1, while dep2
	// is exported via .llgotypes.
	// Build 2 (warm edit) starts calling inlineable dep2.Add -> targeted retry
	// adds ONLY dep2 (preserving skipped_parse > 0!) and refreshes the hint so
	// Build 3 (warm edit) succeeds with 0 retries and skipped_parse > 0!
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("GOWORK", "off")
	t.Setenv(llgoBuildCache, "1")
	t.Setenv(llgoEscapeCache, "1")
	cache := t.TempDir()
	old := cacheRootFunc
	cacheRootFunc = func() string { return cache }
	t.Cleanup(func() { cacheRootFunc = old })

	modDir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(modDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/inlinehint\n\ngo 1.24\n")
	write("dep1/dep1.go", "package dep1\n\nfunc Twice(x int) int { return x + x }\n")
	write("dep2/dep2.go", "package dep2\n\nfunc Add(a, b int) int { return a + b }\n\n//go:noinline\nfunc Big(n int) int {\n\ts := 0\n\tfor i := 0; i < n; i++ {\n\t\ts += i\n\t}\n\treturn s\n}\n")
	write("dep3/dep3.go", "package dep3\n\n//go:noinline\nfunc StaySkipped(x int) int { return x + 1 }\n")
	write("main.go", "package main\n\nimport (\n\t\"example.com/inlinehint/dep1\"\n\t\"example.com/inlinehint/dep2\"\n\t\"example.com/inlinehint/dep3\"\n)\n\nfunc main() {\n\tprintln(dep1.Twice(3), dep2.Big(4), dep3.StaySkipped(10))\n}\n")

	buildAndRun := func(want string) int {
		t.Helper()
		cfg := NewDefaultConf(ModeBuild)
		cfg.OptLevel = optlevel.Os
		cfg.BuildParallelism = 4
		cfg.OutFile = filepath.Join(t.TempDir(), "app")
		if runtime.GOOS == "windows" {
			cfg.OutFile += ".exe"
		}
		cfg.BuildTrace = filepath.Join(t.TempDir(), "trace.json")
		if _, err := Build(Invocation{Args: []string{"."}, Dir: modDir, Config: cfg}); err != nil {
			t.Fatalf("build (%s): %v", want, err)
		}
		out, err := exec.Command(cfg.OutFile).CombinedOutput()
		if err != nil || !strings.Contains(string(out), want) {
			t.Fatalf("run (%s): err=%v out=%q", want, err, string(out))
		}
		data, err := os.ReadFile(cfg.BuildTrace)
		if err != nil {
			t.Fatal(err)
		}
		var events []buildTraceEvent
		if err := json.Unmarshal(data, &events); err != nil {
			t.Fatal(err)
		}
		skippedParse := 0
		for _, e := range events {
			if e.Name == "certify frontend exports" {
				if n, ok := e.Args["skipped_parse"].(float64); ok {
					skippedParse = int(n)
				}
			}
		}
		return skippedParse
	}

	// Build 1 (cold): dep1 is inline-used; dep2.Big and dep3.StaySkipped are noinline so dep2 & dep3 are NOT inline-needed yet.
	_ = buildAndRun("6 6 11")

	// Build 2 (warm edit): now calls inlineable dep2.Add -> triggers targeted retry for dep2 ONLY while dep3 stays skipped!
	write("main.go", "package main\n\nimport (\n\t\"example.com/inlinehint/dep1\"\n\t\"example.com/inlinehint/dep2\"\n\t\"example.com/inlinehint/dep3\"\n)\n\nfunc main() {\n\tprintln(dep1.Twice(4), dep2.Add(10, 5), dep3.StaySkipped(20))\n}\n")
	if sp := buildAndRun("8 15 21"); sp == 0 {
		t.Fatal("targeted retry unexpectedly fell back to full source (skipped_parse=0)")
	}

	// Build 3 (warm edit): hint now includes dep2 -> immediate hit without retry and dep3 stays skipped.
	write("main.go", "package main\n\nimport (\n\t\"example.com/inlinehint/dep1\"\n\t\"example.com/inlinehint/dep2\"\n\t\"example.com/inlinehint/dep3\"\n)\n\nfunc main() {\n\tprintln(dep1.Twice(5), dep2.Add(20, 7), dep3.StaySkipped(30))\n}\n")
	if sp := buildAndRun("10 27 31"); sp == 0 {
		t.Fatal("warm build after hint update had skipped_parse=0")
	}
}

func TestFingerprintDepfileIncludeHelpers(t *testing.T) {
	// sortDeps tie-breakers (equal ID, equal Version -> Fingerprint).
	sorted := sortDeps([]depEntry{
		{ID: "a", Version: "v2", Fingerprint: "2"},
		{ID: "a", Version: "v1", Fingerprint: "b"},
		{ID: "a", Version: "v1", Fingerprint: "a"},
	})
	if sorted[0].Fingerprint != "a" || sorted[1].Fingerprint != "b" || sorted[2].Version != "v2" {
		t.Fatalf("unexpected sortDeps order: %+v", sorted)
	}

	// decodeManifest edge cases (empty, oversized, invalid YAML).
	if data, err := decodeManifest("   "); err != nil || data.Package != nil {
		t.Fatalf("decodeManifest(empty) = (%+v, %v)", data, err)
	}
	if _, err := decodeManifest(strings.Repeat("a", maxManifestSize+1)); err == nil {
		t.Fatal("decodeManifest(oversized) expected error")
	}
	if _, err := decodeManifest(":\n  - [unclosed"); err == nil {
		t.Fatal("decodeManifest(invalid yaml) expected error")
	}

	// digestEmbedFiles success + error.
	dir := t.TempDir()
	emb1 := filepath.Join(dir, "b.txt")
	emb2 := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(emb1, []byte("beta"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(emb2, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	embs, err := digestEmbedFiles([]string{emb1, emb2})
	if err != nil || len(embs) != 2 || embs[0].Path != emb2 {
		t.Fatalf("digestEmbedFiles = (%+v, %v)", embs, err)
	}
	if _, err := digestEmbedFiles([]string{filepath.Join(dir, "missing.txt")}); err == nil {
		t.Fatal("digestEmbedFiles(missing) expected error")
	}

	// depfileEnvIdentity with empty workDir and resolveSearchDir edge cases.
	ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}}
	if envID := depfileEnvIdentity(ctx, "", nil, []string{"-O2"}); envID == "" {
		t.Fatal("depfileEnvIdentity returned empty")
	}
	if resolveSearchDir("", "") != "" || resolveSearchDir(dir, "") != filepath.Clean(dir) || resolveSearchDir(dir, "sub") != filepath.Join(dir, "sub") {
		t.Fatal("resolveSearchDir edge cases failed")
	}

	// collectIncludeSearchDirs with relative input, two-arg flags, joined prefixes, and env vars.
	sep := string(os.PathListSeparator)
	env := []string{
		"INCLUDE=" + sep + "inc_env",
		"CPATH=" + sep + "cpath_env",
	}
	dirs := collectIncludeSearchDirs("sub/src.c", dir, env, []string{
		"-I", "inc1",
		"-isystem", "sys1",
		"-Iinc2",
		"--include-directory=inc3",
		"-I",
	})
	if len(dirs) < 5 {
		t.Fatalf("collectIncludeSearchDirs returned too few dirs: %v", dirs)
	}

	// parseIncludeToken & extractIncludeTargets edge cases (including no trailing newline).
	for _, bad := range []string{"", "a", "foo", `"unclosed`, `<unclosed`} {
		if tok, _ := parseIncludeToken(bad); tok != "" {
			t.Fatalf("parseIncludeToken(%q) = %q, want empty", bad, tok)
		}
	}
	hdr := "#include \"local.h\"\n#include_next <sys.h>\n#import \"imp.h\"\n#if __has_include(\"opt.h\") && __has_include_next(<opt2.h>)\n#include \"eof.h\""
	quoted, angled := extractIncludeTargets([]byte(hdr))
	if len(quoted) != 4 || len(angled) != 2 {
		t.Fatalf("extractIncludeTargets = (%v, %v)", quoted, angled)
	}

	// collectShadowingMissingPaths empty + populated.
	if got := collectShadowingMissingPaths(nil, nil, nil); got != nil {
		t.Fatalf("collectShadowingMissingPaths(empty) = %v, want nil", got)
	}
	incDir := filepath.Join(dir, "inc")
	subDir := filepath.Join(incDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	depHdr := filepath.Join(subDir, "used.h")
	if err := os.WriteFile(depHdr, []byte("#include \"../parent_miss.h\"\n#include \"./dot.h\"\n#include \"nested.h\"\n#include <angle_miss.h>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rawDep, _ := os.ReadFile(depHdr)
	missing := collectShadowingMissingPaths(
		[]string{dir, incDir},
		[]string{"", depHdr},
		map[string][]byte{depHdr: rawDep},
	)
	if len(missing) == 0 {
		t.Fatal("collectShadowingMissingPaths returned no candidates")
	}

	// parseMakeDepfileForOS (both Unix and Windows backslash/space/$$ rules).
	if got := parseMakeDepfileForOS("no-colon-here", false); got != nil {
		t.Fatalf("missing colon = %v, want nil", got)
	}
	unixDeps := parseMakeDepfileForOS("out.o: a.c \\\n  dir\\ with\\ space/b.h $$dollar.h\n", false)
	if len(unixDeps) != 3 {
		t.Fatalf("unix parseMakeDepfileForOS = %v", unixDeps)
	}
	winDeps := parseMakeDepfileForOS("out.o: C:\\Users\\foo\\a.h C:\\dir\\ with\\ space\\b.h C:\\even\\\\ c.h C:\\odd\\\\\\ d.h #end", true)
	if len(winDeps) < 4 {
		t.Fatalf("windows parseMakeDepfileForOS = %v", winDeps)
	}
}

func TestFrontendTypesAndPrepareEdgeCases(t *testing.T) {
	cacheDir := t.TempDir()
	srcDir := t.TempDir()
	fset := token.NewFileSet()
	srcPath := filepath.Join(srcDir, "p.go")
	srcText := "package p\n\ntype MyInt int\nconst C = 1\nvar V = 2\nfunc F() int { return C + V }\n"
	if err := os.WriteFile(srcPath, []byte(srcText), 0o600); err != nil {
		t.Fatal(err)
	}
	parsedFile, err := parser.ParseFile(fset, srcPath, srcText, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	checked, err := (&types.Config{}).Check("example.com/edgep", fset, []*ast.File{parsedFile}, info)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &context{
		buildConf: &Config{
			Goos:           runtime.GOOS,
			Goarch:         runtime.GOARCH,
			Verbose:        true,
			GlobalRewrites: map[string]Rewrites{"example.com/rewritten": {"X": "1"}},
		},
		cacheManager: &cacheManager{root: cacheDir},
		pkgByID:      map[string]Package{},
		progSSA:      ssa.NewProgram(fset, 0),
	}
	loader := &frontendTypeLoader{
		ctx:            ctx,
		cache:          ctx.cacheManager,
		namespace:      "ns",
		keys:           map[string]string{},
		exports:        map[string]*frontendTypeExport{},
		selected:       map[string]bool{},
		exportDataHits: map[string]bool{},
	}

	// GlobalRewrites makes package ineligible.
	if loader.eligible(&packages.Package{ID: "example.com/rewritten", PkgPath: "example.com/rewritten", Name: "rewritten"}) {
		t.Fatal("package with GlobalRewrites must not be eligible")
	}

	// load() rejects dep.ID != dep.PkgPath and missing source files (empty key).
	pkgWithTestDep := &packages.Package{
		ID:      "example.com/edgep",
		PkgPath: "example.com/edgep",
		Name:    "p",
		Imports: map[string]*packages.Package{"d": {ID: "d [d.test]", PkgPath: "d"}},
	}
	if loader.load(pkgWithTestDep, fset) {
		t.Fatal("load should reject dep with ID != PkgPath")
	}
	pkgMissingFile := &packages.Package{
		ID:              "example.com/missingfile",
		PkgPath:         "example.com/missingfile",
		Name:            "missingfile",
		CompiledGoFiles: []string{filepath.Join(srcDir, "does_not_exist.go")},
	}
	if loader.load(pkgMissingFile, fset) {
		t.Fatal("load should reject missing source file")
	}

	// Build companion archive + frontend record for legacy JSON Package format + Syntax Defs population.
	depTypes := types.NewPackage("example.com/edgedep", "edgedep")
	depTypes.MarkComplete()
	p := &packages.Package{
		ID:              "example.com/edgep",
		PkgPath:         "example.com/edgep",
		Name:            "p",
		CompiledGoFiles: []string{srcPath},
		Syntax:          []*ast.File{parsedFile},
		Imports:         map[string]*packages.Package{"example.com/edgedep": {ID: "example.com/edgedep", PkgPath: "example.com/edgedep", Name: "edgedep", Types: depTypes}},
	}
	key := loader.key(p, map[string]bool{})
	if key == "" {
		t.Fatal("expected non-empty key")
	}
	archFP := "archfp-edge"
	paths := ctx.cacheManager.PackagePaths(ctx.targetTriple(), p.PkgPath, archFP)
	if err := os.MkdirAll(filepath.Dir(paths.Archive), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Archive, []byte("!<arch>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	feDir := filepath.Join(cacheDir, "frontend")
	if err := os.MkdirAll(feDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(feDir, "fe-edge.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	writeTypeRecord := func(rec frontendTypeExport) {
		t.Helper()
		payload, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		env, err := json.Marshal(frontendEnvelope{Digest: frontendDigest(payload), Payload: payload})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(loader.path(key), env, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// exportdata with wrong PkgID -> false.
	writeTypeRecord(frontendTypeExport{
		Version: frontendExportVersion, Input: key, Archive: archFP, Frontend: "fe-edge",
		PkgID: "wrong/id", PkgPath: p.PkgPath, TypesFormat: frontendTypesFormatExportData,
	})
	if loader.load(p, fset) {
		t.Fatal("load accepted mismatched PkgID")
	}
	// exportdata with missing .llgotypes sidecar -> false.
	writeTypeRecord(frontendTypeExport{
		Version: frontendExportVersion, Input: key, Archive: archFP, Frontend: "fe-edge",
		PkgID: p.ID, PkgPath: p.PkgPath, TypesFormat: frontendTypesFormatExportData,
	})
	if loader.load(p, fset) {
		t.Fatal("load accepted missing .llgotypes sidecar")
	}

	// Legacy JSON PackageExport format WITH Syntax -> populates pkg.TypesInfo.Defs for Type/Const/Var/Func!
	idx := frontcache.NewTypes(map[*types.Package]string{checked: p.ID, depTypes: "example.com/edgedep"}, fset)
	idx.AddPackage(checked, info)
	pkgExp := frontcache.ExportPackage(checked, idx, fset)
	writeTypeRecord(frontendTypeExport{
		Version: frontendExportVersion, Input: key, Archive: archFP, Frontend: "fe-edge",
		Package: pkgExp,
	})
	if !loader.load(p, fset) {
		t.Fatal("legacy JSON PackageExport load failed")
	}
	if len(p.TypesInfo.Defs) < 4 {
		t.Fatalf("expected Defs populated from Syntax, got %d", len(p.TypesInfo.Defs))
	}

	// capture() skips packages with TLS/GLS var (Locality != LocalityNone).
	tlsPkg := types.NewPackage("example.com/tlspkg", "tlspkg")
	tlsVar := types.NewVar(token.NoPos, tlsPkg, "slot", types.Typ[types.Uintptr])
	tlsVar.SetLocality(directive.LocalityThread)
	tlsPkg.Scope().Insert(tlsVar)
	tlsPkg.MarkComplete()
	loader.capture(&aPackage{
		Package: &packages.Package{ID: "example.com/tlspkg", PkgPath: "example.com/tlspkg", Name: "tlspkg", Types: tlsPkg},
	}, &frontendExport{Input: "in"}, idx)
	if loader.exports["example.com/tlspkg"] != nil {
		t.Fatal("capture should skip package with TLS variable")
	}

	// store() writes missing .llgotypes sidecar and surfaces writePackageTypesSidecar error on nil Types.
	loader.exports["example.com/storeerr"] = &frontendTypeExport{
		Version: frontendExportVersion, Input: "storeerr-key", Frontend: "fe",
		PkgID: "example.com/storeerr", PkgPath: "example.com/storeerr", TypesFormat: frontendTypesFormatExportData,
	}
	if err := loader.store(&aPackage{
		Package:     &packages.Package{ID: "example.com/storeerr", PkgPath: "example.com/storeerr", Name: "storeerr"},
		Fingerprint: "fp-storeerr",
	}); err == nil {
		t.Fatal("store with nil Types expected error")
	}

	// frontendCache.loadByKey("") -> nil, and bindFunctions duplicate-ID ambiguity -> uncacheable.
	fc := &frontendCache{
		ctx:     ctx,
		records: map[string]*frontendExport{},
		owners:  map[*ssa.Package]Package{},
		types:   idx,
	}
	if fc.loadByKey(cacheDir, "") != nil {
		t.Fatal("loadByKey empty key should return nil")
	}
	ssaPkg := ctx.progSSA.CreatePackage(checked, []*ast.File{parsedFile}, info, true)
	ssaPkg.Build()
	ownerPkg := &aPackage{Package: p, SSA: ssaPkg}
	fc.owners[ssaPkg] = ownerPkg
	dupSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	f1 := &ssa.Function{Prog: ctx.progSSA, Pkg: ssaPkg, Signature: dupSig}
	f2 := &ssa.Function{Prog: ctx.progSSA, Pkg: ssaPkg, Signature: dupSig}
	f3 := &ssa.Function{Prog: ctx.progSSA, Pkg: ssaPkg, Signature: dupSig}
	fc.methods = []*ssa.Function{f1, f2, f3}
	fc.bindFunctions()
	if !fc.uncacheable[ownerPkg] {
		t.Fatal("duplicate function identity should mark owner uncacheable")
	}

	// writeExportMetaSidecar with nil dep / missing pkgByID dep skips cleanly.
	pWithNilDep := &aPackage{
		Package: &packages.Package{
			ID:      "example.com/edgep",
			PkgPath: "example.com/edgep",
			Name:    "p",
			Imports: map[string]*packages.Package{
				"nildep":     nil,
				"missingdep": {ID: "example.com/not-in-pkgbyid", PkgPath: "example.com/not-in-pkgbyid"},
			},
		},
		SSA: ssaPkg,
	}
	if err := ctx.writeExportMetaSidecar(filepath.Join(cacheDir, "sidecar.llgoex"), pWithNilDep); err != nil {
		t.Fatalf("writeExportMetaSidecar unexpected err: %v", err)
	}
}

func TestRuntimeABIBlockersAndBoundaryErrors(t *testing.T) {
	blockers := runtimeABIBlockers(&RuntimeABI{
		GoType:             RuntimeTypeLayout{Size: 48, Align: 8, Fields: []RuntimeFieldLayout{{Name: "Size", Offset: 0, Size: 8}}},
		LLGoType:           RuntimeTypeLayout{Size: 56, Align: 8, Fields: []RuntimeFieldLayout{{Name: "Size", Offset: 0, Size: 8}}},
		GoFuncValueBytes:   8,
		LLGoFuncValueBytes: 16,
	})
	if len(blockers) < 5 {
		t.Fatalf("expected size + funcval + standing blockers, got %v", blockers)
	}

	dir := t.TempDir()
	nonStruct := filepath.Join(dir, "nonstruct.go")
	if err := os.WriteFile(nonStruct, []byte("package p\ntype Alias int\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := goRuntimeMetadataType(filepath.Join(dir, "missing.go"), "Type"); err == nil {
		t.Fatal("expected missing file error")
	}
	if _, err := goRuntimeMetadataType(nonStruct, "Missing"); err == nil {
		t.Fatal("expected missing type error")
	}
	if _, err := goRuntimeMetadataType(nonStruct, "Alias"); err == nil {
		t.Fatal("expected non-struct error")
	}
	if _, err := runtimeMetadataType(nonStruct, "Missing"); err == nil {
		t.Fatal("expected runtimeMetadataType missing type error")
	}
	if _, err := runtimeMetadataType(nonStruct, "Alias"); err == nil {
		t.Fatal("expected runtimeMetadataType non-struct error")
	}
}
