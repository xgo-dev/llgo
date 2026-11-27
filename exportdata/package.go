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

package exportdata

import (
	"bytes"
	"fmt"
	"go/constant"
	"go/token"
	"io"
	"sort"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/types"
)

const (
	typeTagNil byte = iota
	typeTagBasic
	typeTagPointer
	typeTagSlice
	typeTagArray
	typeTagMap
	typeTagChan
	typeTagTuple
	typeTagSignature
	typeTagStruct
	typeTagInterface
	typeTagUnion
	typeTagNamedRef
	typeTagAliasRef
	typeTagTypeParamRef
	typeTagSIMDVec
	typeTagNamedInst
	typeTagAliasInst
)

const (
	objTagConst byte = iota + 1
	objTagVar
	objTagFunc
	objTagTypeName
)

const (
	constTagNil byte = iota
	constTagBool
	constTagString
	constTagInt
	constTagFloat
	constTagComplex
)

// Write writes the binary export data for pkg to out, mirroring
// golang.org/x/tools/go/gcexportdata.Write.
func Write(out io.Writer, fset *token.FileSet, pkg *types.Package) error {
	raw, err := marshalPackage(pkg)
	if err != nil {
		return err
	}
	_, err = out.Write(raw)
	return err
}

// Read reads export data from in, constructs the types.Package for path, and
// inserts it into imports (mirroring golang.org/x/tools/go/gcexportdata.Read).
func Read(in io.Reader, fset *token.FileSet, imports map[string]*types.Package, path string) (*types.Package, error) {
	raw, err := readAll(in)
	if err != nil {
		return nil, err
	}
	pkg, err := unmarshalPackage(raw, imports)
	if err != nil {
		return nil, err
	}
	if path != "" && imports != nil {
		if _, exists := imports[path]; !exists {
			imports[path] = pkg
		}
	}
	return pkg, nil
}

// WriteBundle writes the binary export data for a slice of packages to out,
// mirroring golang.org/x/tools/go/gcexportdata.WriteBundle.
func WriteBundle(out io.Writer, fset *token.FileSet, pkgs []*types.Package) error {
	e := newEncoder()
	e.writeUvarint(uint64(len(pkgs)))
	for _, pkg := range pkgs {
		raw, err := marshalPackage(pkg)
		if err != nil {
			return err
		}
		e.writeBytes(raw)
	}
	_, err := out.Write(e.finish(kindBundle))
	return err
}

// ReadBundle reads a bundle of packages written by WriteBundle, reusing and
// populating imports (mirroring golang.org/x/tools/go/gcexportdata.ReadBundle).
// Package type identities are resolved independently of the order packages
// appear in the bundle.
func ReadBundle(in io.Reader, fset *token.FileSet, imports map[string]*types.Package) ([]*types.Package, error) {
	raw, err := readAll(in)
	if err != nil {
		return nil, err
	}
	d, err := newDecoder(raw, kindBundle)
	if err != nil {
		return nil, err
	}
	n, err := d.readUvarint()
	if err != nil || n > uint64(len(raw)) {
		return nil, errCorrupt
	}
	if imports == nil {
		imports = make(map[string]*types.Package)
	}
	readers := make([]*pkgReader, n)
	bundle := newBundleState()
	// Phase 1: parse headers and establish shared *types.Package shells in
	// imports before any package table or declaration is decoded.
	for i := uint64(0); i < n; i++ {
		blob, err := d.readBytes()
		if err != nil {
			return nil, err
		}
		dec, err := newDecoder(blob, kindPackage)
		if err != nil {
			return nil, err
		}
		pr := &pkgReader{
			dec:     dec,
			imports: imports,
			bundle:  bundle,
		}
		if err := pr.readHeader(); err != nil {
			return nil, err
		}
		pr.initPackage()
		readers[i] = pr
	}
	// Phase 2: resolve every package's dependency table against the shared
	// imports map so cross-package references always share one *types.Package.
	pkgIndex := make(map[*types.Package]int, len(readers))
	for i, pr := range readers {
		if err := pr.readPackageTable(); err != nil {
			return nil, err
		}
		if pr.pkg != nil {
			if _, exists := pkgIndex[pr.pkg]; !exists {
				pkgIndex[pr.pkg] = i
			}
		}
	}
	// Phase 3: decode package bodies in dependency-first topological order
	// (breaking any cycles deterministically by bundle index) so owning
	// packages populate their types before importers reference them.
	visited := make([]bool, len(readers))
	order := make([]int, 0, len(readers))
	var visit func(int)
	visit = func(i int) {
		if visited[i] {
			return
		}
		visited[i] = true
		for _, dep := range readers[i].pkgs {
			if depIdx, ok := pkgIndex[dep]; ok && depIdx != i {
				visit(depIdx)
			}
		}
		order = append(order, i)
	}
	for i := range readers {
		visit(i)
	}
	for _, idx := range order {
		if err := readers[idx].decodeBody(); err != nil {
			return nil, err
		}
	}
	pkgs := make([]*types.Package, len(readers))
	for i, pr := range readers {
		pkgs[i] = pr.pkg
	}
	return pkgs, nil
}

// NewImporter returns a new instance of the types.ImporterFrom interface that
// resolves packages from imports (mirroring golang.org/x/tools/go/gcexportdata.NewImporter).
func NewImporter(fset *token.FileSet, imports map[string]*types.Package) types.ImporterFrom {
	return importer{fset: fset, imports: imports}
}

type importer struct {
	fset    *token.FileSet
	imports map[string]*types.Package
}

func (imp importer) Import(importPath string) (*types.Package, error) {
	return imp.ImportFrom(importPath, "", 0)
}

