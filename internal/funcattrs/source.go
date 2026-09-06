// Package funcattrs describes source-level function contracts independently of
// the LLVM signature used to implement a function.
package funcattrs

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/xgo-dev/llgo/internal/directive"
)

type Scope string

const (
	Function  Scope = "function"
	Parameter Scope = "parameter"
	Result    Scope = "result"
	Receiver  Scope = "receiver"
)

// Target identifies a logical source value. Index never includes compiler-owned
// arguments and remains meaningful if an ABI splits or stores that value.
type Target struct {
	Scope Scope
	Index int
}

type Attribute struct {
	Target   Target
	Name     string
	Args     string
	Position token.Position
}

func (a Attribute) Error(format string, args ...any) error {
	return fmt.Errorf("%s: llgo:attribute: %s", a.Position, fmt.Sprintf(format, args...))
}

// Parse resolves source selectors before imported signatures can lose parameter
// names. Type validation is separate so generic instances use concrete types.
func Parse(fset *token.FileSet, decl *ast.FuncDecl) ([]Attribute, error) {
	var attrs []Attribute
	for _, d := range directive.ParseGroup(decl.Doc) {
		if d.Name != "llgo:attribute" {
			continue
		}
		a := Attribute{Target: Target{Scope: Function}, Position: fset.Position(d.Pos)}
		words, err := split(d.Args)
		if err != nil {
			return nil, a.Error("%v", err)
		}
		if len(words) == 0 {
			return nil, a.Error("expected an attribute")
		}
		name, args, err := expression(words[0])
		if err != nil {
			return nil, a.Error("%v", err)
		}
		switch name {
		case "param", "result":
			fields := decl.Type.Params
			a.Target.Scope = Parameter
			if name == "result" {
				fields, a.Target.Scope = decl.Type.Results, Result
			}
			index, err := selectIndex(fields, args)
			if err != nil {
				return nil, a.Error("%s: %v", name, err)
			}
			a.Target.Index = index
			words = words[1:]
		case "receiver":
			if args != "" || words[0] != "receiver" || decl.Recv == nil {
				return nil, a.Error("receiver requires a method")
			}
			a.Target.Scope = Receiver
			words = words[1:]
		}
		if len(words) == 0 {
			return nil, a.Error("expected an attribute after selector")
		}
		for _, word := range words {
			a.Name, a.Args, err = expression(word)
			if err != nil {
				return nil, a.Error("%v", err)
			}
			if err = checkSyntax(a); err != nil {
				return nil, err
			}
			attrs = append(attrs, a)
		}
	}
	return Merge(attrs)
}

// Reject reports attributes attached to declarations/locations outside v1.
func Reject(fset *token.FileSet, doc *ast.CommentGroup) error {
	for _, d := range directive.ParseGroup(doc) {
		if d.Name == "llgo:attribute" {
			return (Attribute{Position: fset.Position(d.Pos)}).Error("requires a named function or method declaration")
		}
	}
	return nil
}

