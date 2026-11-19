package binaryspecialize

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

type rewrite struct {
	call      *ast.CallExpr
	op        string
	order     types.Type
	input     types.Type
	wire      *layout
	ptr       bool
	slice     bool
	unchecked bool
	helper    string
}

// Rewrite specializes a program only when it accounts for all Go uses of
// the generic encoding/binary operations. Mixed uses keep one implementation.
func Rewrite(closure []*packages.Package, library, entry *packages.Package, shortBuffer string) ([]*packages.Package, error) {
	var planned []packageRewrites
	needsError := false
	for _, pkg := range closure {
		if pkg == library || pkg.Imports["encoding/binary"] == nil {
			continue
		}
		if pkg.Imports["encoding/binary"].Types != library.Types || pkg.TypesInfo == nil {
			return nil, nil
		}
		calls, ok := planPackage(pkg, library.Types)
		if !ok {
			return nil, nil
		}
		if len(calls) != 0 {
			if pkg != entry {
				return nil, nil
			}
			planned = append(planned, packageRewrites{pkg, calls})
			for _, call := range calls {
				needsError = needsError || call.op != "Size" && !call.unchecked
			}
		}
	}
	if len(planned) == 0 {
		return nil, nil
	}

	if needsError && shortBuffer == "" {
		return nil, nil
	}
	var changed []*packages.Package
	for _, group := range planned {
		if err := group.apply(library, shortBuffer); err != nil {
			return nil, err
		}
		changed = append(changed, group.pkg)
	}
	return changed, nil
}

type packageRewrites struct {
	pkg   *packages.Package
	calls []rewrite
}

func planPackage(pkg *packages.Package, library *types.Package) ([]rewrite, bool) {
	if pkg.Types == nil || pkg.TypesSizes == nil || len(pkg.Syntax) == 0 {
		return nil, false
	}
	maxWidth := int64(math.MaxInt64)
	if pkg.TypesSizes.Sizeof(types.Typ[types.Int]) == 4 {
		maxWidth = math.MaxInt32
	}
	info := pkg.TypesInfo
	accepted := make(map[*ast.Ident]bool)
	var calls []rewrite
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || call.Ellipsis.IsValid() {
				return true
			}
			id := calleeIdent(call.Fun)
			fn, _ := info.Uses[id].(*types.Func)
			if fn != nil && fn.Pkg() == library && genericOperation(fn.Name()) {
				if candidate, ok := planCall(pkg, call, fn.Name(), library, maxWidth); ok {
					calls = append(calls, candidate)
					accepted[id] = true
				}
			}
			return true
		})
	}
	for id, obj := range info.Uses {
		if fn, ok := obj.(*types.Func); ok && fn.Pkg() == library && genericOperation(fn.Name()) && !accepted[id] {
			return nil, false
		}
	}
	if len(calls) != 0 && shadowsGeneratedBuiltins(pkg.Types) {
		return nil, false
	}
	return calls, true
}

func genericOperation(name string) bool {
	switch name {
	case "Append", "Decode", "Encode", "Read", "Size", "Write":
		return true
	}
	return false
}

func calleeIdent(expr ast.Expr) *ast.Ident {
	switch expr := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return expr
	case *ast.SelectorExpr:
		return expr.Sel
	}
	return nil
}