func (imp importer) ImportFrom(importPath, srcDir string, mode types.ImportMode) (*types.Package, error) {
	if mode != 0 {
		return nil, fmt.Errorf("exportdata: unsupported import mode %d", mode)
	}
	if importPath == "unsafe" {
		return types.Unsafe, nil
	}
	if imp.imports != nil {
		if pkg := imp.imports[importPath]; pkg != nil && pkg.Complete() {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("exportdata: package %q not found in imports", importPath)
}

func marshalPackage(pkg *types.Package) ([]byte, error) {
	if pkg == nil {
		return nil, fmt.Errorf("exportdata: nil package")
	}
	pw := &pkgWriter{
		enc:        newEncoder(),
		pkg:        pkg,
		pkgIndex:   make(map[*types.Package]uint64),
		namedIndex: make(map[*types.Named]uint64),
		aliasIndex: make(map[*types.Alias]uint64),
		tpIndex:    make(map[*types.TypeParam]uint64),
	}
	pw.encode()
	return pw.enc.finish(kindPackage), nil
}

func unmarshalPackage(raw []byte, imports map[string]*types.Package) (*types.Package, error) {
	dec, err := newDecoder(raw, kindPackage)
	if err != nil {
		return nil, err
	}
	pr := &pkgReader{
		dec:     dec,
		imports: imports,
	}
	return pr.decode()
}

type pkgWriter struct {
	enc *encoder
	pkg *types.Package

	pkgs       []*types.Package
	pkgIndex   map[*types.Package]uint64
	nameds     []*types.Named
	namedIndex map[*types.Named]uint64
	aliases    []*types.Alias
	aliasIndex map[*types.Alias]uint64
	tparams    []*types.TypeParam
	tpIndex    map[*types.TypeParam]uint64
}

func (w *pkgWriter) internPkg(p *types.Package) uint64 {
	if p == nil {
		return 0
	}
	if idx, ok := w.pkgIndex[p]; ok {
		return idx
	}
	idx := uint64(len(w.pkgs) + 1) // 1-based; 0 = nil
	w.pkgs = append(w.pkgs, p)
	w.pkgIndex[p] = idx
	return idx
}

func (w *pkgWriter) registerType(t types.Type, seen map[types.Type]bool) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	switch u := t.(type) {
	case *types.Basic:
	case *types.Pointer:
		w.registerType(u.Elem(), seen)
	case *types.Slice:
		w.registerType(u.Elem(), seen)
	case *types.Array:
		w.registerType(u.Elem(), seen)
	case *types.Map:
		w.registerType(u.Key(), seen)
		w.registerType(u.Elem(), seen)
	case *types.Chan:
		w.registerType(u.Elem(), seen)
	case *types.Tuple:
		for i := 0; i < u.Len(); i++ {
			w.registerVar(u.At(i), seen)
		}
	case *types.Signature:
		w.registerVar(u.Recv(), seen)
		for i := 0; i < u.RecvTypeParams().Len(); i++ {
			w.registerType(u.RecvTypeParams().At(i), seen)
		}
		for i := 0; i < u.TypeParams().Len(); i++ {
			w.registerType(u.TypeParams().At(i), seen)
		}
		w.registerType(u.Params(), seen)
		w.registerType(u.Results(), seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			w.registerVar(u.Field(i), seen)
		}
	case *types.Interface:
		for i := 0; i < u.NumExplicitMethods(); i++ {
			m := u.ExplicitMethod(i)
			w.internPkg(m.Pkg())
			w.registerType(m.Type(), seen)
		}
		for i := 0; i < u.NumEmbeddeds(); i++ {
			w.registerType(u.EmbeddedType(i), seen)
		}
	case *types.Union:
		for i := 0; i < u.Len(); i++ {
			w.registerType(u.Term(i).Type(), seen)
		}
	case *types.Named:
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			w.registerType(orig, seen)
			for i := 0; i < u.TypeArgs().Len(); i++ {
				w.registerType(u.TypeArgs().At(i), seen)
			}
			return
		}
		if _, ok := w.namedIndex[u]; !ok {
			idx := uint64(len(w.nameds))
			w.nameds = append(w.nameds, u)
			w.namedIndex[u] = idx
			if u.Obj() != nil {
				w.internPkg(u.Obj().Pkg())
			}
			for i := 0; i < u.TypeParams().Len(); i++ {
				w.registerType(u.TypeParams().At(i), seen)
			}
			w.registerType(u.Underlying(), seen)
			for i := 0; i < u.NumMethods(); i++ {
				m := u.Method(i)
				w.internPkg(m.Pkg())
				w.registerType(m.Type(), seen)
			}
		}
	case *types.Alias:
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			w.registerType(orig, seen)
			for i := 0; i < u.TypeArgs().Len(); i++ {
				w.registerType(u.TypeArgs().At(i), seen)
			}
			return
		}
		if _, ok := w.aliasIndex[u]; !ok {
			idx := uint64(len(w.aliases))
			w.aliases = append(w.aliases, u)
			w.aliasIndex[u] = idx
			if u.Obj() != nil {
				w.internPkg(u.Obj().Pkg())
			}
			for i := 0; i < u.TypeParams().Len(); i++ {
				w.registerType(u.TypeParams().At(i), seen)
			}
			w.registerType(u.Rhs(), seen)
		}
	case *types.TypeParam:
		if _, ok := w.tpIndex[u]; !ok {
			idx := uint64(len(w.tparams))
			w.tparams = append(w.tparams, u)
			w.tpIndex[u] = idx
			if u.Obj() != nil {
				w.internPkg(u.Obj().Pkg())
			}
			w.registerType(u.Constraint(), seen)
		}
	case *types.SIMDVec:
	}
}

func (w *pkgWriter) registerVar(v *types.Var, seen map[types.Type]bool) {
	if v == nil {
		return
	}
	w.internPkg(v.Pkg())
	w.registerType(v.Type(), seen)
}

func (w *pkgWriter) encode() {
	// Pass 1: discover all referenced packages, Named, Alias, TypeParam shells.
	w.internPkg(w.pkg)
	for _, imp := range w.pkg.Imports() {
		w.internPkg(imp)
	}
	seen := make(map[types.Type]bool)
	names := w.pkg.Scope().Names()
	sort.Strings(names)
	var objs []types.Object
	for _, name := range names {
		obj := w.pkg.Scope().Lookup(name)
		if obj == nil {
			continue
		}
		switch o := obj.(type) {
		case *types.Const, *types.Var, *types.Func, *types.TypeName:
			objs = append(objs, o)
			w.internPkg(o.Pkg())
			w.registerType(o.Type(), seen)
		}
	}

	// Order generic Named origins ahead of non-generic Nameds so instantiated
	// references during decode always see populated origins.
	if len(w.nameds) > 1 {
		ordered := make([]*types.Named, 0, len(w.nameds))
		for _, n := range w.nameds {
			if n.TypeParams().Len() > 0 {
				ordered = append(ordered, n)
			}
		}
		for _, n := range w.nameds {
			if n.TypeParams().Len() == 0 {
				ordered = append(ordered, n)
			}
		}
		w.nameds = ordered
		for i, n := range w.nameds {
			w.namedIndex[n] = uint64(i)
		}
	}

	// Pass 2: emit header & referenced package table.
	w.enc.writeStr(w.pkg.Path())
	w.enc.writeStr(w.pkg.Name())
	w.enc.writeStr(w.pkg.GoVersion())
	w.enc.writeBool(w.pkg.Complete())
	w.enc.writeByte(byte(w.pkg.Kind()))
	w.enc.writeStr(w.pkg.KindParam())
	w.enc.writeBool(w.pkg.SkipAll())
	w.enc.writeStringSlice(w.pkg.SkipNames())

	w.enc.writeUvarint(uint64(len(w.pkgs)))
	for _, p := range w.pkgs {
		w.enc.writeStr(p.Path())
		w.enc.writeStr(p.Name())
	}

	w.enc.writeUvarint(uint64(len(w.pkg.Imports())))
	for _, imp := range w.pkg.Imports() {
		w.enc.writeUvarint(w.internPkg(imp))
	}

	// Emit TypeParam shells.
	w.enc.writeUvarint(uint64(len(w.tparams)))
	for _, tp := range w.tparams {
		var pkgID uint64
		var name string
		if tp.Obj() != nil {
			pkgID = w.internPkg(tp.Obj().Pkg())
			name = tp.Obj().Name()
		}
		w.enc.writeUvarint(pkgID)
		w.enc.writeStr(name)
		w.enc.writeUvarint(uint64(tp.Index()))
	}

	// Emit Named shells.
	w.enc.writeUvarint(uint64(len(w.nameds)))
	for _, n := range w.nameds {
		var pkgID uint64
		var name string
		if n.Obj() != nil {
			pkgID = w.internPkg(n.Obj().Pkg())
			name = n.Obj().Name()
		}
		w.enc.writeUvarint(pkgID)
		w.enc.writeStr(name)
		w.enc.writeByte(byte(n.CallConv()))
	}

	// Emit Alias shells.
	w.enc.writeUvarint(uint64(len(w.aliases)))
	for _, a := range w.aliases {
		var pkgID uint64
		var name string
		if a.Obj() != nil {
			pkgID = w.internPkg(a.Obj().Pkg())
			name = a.Obj().Name()
		}
		w.enc.writeUvarint(pkgID)
		w.enc.writeStr(name)
		w.enc.writeByte(byte(a.CallConv()))
	}

	// Emit TypeParam constraints as length-prefixed sub-blobs so decoding can
	// resolve Alias and Named bodies on demand in dependency order.
	for _, tp := range w.tparams {
		w.writeSubBlob(func() {
			w.writeType(tp.Constraint())
		})
	}

	// Emit Named definitions (typeParams, underlying, methods) as sub-blobs.
	for _, n := range w.nameds {
		w.writeSubBlob(func() {
			tps := n.TypeParams()
			w.enc.writeUvarint(uint64(tps.Len()))
			for i := 0; i < tps.Len(); i++ {
				w.enc.writeUvarint(w.tpIndex[tps.At(i)])
			}
			w.writeType(n.Underlying())
			w.enc.writeUvarint(uint64(n.NumMethods()))
			for i := 0; i < n.NumMethods(); i++ {
				m := n.Method(i)
				w.enc.writeUvarint(w.internPkg(m.Pkg()))
				w.enc.writeStr(m.Name())
				w.writeType(m.Type())
				w.writeFuncMeta(m)
			}
		})
	}

	// Emit Alias definitions (typeParams, rhs) as sub-blobs.
	for _, a := range w.aliases {
		w.writeSubBlob(func() {
			tps := a.TypeParams()
			w.enc.writeUvarint(uint64(tps.Len()))
			for i := 0; i < tps.Len(); i++ {
				w.enc.writeUvarint(w.tpIndex[tps.At(i)])
			}
			w.writeType(a.Rhs())
		})
	}

	// Emit package scope objects.
	w.enc.writeUvarint(uint64(len(objs)))
	for _, obj := range objs {
		switch o := obj.(type) {
		case *types.Const:
			w.enc.writeByte(objTagConst)
			w.enc.writeStr(o.Name())
			w.writeType(o.Type())
			w.writeConst(o.Val())
		case *types.Var:
			w.enc.writeByte(objTagVar)
			w.enc.writeStr(o.Name())
			w.enc.writeByte(byte(o.Kind()))
			w.writeType(o.Type())
			w.writeVarMeta(o)
		case *types.Func:
			w.enc.writeByte(objTagFunc)
			w.enc.writeStr(o.Name())
			w.writeType(o.Type())
			w.writeFuncMeta(o)
		case *types.TypeName:
			w.enc.writeByte(objTagTypeName)
			w.enc.writeStr(o.Name())
			w.enc.writeBool(o.IsAlias())
			w.writeType(o.Type())
		}
	}
}

