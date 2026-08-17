package cl

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/xgo-dev/llgo/internal/goembed"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

const (
	embedInitFS = iota + 1
)

type embedInit struct {
	globalName string
	kind       int
	fsEntries  []goembed.FileData
}

func isStringType(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}

func isByteSliceType(t types.Type) bool {
	s, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	b, ok := types.Unalias(s.Elem()).Underlying().(*types.Basic)
	return ok && (b.Kind() == types.Byte || b.Kind() == types.Uint8)
}

func isEmbedFSType(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "embed" && obj.Name() == "FS"
}

func (p *context) tryEmbedGlobalInit(pkg llssa.Package, gbl *ssa.Global, g llssa.Global, globalName string) bool {
	if len(p.embedMap) == 0 {
		return false
	}
	info, ok := p.embedMap[gbl.Name()]
	if !ok || len(info.Files) == 0 {
		return false
	}
	ptr := types.Unalias(gbl.Type()).(*types.Pointer)
	elem := types.Unalias(ptr.Elem())
	switch {
	case isStringType(elem):
		if len(info.Files) != 1 {
			panic("//go:embed string variable must resolve to one file")
		}
		g.Init(pkg.ConstString(string(info.Files[0].Data)))
		return true
	case isByteSliceType(elem):
		if len(info.Files) != 1 {
			panic("//go:embed []byte variable must resolve to one file")
		}
		g.Init(pkg.ConstBytes(info.Files[0].Data))
		return true
	case isEmbedFSType(elem):
		p.embedInits = append(p.embedInits, embedInit{
			globalName: globalName,
			kind:       embedInitFS,
			fsEntries:  goembed.BuildFSEntries(info.Files),
		})
		return false
	default:
		panic(fmt.Sprintf("go:embed cannot apply to var of type %s", elem.String()))
	}
}

func (p *context) applyEmbedInits(b llssa.Builder) {
	if len(p.embedInits) == 0 {
		return
	}
	sort.Slice(p.embedInits, func(i, j int) bool {
		return p.embedInits[i].globalName < p.embedInits[j].globalName
	})
	inits := p.embedInits
	p.embedInits = nil
	for _, it := range inits {
		g := p.pkg.VarOf(it.globalName)
		if g == nil {
			continue
		}
		switch it.kind {
		case embedInitFS:
			p.applyEmbedFSInit(b, g, it.fsEntries)
		}
	}
}

const (
	embedFSFieldFiles   = 0
	embedFileFieldName  = 0
	embedFileFieldBytes = 1
)

func (p *context) applyEmbedFSInit(b llssa.Builder, g llssa.Global, entries []goembed.FileData) {
	ptr := g.Expr.Type.RawType().(*types.Pointer)
	fsTyp := types.Unalias(ptr.Elem())
	fsStruct := fsTyp.Underlying().(*types.Struct)
	filesPtr := types.Unalias(fsStruct.Field(embedFSFieldFiles).Type()).(*types.Pointer)
	filesSlice := types.Unalias(filesPtr.Elem()).(*types.Slice)
	fileTyp := filesSlice.Elem()
	llFileTyp := p.type_(fileTyp, llssa.InGo)
	llFilesSliceTyp := p.prog.Slice(llFileTyp)

	n := p.prog.IntVal(uint64(len(entries)), p.prog.Int())
	files := b.MakeSlice(llFilesSliceTyp, n, n)
	for i, entry := range entries {
		idx := p.prog.IntVal(uint64(i), p.prog.Int())
		elemPtr := b.IndexAddr(files, idx)
		b.Store(b.FieldAddr(elemPtr, embedFileFieldName), b.Str(entry.Name))
		b.Store(b.FieldAddr(elemPtr, embedFileFieldBytes), b.Str(string(entry.Data)))
	}

	filesPtrVal := b.Alloc(llFilesSliceTyp, true)
	b.Store(filesPtrVal, files)
	b.Store(b.FieldAddr(g.Expr, embedFSFieldFiles), filesPtrVal)
}
