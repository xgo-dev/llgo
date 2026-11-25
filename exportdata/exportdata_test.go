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

package exportdata_test

import (
	"bytes"
	"errors"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestPackageRoundTripAndDeterminism(t *testing.T) {
	src := `package sample

import "unsafe"

//llgo:cdecl
type CFn func(a int32, args ...any) int32

type Signed interface {
	~int | ~int64
}

type Node struct {
	Val  int64 ` + "`json:\"val\"`" + `
	Next *Node
}

func (n *Node) Value() int64 { return n.Val }

type Box[T Signed] struct {
	Item T
}

type AliasInt = int64

const (
	CBool  = true
	CInt   = 123456789012345
	CFloat = 3.25
	CStr   = "hello"
	CCmplx = 1 + 2i
	CSize  = unsafe.Sizeof(CFn(nil))
)

var (
	GMap  map[string][]int
	GChan chan<- int32
	GArr  [4]byte
	GNode *Node
)

func Add(a, b int) (int, bool) { return a + b, true }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{
		GoVersion: "go1.20",
		Importer:  unsafeOnlyImporter{},
	}).Check("example.com/sample", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	raw1, err := exportdata.MarshalPackage(pkg)
	if err != nil {
		t.Fatalf("MarshalPackage: %v", err)
	}
	raw2, err := exportdata.MarshalPackage(pkg)
	if err != nil {
		t.Fatalf("MarshalPackage 2: %v", err)
	}
	if !bytes.Equal(raw1, raw2) {
		t.Fatal("MarshalPackage is not deterministic")
	}

	var buf bytes.Buffer
	if err := exportdata.WritePackage(&buf, pkg); err != nil {
		t.Fatal(err)
	}
	imports := make(map[string]*types.Package)
	got, err := exportdata.ReadPackage(&buf, imports)
	if err != nil {
		t.Fatalf("ReadPackage: %v", err)
	}
	if got.Path() != "example.com/sample" || got.Name() != "sample" || !got.Complete() {
		t.Fatalf("decoded pkg metadata mismatch: %v", got)
	}
	if imports["example.com/sample"] != got {
		t.Fatal("decoded package not registered in imports map")
	}

	// Verify CallConv preserved on CFn.
	cFn := got.Scope().Lookup("CFn").Type().Underlying().(*types.Signature)
	if cFn.CallConv() != types.CDecl || !cFn.Variadic() {
		t.Fatalf("decoded CFn signature = %s (cc=%v variadic=%v)", cFn, cFn.CallConv(), cFn.Variadic())
	}

	// Verify cyclic Node struct + tag + method.
	node := got.Scope().Lookup("Node").Type().(*types.Named)
	st := node.Underlying().(*types.Struct)
	if st.NumFields() != 2 || st.Tag(0) != `json:"val"` || st.Field(1).Type().(*types.Pointer).Elem() != node {
		t.Fatalf("decoded cyclic Node mismatch: %v", st)
	}
	if node.NumMethods() != 1 || node.Method(0).Name() != "Value" {
		t.Fatalf("decoded Node methods = %d", node.NumMethods())
	}

	// Verify generic Box[T Signed] & Union constraint.
	box := got.Scope().Lookup("Box").Type().(*types.Named)
	if box.TypeParams().Len() != 1 {
		t.Fatalf("decoded Box TypeParams = %v", box.TypeParams())
	}
	signed := got.Scope().Lookup("Signed").Type().Underlying().(*types.Interface)
	if signed.NumEmbeddeds() != 1 || signed.EmbeddedType(0).(*types.Union).Len() != 2 {
		t.Fatalf("decoded Signed union mismatch: %v", signed)
	}

	// Verify Alias.
	if aliasObj := got.Scope().Lookup("AliasInt").(*types.TypeName); !aliasObj.IsAlias() {
		t.Fatal("decoded AliasInt lost IsAlias")
	}

	// Verify all constant kinds.
	if !constant.BoolVal(got.Scope().Lookup("CBool").(*types.Const).Val()) {
		t.Fatal("CBool mismatch")
	}
	if v, ok := constant.Int64Val(got.Scope().Lookup("CInt").(*types.Const).Val()); !ok || v != 123456789012345 {
		t.Fatalf("CInt = %v", got.Scope().Lookup("CInt").(*types.Const).Val())
	}
	if v, _ := constant.Float64Val(got.Scope().Lookup("CFloat").(*types.Const).Val()); v != 3.25 {
		t.Fatalf("CFloat = %v", v)
	}
	if constant.StringVal(got.Scope().Lookup("CStr").(*types.Const).Val()) != "hello" {
		t.Fatal("CStr mismatch")
	}
	if got.Scope().Lookup("CCmplx").(*types.Const).Val().Kind() != constant.Complex {
		t.Fatal("CCmplx kind mismatch")
	}

	// Verify CPtr preserved on synthetic package.
	syn := types.NewPackage("example.com/syn", "syn")
	syn.Scope().Insert(types.NewVar(0, syn, "CP", types.NewPointerWithKind(types.Typ[types.Int8], types.CPtr)))
	synRaw, err := exportdata.MarshalPackage(syn)
	if err != nil {
		t.Fatal(err)
	}
	synGot, err := exportdata.UnmarshalPackage(synRaw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if synGot.Scope().Lookup("CP").Type().(*types.Pointer).Kind() != types.CPtr {
		t.Fatal("CPtr PtrKind lost across exportdata round-trip")
	}
}

func TestMetaLinkSnapshotDepfileAndErrors(t *testing.T) {
	meta := &exportdata.PackageMeta{
		Summary: ssa.PackageSummary{
			Path:            "example.com/p",
			TargetTriple:    "arm64-apple-macosx",
			Exports:         map[string]string{"p.A": "p_a"},
			DefinedFuncs:    []string{"p.A", "p.B"},
			DirectCalls:     []string{"fmt.Println"},
			ClosureRefs:     []string{"p.A$1"},
			InterfaceCalls:  []string{"String"},
			HasSafepoints:   true,
			HasWriteBarrier: true,
		},
		DefinedGlobals: []string{"p.G"},
		Escape: exportdata.EscapeSummary{
			PkgPath: "example.com/p",
			Funcs: []exportdata.FuncEscape{
				{Name: "p.A", Params: []exportdata.ParamEscapeFlags{exportdata.ParamNoEscape, exportdata.ParamToHeap}, StackAllocs: 2, HeapAllocs: 1},
			},
		},
		Inline: exportdata.InlineSummary{
			PkgPath: "example.com/p",
			Budget:  80,
			Funcs: []exportdata.FuncInline{
				{Name: "p.A", Cost: 12, Inlineable: true, Callees: []string{"p.B"}},
			},
		},
		CallEdges: []exportdata.CallEdge{
			{Caller: "p.A", Callee: "p.B", Kind: 0},
		},
		LiveTypes:   []string{"*p.T"},
		ExportFuncs: []string{"p.A"},
	}
	var mbuf bytes.Buffer
	if err := exportdata.WriteMeta(&mbuf, meta); err != nil {
		t.Fatal(err)
	}
	metaGot, err := exportdata.ReadMeta(&mbuf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(meta, metaGot) {
		t.Fatalf("PackageMeta round-trip mismatch:\ngot:  %+v\nwant: %+v", metaGot, meta)
	}

	snap := &exportdata.LinkSnapshot{
		PkgPath:         "example.com/p",
		PkgName:         "p",
		NeedRuntime:     true,
		NeedAbiInit:     true,
		HasLocalExports: true,
		MethodByIndex:   true,
		MethodByName:    []string{"Read", "Write"},
		AbiSymbols:      []string{"_llgo_p.T"},
		AbiTypes:        []string{"p.T"},
		CExports:        []string{"p_init"},
		Imports:         []string{"runtime"},
		PCLNFuncs: []exportdata.PCLNFunc{
			{
				Symbol: "p.A", Name: "example.com/p.A", File: "a.go", Line: 10, StartPC: 0x100, EndPC: 0x140,
				Entries: []exportdata.PCLNEntry{{PC: 0, Line: 10, File: "a.go"}, {PC: 16, Line: 12, File: "a.go"}},
			},
		},
	}
	var sbuf bytes.Buffer
	if err := exportdata.WriteLinkSnapshot(&sbuf, snap); err != nil {
		t.Fatal(err)
	}
	snapGot, err := exportdata.ReadLinkSnapshot(&sbuf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snap, snapGot) {
		t.Fatalf("LinkSnapshot round-trip mismatch:\ngot:  %+v\nwant: %+v", snapGot, snap)
	}

	// Depfile stat validation.
	dir := t.TempDir()
	hFile := filepath.Join(dir, "foo.h")
	if err := os.WriteFile(hFile, []byte("#define X 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stats, err := exportdata.StatFiles([]string{hFile, hFile})
	if err != nil || len(stats) != 1 {
		t.Fatalf("StatFiles = (%v, %v)", stats, err)
	}
	df := &exportdata.Depfile{Key: "demo", PreprocessedHash: "abc123", Files: stats}
	dfRaw, err := exportdata.MarshalDepfile(df)
	if err != nil {
		t.Fatal(err)
	}
	dfGot, err := exportdata.UnmarshalDepfile(dfRaw)
	if err != nil || !exportdata.ValidateDepfile(dfGot) {
		t.Fatalf("ValidateDepfile failed after round-trip: %+v err=%v", dfGot, err)
	}
	// Mutate header content + timestamp -> ValidateDepfile must return false.
	future := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(hFile, []byte("#define X 200\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(hFile, future, future)
	if exportdata.ValidateDepfile(dfGot) {
		t.Fatal("ValidateDepfile should return false after header modification")
	}

	// Error cases: bad magic, wrong kind, bad version, truncated payload.
	if _, err := exportdata.UnmarshalMeta([]byte("BADMAG\x02\x01\x00\x00\x00")); !errors.Is(err, exportdata.ErrInvalidMagic) {
		t.Fatalf("expected ErrInvalidMagic, got %v", err)
	}
	if _, err := exportdata.UnmarshalLinkSnapshot(dfRaw); !errors.Is(err, exportdata.ErrKindMismatch) {
		t.Fatalf("expected ErrKindMismatch, got %v", err)
	}
	badVer := append([]byte(nil), dfRaw...)
	badVer[7] = 99
	if _, err := exportdata.UnmarshalDepfile(badVer); !errors.Is(err, exportdata.ErrUnsupportedVer) {
		t.Fatalf("expected ErrUnsupportedVer, got %v", err)
	}
	if _, err := exportdata.UnmarshalPackage([]byte("LL"), nil); !errors.Is(err, exportdata.ErrCorrupt) {
		t.Fatalf("expected ErrCorrupt, got %v", err)
	}
}

type unsafeOnlyImporter struct{}

func (unsafeOnlyImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	return nil, errors.New("not found")
}