func (w *pkgWriter) writeSubBlob(fn func()) {
	origBuf := w.enc.buf
	w.enc.buf = bytes.Buffer{}
	fn()
	sub := w.enc.buf.Bytes()
	w.enc.buf = origBuf
	w.enc.writeBytes(sub)
}

const (
	funcMetaLinkName uint64 = 1 << iota
	funcMetaExportName
	funcMetaWasmImport
	funcMetaWasmExport
	funcMetaClosureEnv
	funcMetaNoInline
	funcMetaNoInterface
	funcMetaNoEscape
	funcMetaUintptrEscapes
	funcMetaNoSplit

	funcMetaAllFlags = funcMetaLinkName | funcMetaExportName | funcMetaWasmImport |
		funcMetaWasmExport | funcMetaClosureEnv | funcMetaNoInline | funcMetaNoInterface |
		funcMetaNoEscape | funcMetaUintptrEscapes | funcMetaNoSplit
)

func (w *pkgWriter) writeFuncMeta(fn *types.Func) {
	var flags uint64
	if fn != nil {
		if fn.LinkName() != "" {
			flags |= funcMetaLinkName
		}
		if fn.ExportName() != "" {
			flags |= funcMetaExportName
		}
		if fn.WasmImport() != nil {
			flags |= funcMetaWasmImport
		}
		if fn.WasmExport() != "" {
			flags |= funcMetaWasmExport
		}
		if fn.HasClosureEnv() {
			flags |= funcMetaClosureEnv
		}
		if fn.NoInline() {
			flags |= funcMetaNoInline
		}
		if fn.NoInterface() {
			flags |= funcMetaNoInterface
		}
		if fn.NoEscape() {
			flags |= funcMetaNoEscape
		}
		if fn.UintptrEscapes() {
			flags |= funcMetaUintptrEscapes
		}
		if fn.NoSplit() {
			flags |= funcMetaNoSplit
		}
	}
	w.enc.writeUvarint(flags)
	if flags&funcMetaLinkName != 0 {
		w.enc.writeStr(fn.LinkName())
	}
	if flags&funcMetaExportName != 0 {
		w.enc.writeStr(fn.ExportName())
	}
	if flags&funcMetaWasmImport != 0 {
		wi := fn.WasmImport()
		w.enc.writeStr(wi.Module)
		w.enc.writeStr(wi.Name)
	}
	if flags&funcMetaWasmExport != 0 {
		w.enc.writeStr(fn.WasmExport())
	}
}

const (
	varMetaLinkName byte = 1 << iota
	varMetaExportName
	varMetaLocality

	varMetaAllFlags = varMetaLinkName | varMetaExportName | varMetaLocality
)

func validPackageKind(b byte) bool {
	return b <= byte(directive.PkgLinkExtern)
}

func validCallConv(b byte) bool {
	return b <= byte(types.Wasm)
}

func validLocality(b byte) bool {
	return b <= byte(directive.LocalityGoroutine)
}

func (w *pkgWriter) writeVarMeta(v *types.Var) {
	var flags byte
	if v != nil {
		if v.LinkName() != "" {
			flags |= varMetaLinkName
		}
		if v.ExportName() != "" {
			flags |= varMetaExportName
		}
		if v.Locality() != directive.LocalityNone {
			flags |= varMetaLocality
		}
	}
	w.enc.writeByte(flags)
	if flags&varMetaLinkName != 0 {
		w.enc.writeStr(v.LinkName())
	}
	if flags&varMetaExportName != 0 {
		w.enc.writeStr(v.ExportName())
	}
	if flags&varMetaLocality != 0 {
		w.enc.writeByte(byte(v.Locality()))
	}
}

func (w *pkgWriter) writeVar(v *types.Var) {
	if v == nil {
		w.enc.writeBool(false)
		return
	}
	w.enc.writeBool(true)
	w.enc.writeUvarint(w.internPkg(v.Pkg()))
	w.enc.writeStr(v.Name())
	w.enc.writeBool(v.Embedded())
	w.enc.writeBool(v.IsField())
	w.enc.writeByte(byte(v.Kind()))
	w.writeType(v.Type())
}