func planCall(pkg *packages.Package, call *ast.CallExpr, op string, library *types.Package, maxWidth int64) (rewrite, bool) {
	r := rewrite{call: call, op: op}
	switch {
	case op == "Size" && len(call.Args) == 1:
	case (op == "Encode" || op == "Decode") && len(call.Args) == 3:
		r.order = pkg.TypesInfo.TypeOf(call.Args[1])
		if !standardByteOrder(r.order, library) {
			return r, false
		}
	default:
		return r, false
	}
	operand := call.Args[len(call.Args)-1]
	r.input = pkg.TypesInfo.TypeOf(operand)
	if r.input == nil || llssa.HasTypeParam(r.input) {
		return r, false
	}
	value := r.input
	if ptr, ok := value.Underlying().(*types.Pointer); ok {
		if op != "Size" && !nonNilPointer(pkg.TypesInfo, operand) {
			return r, false
		}
		r.ptr = true
		value = ptr.Elem()
	}
	if slice, ok := value.Underlying().(*types.Slice); ok {
		r.slice = true
		value = slice.Elem()
	} else if op == "Decode" && !r.ptr {
		return r, false
	}
	// Basic operands already have fast paths with distinct nil, overlap and
	// floating-point behavior. Start at the fixed-size composite fallback.
	switch value.Underlying().(type) {
	case *types.Array, *types.Struct:
	default:
		return r, false
	}
	r.wire = fixedLayout(value, maxWidth)
	if r.wire == nil || op != "Size" && !canAccess(r.wire, pkg.Types, op == "Decode") {
		return r, false
	}
	if op != "Size" && !r.slice {
		r.unchecked = r.wire.width == 0 || arraySliceLength(pkg.TypesInfo, call.Args[0]) >= r.wire.width
	}
	return r, true
}

// A slice of an array has fixed bounds when both indices are constant. The
// argument is still evaluated, including any nil-pointer and bounds checks.
func arraySliceLength(info *types.Info, expr ast.Expr) int64 {
	slice, ok := ast.Unparen(expr).(*ast.SliceExpr)
	if !ok {
		return -1
	}
	typ := info.TypeOf(slice.X)
	if typ == nil {
		return -1
	}
	base := typ.Underlying()
	if ptr, ok := base.(*types.Pointer); ok {
		base = ptr.Elem().Underlying()
	}
	array, ok := base.(*types.Array)
	if !ok {
		return -1
	}
	lo, hi := int64(0), array.Len()
	for i, index := range []ast.Expr{slice.Low, slice.High} {
		if index == nil {
			continue
		}
		value := info.Types[index].Value
		if value == nil || value.Kind() != constant.Int {
			return -1
		}
		n, ok := constant.Int64Val(value)
		if !ok {
			return -1
		}
		if i == 0 {
			lo = n
		} else {
			hi = n
		}
	}
	if lo < 0 || hi < lo || hi > array.Len() {
		return -1
	}
	return hi - lo
}

func standardByteOrder(typ types.Type, library *types.Package) bool {
	if typ == nil {
		return false
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() != library {
		return false
	}
	switch named.Obj().Name() {
	case "littleEndian", "bigEndian", "nativeEndian":
		return true
	}
	return false
}

func nonNilPointer(info *types.Info, expr ast.Expr) bool {
	switch expr := ast.Unparen(expr).(type) {
	case *ast.UnaryExpr:
		id := calleeIdent(expr.X)
		v, _ := info.Uses[id].(*types.Var)
		// A variable's own address is non-nil. Selectors on pointer
		// receivers can need evaluation and nil checks along the path.
		_, local := ast.Unparen(expr.X).(*ast.Ident)
		return expr.Op == token.AND && local && v != nil
	case *ast.CallExpr:
		fn, _ := info.Uses[calleeIdent(expr.Fun)].(*types.Builtin)
		return fn != nil && fn.Name() == "new"
	}
	return false
}

func canAccess(wire *layout, owner *types.Package, decode bool) bool {
	for _, f := range wire.fields {
		if f.field.Name() == "_" {
			continue
		}
		if !f.field.Exported() && (decode || f.field.Pkg() != owner) || !canAccess(f.value, owner, decode) {
			return false
		}
	}
	return wire.elem == nil || canAccess(wire.elem, owner, decode)
}

func shadowsGeneratedBuiltins(pkg *types.Package) bool {
	for _, name := range []string{"bool", "byte", "complex", "error", "false", "float32",
		"float64", "imag", "int", "len", "nil", "real", "true", "uint16", "uint32", "uint64"} {
		if pkg.Scope().Lookup(name) != nil {
			return true
		}
	}
	return false
}