func split(s string) ([]string, error) {
	var words []string
	depth, start := 0, -1
	for i, r := range s {
		if unicode.IsSpace(r) && depth == 0 {
			if start >= 0 {
				words = append(words, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses")
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced parentheses")
	}
	if start >= 0 {
		words = append(words, s[start:])
	}
	return words, nil
}

func expression(s string) (name, args string, err error) {
	name = s
	if i := strings.IndexByte(s, '('); i >= 0 {
		if !strings.HasSuffix(s, ")") || strings.ContainsAny(s[i+1:len(s)-1], "()") {
			return "", "", fmt.Errorf("invalid attribute expression %q", s)
		}
		name, args = s[:i], strings.TrimSpace(s[i+1:len(s)-1])
		if args == "" {
			return "", "", fmt.Errorf("empty arguments for %s", name)
		}
	}
	return
}

func selectIndex(fields *ast.FieldList, selector string) (int, error) {
	var names []string
	if fields != nil {
		for _, f := range fields.List {
			if len(f.Names) == 0 {
				names = append(names, "")
			}
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
		}
	}
	if i, err := strconv.Atoi(selector); err == nil && i >= 0 && i < len(names) {
		return i, nil
	}
	if selector != "" && selector != "_" {
		if i := slices.Index(names, selector); i >= 0 {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown source value %q", selector)
}

func compact(s string) string { return strings.Join(strings.Fields(s), "") }

func checkSyntax(a Attribute) error {
	function := a.Target.Scope == Function
	input := a.Target.Scope == Parameter || a.Target.Scope == Receiver
	result := a.Target.Scope == Result
	valid, takesArgs := false, false
	switch a.Name {
	case "cold", "noreturn", "nounwind", "willreturn", "nofree", "nosync":
		valid = function
	case "memory":
		valid, takesArgs = function, true
	case "readonly", "writeonly", "returned":
		valid = input
	case "captures":
		valid, takesArgs = input, true
	case "nonnull":
		valid = input || result
	case "range":
		valid, takesArgs = result, true
	case "nonnegative":
		valid = result
	default:
		return a.Error("unsupported attribute %q", a.Name)
	}
	if !valid {
		return a.Error("%s is not supported on %s", a.Name, a.Target.Scope)
	}
	if takesArgs != (a.Args != "") {
		return a.Error("invalid arguments for %s", a.Name)
	}
	switch a.Name {
	case "memory":
		switch compact(a.Args) {
		case "read", "argmem:read", "argmem:readwrite", "read,argmem:readwrite":
		default:
			return a.Error("unsupported memory effects %q", a.Args)
		}
	case "captures":
		switch compact(a.Args) {
		case "none", "ret:address,provenance":
		default:
			return a.Error("unsupported capture effects %q", a.Args)
		}
	case "range":
		if _, _, err := bounds(a.Args); err != nil {
			return a.Error("%v", err)
		}
	}
	return nil
}

// Merge is deterministic and checks conflicts across both repeated annotations
// and declarations of the same resolved symbol. Returned slices are immutable.
func Merge(attrs ...[]Attribute) ([]Attribute, error) {
	var out []Attribute
	for _, list := range attrs {
		for _, a := range list {
			a.Args = compact(a.Args)
			duplicate := false
			for _, prev := range out {
				if prev.Target != a.Target {
					continue
				}
				if prev.Name == a.Name {
					if prev.Args != a.Args {
						return nil, a.Error("conflicting %s (previous declaration at %s)", a.Name, prev.Position)
					}
					duplicate = true
				}
				if prev.Name == "readonly" && a.Name == "writeonly" || prev.Name == "writeonly" && a.Name == "readonly" || prev.Name == "range" && a.Name == "nonnegative" || prev.Name == "nonnegative" && a.Name == "range" {
					return nil, a.Error("conflicting %s and %s", prev.Name, a.Name)
				}
			}
			if !duplicate {
				out = append(out, a)
			}
		}
	}
	slices.SortFunc(out, func(a, b Attribute) int {
		if n := strings.Compare(string(a.Target.Scope), string(b.Target.Scope)); n != 0 {
			return n
		}
		if a.Target.Index != b.Target.Index {
			return a.Target.Index - b.Target.Index
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out, nil
}

func bounds(args string) (*big.Int, *big.Int, error) {
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("range expects two integer literals")
	}
	lo, ok1 := new(big.Int).SetString(strings.TrimSpace(parts[0]), 0)
	hi, ok2 := new(big.Int).SetString(strings.TrimSpace(parts[1]), 0)
	if !ok1 || !ok2 || lo.Cmp(hi) >= 0 {
		return nil, nil, fmt.Errorf("range requires a nonempty, non-wrapping interval of integer literals")
	}
	return lo, hi, nil
}

func valueType(sig *types.Signature, t Target) types.Type {
	switch t.Scope {
	case Receiver:
		if sig.Recv() != nil {
			return sig.Recv().Type()
		}
	case Parameter:
		if t.Index < sig.Params().Len() {
			return sig.Params().At(t.Index).Type()
		}
	case Result:
		if t.Index < sig.Results().Len() {
			return sig.Results().At(t.Index).Type()
		}
	}
	return nil
}

func pointer(t types.Type) bool {
	if t == nil {
		return false
	}
	switch t := t.Underlying().(type) {
	case *types.Pointer:
		return true
	case *types.Basic:
		return t.Kind() == types.UnsafePointer
	}
	return false
}

// Validate checks a concrete source signature. During preloading, deferTypeParams
// permits type-dependent checks to run when an instance is lowered instead.
func Validate(attrs []Attribute, sig *types.Signature, intBits int, deferTypeParams bool) error {
	returned := false
	for _, a := range attrs {
		if a.Target.Scope == Function {
			continue
		}
		t := valueType(sig, a.Target)
		if t == nil {
			return a.Error("selector does not exist in this signature")
		}
		if a.Target.Scope == Result && sig.Results().Len() != 1 {
			return a.Error("result contracts require a single scalar result in v1")
		}
		if _, ok := types.Unalias(t).(*types.TypeParam); ok && deferTypeParams {
			continue
		}
		switch a.Name {
		case "nonnull", "readonly", "writeonly", "captures":
			if !pointer(t) {
				return a.Error("%s requires a pointer, got %s", a.Name, t)
			}
		case "returned":
			if returned || sig.Results().Len() != 1 {
				return a.Error("returned requires one source value and one scalar result")
			}
			returned = true
			rt := sig.Results().At(0).Type()
			if _, ok := types.Unalias(rt).(*types.TypeParam); ok && deferTypeParams {
				continue
			}
			if !(pointer(t) && pointer(rt)) && !types.Identical(t, rt) {
				return a.Error("returned parameter and result must have compatible scalar types")
			}
			if !pointer(t) {
				if _, ok := t.Underlying().(*types.Basic); !ok {
					return a.Error("returned requires a scalar value")
				}
			}
		case "range", "nonnegative":
			if _, _, _, err := IntegerRange(a, t, intBits); err != nil {
				return err
			}
		}
	}
	return nil
}

// IntegerRange returns target-width bit-pattern bounds, not source indices.
// full reports that the source promise covers the whole integer domain.
func IntegerRange(a Attribute, t types.Type, intBits int) (bits int, values [2]uint64, full bool, err error) {
	b, ok := t.Underlying().(*types.Basic)
	if !ok || b.Info()&types.IsInteger == 0 {
		return 0, values, false, a.Error("%s requires an integer result", a.Name)
	}
	switch b.Kind() {
	case types.Int8, types.Uint8:
		bits = 8
	case types.Int16, types.Uint16:
		bits = 16
	case types.Int32, types.Uint32:
		bits = 32
	case types.Int64, types.Uint64:
		bits = 64
	case types.Int, types.Uint, types.Uintptr:
		bits = intBits
	default:
		return 0, values, false, a.Error("unsupported integer type %s", t)
	}
	unsigned := b.Info()&types.IsUnsigned != 0
	min, max := new(big.Int), new(big.Int).Lsh(big.NewInt(1), uint(bits))
	if !unsigned {
		max.Rsh(max, 1)
		min.Neg(max)
	}
	lo, hi := new(big.Int), new(big.Int).Set(max)
	if a.Name == "range" {
		lo, hi, err = bounds(a.Args)
		if err != nil {
			return 0, values, false, a.Error("%v", err)
		}
	}
	if lo.Cmp(min) < 0 || hi.Cmp(max) > 0 {
		return 0, values, false, a.Error("range does not fit %s on this target", t)
	}
	full = lo.Cmp(min) == 0 && hi.Cmp(max) == 0
	mod := new(big.Int).Lsh(big.NewInt(1), uint(bits))
	values[0] = new(big.Int).Mod(lo, mod).Uint64()
	values[1] = new(big.Int).Mod(hi, mod).Uint64()
	return
}