func (w *pkgWriter) writeType(t types.Type) {
	if t == nil {
		w.enc.writeByte(typeTagNil)
		return
	}
	switch u := t.(type) {
	case *types.Basic:
		w.enc.writeByte(typeTagBasic)
		w.enc.writeByte(byte(u.Kind()))
	case *types.Pointer:
		w.enc.writeByte(typeTagPointer)
		w.enc.writeByte(byte(u.Kind()))
		w.writeType(u.Elem())
	case *types.Slice:
		w.enc.writeByte(typeTagSlice)
		w.writeType(u.Elem())
	case *types.Array:
		w.enc.writeByte(typeTagArray)
		w.enc.writeVarint(u.Len())
		w.writeType(u.Elem())
	case *types.Map:
		w.enc.writeByte(typeTagMap)
		w.writeType(u.Key())
		w.writeType(u.Elem())
	case *types.Chan:
		w.enc.writeByte(typeTagChan)
		w.enc.writeByte(byte(u.Dir()))
		w.writeType(u.Elem())
	case *types.Tuple:
		w.enc.writeByte(typeTagTuple)
		w.enc.writeUvarint(uint64(u.Len()))
		for i := 0; i < u.Len(); i++ {
			w.writeVar(u.At(i))
		}
	case *types.Signature:
		w.writeSignature(u, false)
	case *types.Struct:
		w.enc.writeByte(typeTagStruct)
		w.enc.writeBool(u.NativeLayout())
		w.enc.writeUvarint(uint64(u.NumFields()))
		for i := 0; i < u.NumFields(); i++ {
			w.writeVar(u.Field(i))
			w.enc.writeStr(u.Tag(i))
		}
	case *types.Interface:
		w.enc.writeByte(typeTagInterface)
		w.enc.writeBool(u.IsImplicit())
		w.enc.writeUvarint(uint64(u.NumExplicitMethods()))
		for i := 0; i < u.NumExplicitMethods(); i++ {
			m := u.ExplicitMethod(i)
			w.enc.writeUvarint(w.internPkg(m.Pkg()))
			w.enc.writeStr(m.Name())
			if sig, ok := m.Type().(*types.Signature); ok && sig.Recv() != nil {
				if _, isIface := sig.Recv().Type().(*types.Interface); isIface {
					w.writeSignature(sig, true)
					continue
				}
			}
			w.writeType(m.Type())
		}
		w.enc.writeUvarint(uint64(u.NumEmbeddeds()))
		for i := 0; i < u.NumEmbeddeds(); i++ {
			w.writeType(u.EmbeddedType(i))
		}
	case *types.Union:
		w.enc.writeByte(typeTagUnion)
		w.enc.writeUvarint(uint64(u.Len()))
		for i := 0; i < u.Len(); i++ {
			term := u.Term(i)
			w.enc.writeBool(term.Tilde())
			w.writeType(term.Type())
		}
	case *types.Named:
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			w.enc.writeByte(typeTagNamedInst)
			w.enc.writeUvarint(w.namedIndex[orig])
			targs := u.TypeArgs()
			w.enc.writeUvarint(uint64(targs.Len()))
			for i := 0; i < targs.Len(); i++ {
				w.writeType(targs.At(i))
			}
			return
		}
		w.enc.writeByte(typeTagNamedRef)
		w.enc.writeUvarint(w.namedIndex[u])
	case *types.Alias:
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			w.enc.writeByte(typeTagAliasInst)
			w.enc.writeUvarint(w.aliasIndex[orig])
			targs := u.TypeArgs()
			w.enc.writeUvarint(uint64(targs.Len()))
			for i := 0; i < targs.Len(); i++ {
				w.writeType(targs.At(i))
			}
			return
		}
		w.enc.writeByte(typeTagAliasRef)
		w.enc.writeUvarint(w.aliasIndex[u])
	case *types.TypeParam:
		w.enc.writeByte(typeTagTypeParamRef)
		w.enc.writeUvarint(w.tpIndex[u])
	case *types.SIMDVec:
		w.enc.writeByte(typeTagSIMDVec)
		w.enc.writeByte(byte(u.Elem().Kind()))
		w.enc.writeUvarint(uint64(u.Len()))
		w.enc.writeBool(u.IsMask())
	default:
		w.enc.writeByte(typeTagNil)
	}
}

func (w *pkgWriter) writeSignature(u *types.Signature, omitRecv bool) {
	w.enc.writeByte(typeTagSignature)
	w.enc.writeByte(byte(u.CallConv()))
	w.enc.writeBool(u.Variadic())
	if omitRecv {
		w.writeVar(nil)
	} else {
		w.writeVar(u.Recv())
	}
	rtps := u.RecvTypeParams()
	w.enc.writeUvarint(uint64(rtps.Len()))
	for i := 0; i < rtps.Len(); i++ {
		w.enc.writeUvarint(w.tpIndex[rtps.At(i)])
	}
	tps := u.TypeParams()
	w.enc.writeUvarint(uint64(tps.Len()))
	for i := 0; i < tps.Len(); i++ {
		w.enc.writeUvarint(w.tpIndex[tps.At(i)])
	}
	w.writeType(u.Params())
	w.writeType(u.Results())
}

func (w *pkgWriter) writeConst(v constant.Value) {
	if v == nil || v.Kind() == constant.Unknown {
		w.enc.writeByte(constTagNil)
		return
	}
	switch v.Kind() {
	case constant.Bool:
		w.enc.writeByte(constTagBool)
		w.enc.writeBool(constant.BoolVal(v))
	case constant.String:
		w.enc.writeByte(constTagString)
		w.enc.writeStr(constant.StringVal(v))
	case constant.Int:
		w.enc.writeByte(constTagInt)
		w.enc.writeStr(v.ExactString())
	case constant.Float:
		w.enc.writeByte(constTagFloat)
		w.enc.writeStr(v.ExactString())
	case constant.Complex:
		w.enc.writeByte(constTagComplex)
		w.enc.writeStr(constant.Real(v).ExactString())
		w.enc.writeStr(constant.Imag(v).ExactString())
	default:
		w.enc.writeByte(constTagNil)
	}
}

type bundleState struct {
	nameds      map[*types.Package]map[string]*types.Named
	filledNamed map[*types.Named]bool
	aliases     map[*types.Package]map[string]*types.Alias
	filledAlias map[*types.Alias]bool
}

func newBundleState() *bundleState {
	return &bundleState{
		nameds:      make(map[*types.Package]map[string]*types.Named),
		filledNamed: make(map[*types.Named]bool),
		aliases:     make(map[*types.Package]map[string]*types.Alias),
		filledAlias: make(map[*types.Alias]bool),
	}
}

func (b *bundleState) lookupNamed(owner *types.Package, name string) *types.Named {
	if b == nil || owner == nil {
		return nil
	}
	return b.nameds[owner][name]
}

func (b *bundleState) registerNamed(owner *types.Package, name string, n *types.Named) {
	if b == nil || owner == nil || n == nil {
		return
	}
	if b.nameds[owner] == nil {
		b.nameds[owner] = make(map[string]*types.Named)
	}
	if b.nameds[owner][name] == nil {
		b.nameds[owner][name] = n
	}
}

func (b *bundleState) lookupAlias(owner *types.Package, name string) *types.Alias {
	if b == nil || owner == nil {
		return nil
	}
	return b.aliases[owner][name]
}

func (b *bundleState) registerAlias(owner *types.Package, name string, a *types.Alias) {
	if b == nil || owner == nil || a == nil {
		return
	}
	if b.aliases[owner] == nil {
		b.aliases[owner] = make(map[string]*types.Alias)
	}
	if b.aliases[owner][name] == nil {
		b.aliases[owner][name] = a
	}
}

type pkgReader struct {
	dec     *decoder
	imports map[string]*types.Package
	bundle  *bundleState

	path      string
	name      string
	complete  bool
	kind      directive.PackageKind
	kindParam string
	skipAll   bool
	skipNames []string
	pkg       *types.Package
	reused    bool

	pkgs    []*types.Package
	tparams []*types.TypeParam
	nameds  []*types.Named
	aliases []*types.Alias

	namedCC       []types.CallConv
	namedReused   []bool
	namedBlobs    [][]byte
	namedResolved []bool

	aliasOwners    []*types.Package
	aliasNames     []string
	aliasCC        []types.CallConv
	aliasReused    []bool
	aliasBlobs     [][]byte
	aliasResolved  []bool
	aliasResolving []bool
}

func (r *pkgReader) pkgByID(id uint64) *types.Package {
	if id == 0 || int(id) > len(r.pkgs) {
		return nil
	}
	return r.pkgs[id-1]
}

func (r *pkgReader) decode() (*types.Package, error) {
	if err := r.readHeader(); err != nil {
		return nil, err
	}
	r.initPackage()
	if err := r.readPackageTable(); err != nil {
		return nil, err
	}
	if err := r.decodeBody(); err != nil {
		return nil, err
	}
	return r.pkg, nil
}

func (r *pkgReader) readHeader() error {
	path, err := r.dec.readStr()
	if err != nil {
		return err
	}
	name, err := r.dec.readStr()
	if err != nil {
		return err
	}
	_, err = r.dec.readStr() // goVersion (informational)
	if err != nil {
		return err
	}
	complete, err := r.dec.readBool()
	if err != nil {
		return err
	}
	kind, err := r.dec.readByte()
	if err != nil {
		return err
	}
	if !validPackageKind(kind) {
		return errCorrupt
	}
	kindParam, err := r.dec.readStr()
	if err != nil {
		return err
	}
	skipAll, err := r.dec.readBool()
	if err != nil {
		return err
	}
	skipNames, err := r.dec.readStringSlice()
	if err != nil {
		return err
	}
	r.path = path
	r.name = name
	r.complete = complete
	r.kind = directive.PackageKind(kind)
	r.kindParam = kindParam
	r.skipAll = skipAll
	r.skipNames = skipNames
	return nil
}

func (r *pkgReader) initPackage() {
	if r.path == "unsafe" {
		r.pkg = types.Unsafe
		r.reused = true
		return
	}
	if r.imports != nil && r.path != "" {
		if existing := r.imports[r.path]; existing != nil {
			r.pkg = existing
			r.reused = existing.Complete()
			return
		}
	}
	r.pkg = types.NewPackage(r.path, r.name)
	if r.kind != directive.PkgNormal || r.kindParam != "" {
		r.pkg.SetKind(r.kind, r.kindParam)
	}
	if r.skipAll {
		r.pkg.SetSkipAll(true)
	}
	if len(r.skipNames) > 0 {
		r.pkg.AddSkip(r.skipNames...)
	}
	if r.imports != nil && r.path != "" {
		r.imports[r.path] = r.pkg
	}
}

func (r *pkgReader) readPackageTable() error {
	pkgCount, err := r.dec.readUvarint()
	if err != nil {
		return err
	}
	r.pkgs = make([]*types.Package, pkgCount)
	for i := uint64(0); i < pkgCount; i++ {
		pPath, err := r.dec.readStr()
		if err != nil {
			return err
		}
		pName, err := r.dec.readStr()
		if err != nil {
			return err
		}
		if pPath == r.path && i == 0 {
			r.pkgs[i] = r.pkg
		} else if pPath == "unsafe" {
			r.pkgs[i] = types.Unsafe
		} else if r.imports != nil && r.imports[pPath] != nil {
			r.pkgs[i] = r.imports[pPath]
		} else {
			p := types.NewPackage(pPath, pName)
			if r.imports != nil && pPath != "" {
				r.imports[pPath] = p
			}
			r.pkgs[i] = p
		}
	}

	impCount, err := r.dec.readUvarint()
	if err != nil {
		return err
	}
	if impCount > 0 {
		imps := make([]*types.Package, impCount)
		for i := uint64(0); i < impCount; i++ {
			id, err := r.dec.readUvarint()
			if err != nil {
				return err
			}
			imps[i] = r.pkgByID(id)
		}
		if !r.reused {
			r.pkg.SetImports(imps)
		}
	}
	return nil
}

func (r *pkgReader) decodeBody() error {
	pkg := r.pkg
	path := r.path

	// Read TypeParam shells.
	tpCount, err := r.dec.readUvarint()
	if err != nil {
		return err
	}
	r.tparams = make([]*types.TypeParam, tpCount)
	for i := uint64(0); i < tpCount; i++ {
		pkgID, err := r.dec.readUvarint()
		if err != nil {
			return err
		}
		tpName, err := r.dec.readStr()
		if err != nil {
			return err
		}
		_, err = r.dec.readUvarint() // index (assigned by SetTypeParams)
		if err != nil {
			return err
		}
		r.tparams[i] = types.NewTypeParam(types.NewTypeName(token.NoPos, r.pkgByID(pkgID), tpName, nil), nil)
	}

	// Read Named shells, reusing Universe, already-imported external Named types,
	// and shared shells within a bundle.
	namedCount, err := r.dec.readUvarint()
	if err != nil {
		return err
	}
	r.nameds = make([]*types.Named, namedCount)
	r.namedCC = make([]types.CallConv, namedCount)
	r.namedReused = make([]bool, namedCount)
	r.namedResolved = make([]bool, namedCount)
	for i := uint64(0); i < namedCount; i++ {
		pkgID, err := r.dec.readUvarint()
		if err != nil {
			return err
		}
		nName, err := r.dec.readStr()
		if err != nil {
			return err
		}
		cc, err := r.dec.readByte()
		if err != nil {
			return err
		}
		if !validCallConv(cc) {
			return errCorrupt
		}
		r.namedCC[i] = types.CallConv(cc)
		owner := r.pkgByID(pkgID)
		if pkgID == 0 {
			if obj := types.Universe.Lookup(nName); obj != nil {
				if existing, ok := obj.Type().(*types.Named); ok {
					r.nameds[i] = existing
					r.namedReused[i] = true
					continue
				}
			}
		} else if owner != nil {
			if obj := owner.Scope().Lookup(nName); obj != nil {
				if existing, ok := obj.Type().(*types.Named); ok {
					r.nameds[i] = existing
					r.namedReused[i] = existing.Underlying() != nil
					continue
				}
			}
			if existing := r.bundle.lookupNamed(owner, nName); existing != nil {
				r.nameds[i] = existing
				r.namedReused[i] = r.bundle != nil && r.bundle.filledNamed[existing]
				continue
			}
		}
		n := types.NewNamed(types.NewTypeName(token.NoPos, owner, nName, nil), nil, nil)
		r.nameds[i] = n
		r.bundle.registerNamed(owner, nName, n)
	}

	// Read Alias shells, reusing Universe, already-imported external Alias types,
	// and shared shells within a bundle.
	aliasCount, err := r.dec.readUvarint()
	if err != nil {
		return err
	}
	r.aliases = make([]*types.Alias, aliasCount)
	r.aliasOwners = make([]*types.Package, aliasCount)
	r.aliasNames = make([]string, aliasCount)
	r.aliasCC = make([]types.CallConv, aliasCount)
	r.aliasReused = make([]bool, aliasCount)
	r.aliasResolved = make([]bool, aliasCount)
	r.aliasResolving = make([]bool, aliasCount)
	for i := uint64(0); i < aliasCount; i++ {
		pkgID, err := r.dec.readUvarint()
		if err != nil {
			return err
		}
		aName, err := r.dec.readStr()
		if err != nil {
			return err
		}
		cc, err := r.dec.readByte()
		if err != nil {
			return err
		}
		if !validCallConv(cc) {
			return errCorrupt
		}
		owner := r.pkgByID(pkgID)
		r.aliasOwners[i] = owner
		r.aliasNames[i] = aName
		r.aliasCC[i] = types.CallConv(cc)
		if pkgID == 0 {
			if obj := types.Universe.Lookup(aName); obj != nil {
				if existing, ok := obj.Type().(*types.Alias); ok {
					r.aliases[i] = existing
					r.aliasReused[i] = true
					r.aliasResolved[i] = true
					continue
				}
			}
		} else if owner != nil {
			if obj := owner.Scope().Lookup(aName); obj != nil {
				if existing, ok := obj.Type().(*types.Alias); ok {
					r.aliases[i] = existing
					r.aliasReused[i] = existing.Rhs() != nil
					r.aliasResolved[i] = r.aliasReused[i]
					continue
				}
			}
			if existing := r.bundle.lookupAlias(owner, aName); existing != nil {
				r.aliases[i] = existing
				r.aliasReused[i] = r.bundle != nil && r.bundle.filledAlias[existing]
				r.aliasResolved[i] = r.aliasReused[i]
				continue
			}
		}
		r.aliases[i] = types.NewAlias(nil, nil)
	}

	// Read raw body blobs up front so Aliases and instantiated Named origins
	// can be resolved on demand before any struct field or signature binds them.
	tpConstraintBlobs := make([][]byte, tpCount)
	for i := uint64(0); i < tpCount; i++ {
		blob, err := r.dec.readBytes()
		if err != nil {
			return err
		}
		tpConstraintBlobs[i] = blob
	}
	r.namedBlobs = make([][]byte, namedCount)
	for i := uint64(0); i < namedCount; i++ {
		blob, err := r.dec.readBytes()
		if err != nil {
			return err
		}
		r.namedBlobs[i] = blob
	}
	r.aliasBlobs = make([][]byte, aliasCount)
	for i := uint64(0); i < aliasCount; i++ {
		blob, err := r.dec.readBytes()
		if err != nil {
			return err
		}
		r.aliasBlobs[i] = blob
	}

	// Fill TypeParam constraints (may reference Aliases resolved on demand).
	for i := uint64(0); i < tpCount; i++ {
		c, err := r.decodeTypeBlob(tpConstraintBlobs[i])
		if err != nil {
			return err
		}
		r.tparams[i].SetConstraint(c)
	}

	// Resolve all Aliases and Named definitions in dependency order.
	for i := range r.aliases {
		if _, err := r.resolveAlias(i); err != nil {
			return err
		}
	}
	for i := range r.nameds {
		if err := r.resolveNamed(i); err != nil {
			return err
		}
	}

	// Read package scope objects.
	objCount, err := r.dec.readUvarint()
	if err != nil {
		return err
	}
	for i := uint64(0); i < objCount; i++ {
		tag, err := r.dec.readByte()
		if err != nil {
			return err
		}
		switch tag {
		case objTagConst:
			oName, err := r.dec.readStr()
			if err != nil {
				return err
			}
			typ, err := r.readType()
			if err != nil {
				return err
			}
			val, err := r.readConst()
			if err != nil {
				return err
			}
			if !r.reused {
				pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, oName, typ, val))
			}
		case objTagVar:
			oName, err := r.dec.readStr()
			if err != nil {
				return err
			}
			vk, err := r.dec.readByte()
			if err != nil {
				return err
			}
			typ, err := r.readType()
			if err != nil {
				return err
			}
			varMeta, err := r.readVarMeta()
			if err != nil {
				return err
			}
			if !r.reused {
				v := types.NewVar(token.NoPos, pkg, oName, typ)
				v.SetKind(types.VarKind(vk))
				varMeta.apply(v)
				pkg.Scope().Insert(v)
			}
		case objTagFunc:
			oName, err := r.dec.readStr()
			if err != nil {
				return err
			}
			typ, err := r.readType()
			if err != nil {
				return err
			}
			fnMeta, err := r.readFuncMeta()
			if err != nil {
				return err
			}
			if !r.reused {
				sig, _ := typ.(*types.Signature)
				fn := types.NewFunc(token.NoPos, pkg, oName, sig)
				fnMeta.apply(fn)
				pkg.Scope().Insert(fn)
			}
		case objTagTypeName:
			oName, err := r.dec.readStr()
			if err != nil {
				return err
			}
			isAlias, err := r.dec.readBool()
			if err != nil {
				return err
			}
			typ, err := r.readType()
			if err != nil {
				return err
			}
			if !r.reused {
				if n, ok := typ.(*types.Named); ok && n.Obj() != nil && n.Obj().Name() == oName {
					pkg.Scope().Insert(n.Obj())
				} else if a, ok := typ.(*types.Alias); ok && a.Obj() != nil && a.Obj().Name() == oName {
					pkg.Scope().Insert(a.Obj())
				} else {
					tn := types.NewTypeName(token.NoPos, pkg, oName, typ)
					if isAlias {
						_ = types.NewAlias(tn, typ)
					}
					pkg.Scope().Insert(tn)
				}
			}
		default:
			return errCorrupt
		}
	}

	if r.complete {
		pkg.MarkComplete()
	}
	if r.imports != nil && path != "" {
		r.imports[path] = pkg
	}
	return nil
}

func (r *pkgReader) withSubDecoder(blob []byte, fn func() error) error {
	orig := r.dec
	r.dec = &decoder{data: blob, strings: orig.strings}
	defer func() { r.dec = orig }()
	return fn()
}

func (r *pkgReader) decodeTypeBlob(blob []byte) (types.Type, error) {
	var t types.Type
	err := r.withSubDecoder(blob, func() error {
		var err error
		t, err = r.readType()
		return err
	})
	return t, err
}

type decodedFuncMeta struct {
	linkName       string
	exportName     string
	wasmImport     *directive.WasmImport
	wasmExport     string
	hasClosureEnv  bool
	noInline       bool
	noInterface    bool
	noEscape       bool
	uintptrEscapes bool
	noSplit        bool
}

func (m decodedFuncMeta) apply(fn *types.Func) {
	if fn == nil {
		return
	}
	if m.linkName != "" {
		fn.SetLinkName(m.linkName)
	}
	if m.exportName != "" {
		fn.SetExportName(m.exportName)
	}
	if m.wasmImport != nil {
		fn.SetWasmImport(m.wasmImport)
	}
	if m.wasmExport != "" {
		fn.SetWasmExport(m.wasmExport)
	}
	if m.hasClosureEnv {
		fn.SetHasClosureEnv(true)
	}
	if m.noInline {
		fn.SetNoInline(true)
	}
	if m.noInterface {
		fn.SetNoInterface(true)
	}
	if m.noEscape {
		fn.SetNoEscape(true)
	}
	if m.uintptrEscapes {
		fn.SetUintptrEscapes(true)
	}
	if m.noSplit {
		fn.SetNoSplit(true)
	}
}

func (r *pkgReader) readFuncMeta() (decodedFuncMeta, error) {
	flags, err := r.dec.readUvarint()
	if err != nil {
		return decodedFuncMeta{}, err
	}
	if flags&^funcMetaAllFlags != 0 {
		return decodedFuncMeta{}, errCorrupt
	}
	m := decodedFuncMeta{
		hasClosureEnv:  flags&funcMetaClosureEnv != 0,
		noInline:       flags&funcMetaNoInline != 0,
		noInterface:    flags&funcMetaNoInterface != 0,
		noEscape:       flags&funcMetaNoEscape != 0,
		uintptrEscapes: flags&funcMetaUintptrEscapes != 0,
		noSplit:        flags&funcMetaNoSplit != 0,
	}
	if flags&funcMetaLinkName != 0 {
		if m.linkName, err = r.dec.readStr(); err != nil {
			return decodedFuncMeta{}, err
		}
	}
	if flags&funcMetaExportName != 0 {
		if m.exportName, err = r.dec.readStr(); err != nil {
			return decodedFuncMeta{}, err
		}
	}
	if flags&funcMetaWasmImport != 0 {
		mod, err := r.dec.readStr()
		if err != nil {
			return decodedFuncMeta{}, err
		}
		name, err := r.dec.readStr()
		if err != nil {
			return decodedFuncMeta{}, err
		}
		m.wasmImport = &directive.WasmImport{Module: mod, Name: name}
	}
	if flags&funcMetaWasmExport != 0 {
		if m.wasmExport, err = r.dec.readStr(); err != nil {
			return decodedFuncMeta{}, err
		}
	}
	return m, nil
}

type decodedVarMeta struct {
	linkName   string
	exportName string
	locality   directive.Locality
}

func (m decodedVarMeta) apply(v *types.Var) {
	if v == nil {
		return
	}
	if m.linkName != "" {
		v.SetLinkName(m.linkName)
	}
	if m.exportName != "" {
		v.SetExportName(m.exportName)
	}
	if m.locality != directive.LocalityNone {
		v.SetLocality(m.locality)
	}
}

func (r *pkgReader) readVarMeta() (decodedVarMeta, error) {
	flags, err := r.dec.readByte()
	if err != nil {
		return decodedVarMeta{}, err
	}
	if flags&^varMetaAllFlags != 0 {
		return decodedVarMeta{}, errCorrupt
	}
	var m decodedVarMeta
	if flags&varMetaLinkName != 0 {
		if m.linkName, err = r.dec.readStr(); err != nil {
			return decodedVarMeta{}, err
		}
	}
	if flags&varMetaExportName != 0 {
		if m.exportName, err = r.dec.readStr(); err != nil {
			return decodedVarMeta{}, err
		}
	}
	if flags&varMetaLocality != 0 {
		loc, err := r.dec.readByte()
		if err != nil {
			return decodedVarMeta{}, err
		}
		if !validLocality(loc) {
			return decodedVarMeta{}, errCorrupt
		}
		m.locality = directive.Locality(loc)
	}
	return m, nil
}

func (r *pkgReader) resolveNamed(idx int) error {
	if r.namedResolved[idx] {
		return nil
	}
	r.namedResolved[idx] = true
	return r.withSubDecoder(r.namedBlobs[idx], func() error {
		tps, err := r.readTypeParams()
		if err != nil {
			return err
		}
		under, err := r.readType()
		if err != nil {
			return err
		}
		nm, err := r.dec.readUvarint()
		if err != nil {
			return err
		}
		type methodRec struct {
			pkg  *types.Package
			name string
			sig  *types.Signature
			meta decodedFuncMeta
		}
		var methods []methodRec
		for j := uint64(0); j < nm; j++ {
			mPkgID, err := r.dec.readUvarint()
			if err != nil {
				return err
			}
			mName, err := r.dec.readStr()
			if err != nil {
				return err
			}
			mTyp, err := r.readType()
			if err != nil {
				return err
			}
			mMeta, err := r.readFuncMeta()
			if err != nil {
				return err
			}
			sig, _ := mTyp.(*types.Signature)
			if !r.namedReused[idx] {
				methods = append(methods, methodRec{pkg: r.pkgByID(mPkgID), name: mName, sig: sig, meta: mMeta})
			}
		}
		if r.namedReused[idx] {
			return nil
		}
		if len(tps) > 0 {
			r.nameds[idx].SetTypeParams(tps)
		}
		if sig, ok := under.(*types.Signature); ok && r.namedCC[idx] != types.Go {
			under = sig.WithCallConv(r.namedCC[idx])
		}
		r.nameds[idx].SetUnderlying(under)
		if r.namedCC[idx] != types.Go {
			r.nameds[idx].SetCallConv(r.namedCC[idx])
		}
		for _, mr := range methods {
			fn := types.NewFunc(token.NoPos, mr.pkg, mr.name, mr.sig)
			mr.meta.apply(fn)
			r.nameds[idx].AddMethod(fn)
		}
		if r.bundle != nil {
			r.bundle.filledNamed[r.nameds[idx]] = true
		}
		if obj := r.nameds[idx].Obj(); obj != nil {
			if owner := obj.Pkg(); owner != nil && owner != r.pkg && !owner.Complete() {
				owner.Scope().Insert(obj)
			}
		}
		return nil
	})
}

func (r *pkgReader) resolveAlias(idx int) (*types.Alias, error) {
	if r.aliasResolved[idx] || r.aliasResolving[idx] {
		return r.aliases[idx], nil
	}
	r.aliasResolving[idx] = true
	defer func() { r.aliasResolving[idx] = false }()
	err := r.withSubDecoder(r.aliasBlobs[idx], func() error {
		tps, err := r.readTypeParams()
		if err != nil {
			return err
		}
		rhs, err := r.readType()
		if err != nil {
			return err
		}
		if r.aliasReused[idx] {
			r.aliasResolved[idx] = true
			return nil
		}
		tn := types.NewTypeName(token.NoPos, r.aliasOwners[idx], r.aliasNames[idx], nil)
		if sig, ok := rhs.(*types.Signature); ok && r.aliasCC[idx] != types.Go {
			rhs = sig.WithCallConv(r.aliasCC[idx])
		}
		a := types.NewAlias(tn, rhs)
		if r.aliasCC[idx] != types.Go {
			a.SetCallConv(r.aliasCC[idx])
		}
		if len(tps) > 0 {
			a.SetTypeParams(tps)
		}
		r.aliases[idx] = a
		r.aliasResolved[idx] = true
		if r.bundle != nil {
			r.bundle.registerAlias(r.aliasOwners[idx], r.aliasNames[idx], a)
			r.bundle.filledAlias[a] = true
		}
		if owner := r.aliasOwners[idx]; owner != nil && owner != r.pkg && !owner.Complete() {
			owner.Scope().Insert(tn)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.aliases[idx], nil
}

func (r *pkgReader) readTypeParams() ([]*types.TypeParam, error) {
	n, err := r.dec.readUvarint()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]*types.TypeParam, n)
	for i := uint64(0); i < n; i++ {
		idx, err := r.dec.readUvarint()
		if err != nil || idx >= uint64(len(r.tparams)) {
			return nil, errCorrupt
		}
		out[i] = r.tparams[idx]
	}
	return out, nil
}

func (r *pkgReader) readVar() (*types.Var, error) {
	present, err := r.dec.readBool()
	if err != nil || !present {
		return nil, err
	}
	pkgID, err := r.dec.readUvarint()
	if err != nil {
		return nil, err
	}
	name, err := r.dec.readStr()
	if err != nil {
		return nil, err
	}
	embedded, err := r.dec.readBool()
	if err != nil {
		return nil, err
	}
	isField, err := r.dec.readBool()
	if err != nil {
		return nil, err
	}
	vk, err := r.dec.readByte()
	if err != nil {
		return nil, err
	}
	typ, err := r.readType()
	if err != nil {
		return nil, err
	}
	var v *types.Var
	if isField {
		v = types.NewField(token.NoPos, r.pkgByID(pkgID), name, typ, embedded)
	} else {
		v = types.NewParam(token.NoPos, r.pkgByID(pkgID), name, typ)
	}
	v.SetKind(types.VarKind(vk))
	return v, nil
}

func (r *pkgReader) readType() (types.Type, error) {
	tag, err := r.dec.readByte()
	if err != nil {
		return nil, err
	}
	switch tag {
	case typeTagNil:
		return nil, nil
	case typeTagBasic:
		k, err := r.dec.readByte()
		if err != nil || int(k) >= len(types.Typ) {
			return nil, errCorrupt
		}
		return types.Typ[types.BasicKind(k)], nil
	case typeTagPointer:
		pk, err := r.dec.readByte()
		if err != nil {
			return nil, err
		}
		elem, err := r.readType()
		if err != nil {
			return nil, err
		}
		return types.NewPointerWithKind(elem, types.PtrKind(pk)), nil
	case typeTagSlice:
		elem, err := r.readType()
		if err != nil {
			return nil, err
		}
		return types.NewSlice(elem), nil
	case typeTagArray:
		length, err := r.dec.readVarint()
		if err != nil {
			return nil, err
		}
		elem, err := r.readType()
		if err != nil {
			return nil, err
		}
		return types.NewArray(elem, length), nil
	case typeTagMap:
		key, err := r.readType()
		if err != nil {
			return nil, err
		}
		elem, err := r.readType()
		if err != nil {
			return nil, err
		}
		return types.NewMap(key, elem), nil
	case typeTagChan:
		dir, err := r.dec.readByte()
		if err != nil {
			return nil, err
		}
		elem, err := r.readType()
		if err != nil {
			return nil, err
		}
		return types.NewChan(types.ChanDir(dir), elem), nil
	case typeTagTuple:
		n, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		vars := make([]*types.Var, n)
		for i := uint64(0); i < n; i++ {
			v, err := r.readVar()
			if err != nil {
				return nil, err
			}
			vars[i] = v
		}
		return types.NewTuple(vars...), nil
	case typeTagSignature:
		cc, err := r.dec.readByte()
		if err != nil {
			return nil, err
		}
		if !validCallConv(cc) {
			return nil, errCorrupt
		}
		variadic, err := r.dec.readBool()
		if err != nil {
			return nil, err
		}
		recv, err := r.readVar()
		if err != nil {
			return nil, err
		}
		rtps, err := r.readTypeParams()
		if err != nil {
			return nil, err
		}
		tps, err := r.readTypeParams()
		if err != nil {
			return nil, err
		}
		paramsT, err := r.readType()
		if err != nil {
			return nil, err
		}
		resultsT, err := r.readType()
		if err != nil {
			return nil, err
		}
		params, _ := paramsT.(*types.Tuple)
		results, _ := resultsT.(*types.Tuple)
		sig := types.NewSignatureType(recv, rtps, tps, params, results, variadic)
		if types.CallConv(cc) != types.Go {
			sig = sig.WithCallConv(types.CallConv(cc))
		}
		return sig, nil
	case typeTagStruct:
		nativeLayout, err := r.dec.readBool()
		if err != nil {
			return nil, err
		}
		n, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		fields := make([]*types.Var, n)
		tags := make([]string, n)
		hasTags := false
		for i := uint64(0); i < n; i++ {
			f, err := r.readVar()
			if err != nil {
				return nil, err
			}
			tagStr, err := r.dec.readStr()
			if err != nil {
				return nil, err
			}
			fields[i] = f
			tags[i] = tagStr
			if tagStr != "" {
				hasTags = true
			}
		}
		if !hasTags {
			tags = nil
		}
		st := types.NewStruct(fields, tags)
		if nativeLayout {
			st.SetNativeLayout(true)
		}
		return st, nil
	case typeTagInterface:
		implicit, err := r.dec.readBool()
		if err != nil {
			return nil, err
		}
		nm, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		methods := make([]*types.Func, nm)
		for i := uint64(0); i < nm; i++ {
			pkgID, err := r.dec.readUvarint()
			if err != nil {
				return nil, err
			}
			mName, err := r.dec.readStr()
			if err != nil {
				return nil, err
			}
			mTyp, err := r.readType()
			if err != nil {
				return nil, err
			}
			sig, _ := mTyp.(*types.Signature)
			methods[i] = types.NewFunc(token.NoPos, r.pkgByID(pkgID), mName, sig)
		}
		ne, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		embs := make([]types.Type, ne)
		for i := uint64(0); i < ne; i++ {
			e, err := r.readType()
			if err != nil {
				return nil, err
			}
			embs[i] = e
		}
		iface := types.NewInterfaceType(methods, embs).Complete()
		if implicit {
			iface.MarkImplicit()
		}
		return iface, nil
	case typeTagUnion:
		n, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		terms := make([]*types.Term, n)
		for i := uint64(0); i < n; i++ {
			tilde, err := r.dec.readBool()
			if err != nil {
				return nil, err
			}
			typ, err := r.readType()
			if err != nil {
				return nil, err
			}
			terms[i] = types.NewTerm(tilde, typ)
		}
		return types.NewUnion(terms), nil
	case typeTagNamedRef:
		idx, err := r.dec.readUvarint()
		if err != nil || idx >= uint64(len(r.nameds)) {
			return nil, errCorrupt
		}
		return r.nameds[idx], nil
	case typeTagNamedInst:
		origIdx, err := r.dec.readUvarint()
		if err != nil || origIdx >= uint64(len(r.nameds)) {
			return nil, errCorrupt
		}
		nArgs, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		args := make([]types.Type, nArgs)
		for i := uint64(0); i < nArgs; i++ {
			args[i], err = r.readType()
			if err != nil {
				return nil, err
			}
		}
		if err := r.resolveNamed(int(origIdx)); err != nil {
			return nil, err
		}
		orig := r.nameds[origIdx]
		if orig.Underlying() == nil {
			return orig, nil
		}
		inst, err := types.Instantiate(nil, orig, args, false)
		if err != nil {
			return nil, err
		}
		return inst, nil
	case typeTagAliasRef:
		idx, err := r.dec.readUvarint()
		if err != nil || idx >= uint64(len(r.aliases)) {
			return nil, errCorrupt
		}
		return r.resolveAlias(int(idx))
	case typeTagAliasInst:
		origIdx, err := r.dec.readUvarint()
		if err != nil || origIdx >= uint64(len(r.aliases)) {
			return nil, errCorrupt
		}
		nArgs, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		args := make([]types.Type, nArgs)
		for i := uint64(0); i < nArgs; i++ {
			args[i], err = r.readType()
			if err != nil {
				return nil, err
			}
		}
		orig, err := r.resolveAlias(int(origIdx))
		if err != nil {
			return nil, err
		}
		if orig.Rhs() == nil {
			return orig, nil
		}
		inst, err := types.Instantiate(nil, orig, args, false)
		if err != nil {
			return nil, err
		}
		return inst, nil
	case typeTagTypeParamRef:
		idx, err := r.dec.readUvarint()
		if err != nil || idx >= uint64(len(r.tparams)) {
			return nil, errCorrupt
		}
		return r.tparams[idx], nil
	case typeTagSIMDVec:
		k, err := r.dec.readByte()
		if err != nil || int(k) >= len(types.Typ) {
			return nil, errCorrupt
		}
		n, err := r.dec.readUvarint()
		if err != nil {
			return nil, err
		}
		isMask, err := r.dec.readBool()
		if err != nil {
			return nil, err
		}
		return types.NewSIMDVec(types.Typ[types.BasicKind(k)], int(n), isMask), nil
	default:
		return nil, errCorrupt
	}
}

func (r *pkgReader) readConst() (constant.Value, error) {
	tag, err := r.dec.readByte()
	if err != nil {
		return nil, err
	}
	switch tag {
	case constTagNil:
		return nil, nil
	case constTagBool:
		b, err := r.dec.readBool()
		if err != nil {
			return nil, err
		}
		return constant.MakeBool(b), nil
	case constTagString:
		s, err := r.dec.readStr()
		if err != nil {
			return nil, err
		}
		return constant.MakeString(s), nil
	case constTagInt:
		s, err := r.dec.readStr()
		if err != nil {
			return nil, err
		}
		return constant.MakeFromLiteral(s, token.INT, 0), nil
	case constTagFloat:
		s, err := r.dec.readStr()
		if err != nil {
			return nil, err
		}
		// ExactString for floats may be a rational "num/den".
		if slash := indexByte(s, '/'); slash >= 0 {
			num := constant.MakeFromLiteral(s[:slash], token.INT, 0)
			den := constant.MakeFromLiteral(s[slash+1:], token.INT, 0)
			return constant.BinaryOp(num, token.QUO, den), nil
		}
		return constant.MakeFromLiteral(s, token.FLOAT, 0), nil
	case constTagComplex:
		re, err := r.dec.readStr()
		if err != nil {
			return nil, err
		}
		im, err := r.dec.readStr()
		if err != nil {
			return nil, err
		}
		rVal := parseExactFloat(re)
		iVal := constant.MakeImag(parseExactFloat(im))
		return constant.BinaryOp(rVal, token.ADD, iVal), nil
	default:
		return nil, errCorrupt
	}
}

func parseExactFloat(s string) constant.Value {
	if slash := indexByte(s, '/'); slash >= 0 {
		num := constant.MakeFromLiteral(s[:slash], token.INT, 0)
		den := constant.MakeFromLiteral(s[slash+1:], token.INT, 0)
		return constant.BinaryOp(num, token.QUO, den)
	}
	return constant.MakeFromLiteral(s, token.FLOAT, 0)
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
