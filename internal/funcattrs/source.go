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

// Target identifies a whole source parameter, receiver, result or function.
type Target struct {
	Scope Scope
	Index int
}

func (t Target) Equal(other Target) bool {
	return t.Scope == other.Scope && t.Index == other.Index
}

func (t Target) String() string {
	var s string
	switch t.Scope {
	case Parameter:
		s = fmt.Sprintf("param(%d)", t.Index)
	case Result:
		s = fmt.Sprintf("result(%d)", t.Index)
	default:
		s = string(t.Scope)
	}
	return s
}

type CaptureMode uint8

const (
	CaptureNone CaptureMode = iota
	CaptureResults
	CaptureAny
)

// RangeBounds contains mathematical integers with an exclusive upper bound.
// It stays source-width independent until IntegerRange validates a concrete type.
type RangeBounds struct {
	Lower *big.Int
	Upper *big.Int
}

type Attribute struct {
	Target   Target
	Name     string
	Args     string // Canonical spelling; backends consume the typed operands below.
	Position token.Position
	Range    *RangeBounds
	From     *Target // same_as denotes the logical input snapshot at function entry.
	Access   AccessMode
	Capture  CaptureMode
	Memory   MemoryEffects
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
		base := Attribute{Target: Target{Scope: Function}, Position: fset.Position(d.Pos)}
		words, err := split(d.Args)
		if err != nil {
			return nil, base.Error("%v", err)
		}
		if len(words) == 0 {
			return nil, base.Error("expected an attribute")
		}
		if isSelector(words[0]) {
			base.Target, err = parseTarget(decl, words[0])
			if err != nil {
				return nil, base.Error("%v", err)
			}
			words = words[1:]
		}
		if len(words) == 0 {
			return nil, base.Error("expected an attribute after selector")
		}
		for _, word := range words {
			a := base
			a.Name, a.Args, err = expression(word)
			if err != nil {
				return nil, a.Error("%v", err)
			}
			if a.Name == "same_as" {
				from, err := parseTarget(decl, a.Args)
				if err != nil {
					return nil, a.Error("same_as: %v", err)
				}
				a.From = &from
			}
			if a.Name == "returned" {
				if a.Args != "" || (a.Target.Scope != Parameter && a.Target.Scope != Receiver) {
					return nil, a.Error("returned requires an input selector and no arguments")
				}
				if fieldCount(decl.Type.Results) != 1 {
					return nil, a.Error("returned requires exactly one source result; use result(...) same_as(...) otherwise")
				}
				from := a.Target
				a.Target, a.Name, a.From = Target{Scope: Result}, "same_as", &from
				a.Args = from.String()
			}
			a, err = normalize(a)
			if err != nil {
				return nil, err
			}
			attrs = append(attrs, a)
		}
	}
	return Merge(attrs)
}

// Reject reports attributes attached outside named function declarations.
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

// expression also accepts nesting for same_as(param(...)).
func expression(s string) (name, args string, err error) {
	name = s
	if i := strings.IndexByte(s, '('); i >= 0 {
		end, e := closingParen(s, i)
		if e != nil || end != len(s)-1 {
			return "", "", fmt.Errorf("invalid attribute expression %q", s)
		}
		name, args = s[:i], strings.TrimSpace(s[i+1:end])
		if args == "" {
			return "", "", fmt.Errorf("empty arguments for %s", name)
		}
	}
	if name != "range" && !token.IsIdentifier(name) {
		return "", "", fmt.Errorf("invalid attribute expression %q", s)
	}
	return
}

func closingParen(s string, start int) (int, error) {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced parentheses")
}

func isSelector(s string) bool {
	return s == "receiver" || strings.HasPrefix(s, "receiver.") || strings.HasPrefix(s, "receiver(") ||
		s == "param" || strings.HasPrefix(s, "param(") || s == "result" || strings.HasPrefix(s, "result(")
}

func parseTarget(decl *ast.FuncDecl, s string) (Target, error) {
	var target Target
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "receiver") {
		if decl.Recv == nil {
			return target, fmt.Errorf("receiver requires a method")
		}
		target.Scope, s = Receiver, s[len("receiver"):]
	} else {
		i := strings.IndexByte(s, '(')
		if i < 0 {
			return target, fmt.Errorf("expected param(...), result(...), or receiver selector")
		}
		end, err := closingParen(s, i)
		if err != nil {
			return target, err
		}
		var fields *ast.FieldList
		switch s[:i] {
		case "param":
			target.Scope, fields = Parameter, decl.Type.Params
		case "result":
			target.Scope, fields = Result, decl.Type.Results
		default:
			return target, fmt.Errorf("expected an input or result selector, got %q", s[:i])
		}
		target.Index, err = selectIndex(fields, strings.TrimSpace(s[i+1:end]))
		if err != nil {
			return target, err
		}
		s = s[end+1:]
	}
	if s != "" {
		return target, fmt.Errorf("unsupported selector suffix %q: select the whole parameter or result", s)
	}
	return target, nil
}

func fieldNames(fields *ast.FieldList) []string {
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
	return names
}

func fieldCount(fields *ast.FieldList) int { return len(fieldNames(fields)) }

func selectIndex(fields *ast.FieldList, selector string) (int, error) {
	names := fieldNames(fields)
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
func accessString(mode AccessMode) string {
	return [...]string{"none", "read", "write", "readwrite"}[mode]
}

// normalize is the only string-to-contract translation point. It also populates
// typed operands for programmatically constructed attributes passed to Merge.
func normalize(a Attribute) (Attribute, error) {
	a.Args = strings.TrimSpace(a.Args)
	switch a.Name {
	case "readonly", "writeonly":
		if a.Args != "" {
			return a, a.Error("invalid arguments for %s", a.Name)
		}
		if a.Name == "readonly" {
			a.Args = "read"
		} else {
			a.Args = "write"
		}
		a.Name = "access"
	case "captures":
		a.Name = "capture"
		if compact(a.Args) == "ret:address,provenance" {
			a.Args = "results"
		}
	case "memory":
		parts := strings.Split(a.Args, ",")
		for i, part := range parts {
			location, mode, ok := strings.Cut(part, ":")
			if ok && strings.TrimSpace(location) == "argmem" {
				parts[i] = "args:" + mode
			}
		}
		a.Args = strings.Join(parts, ",")
	}
	function := a.Target.Scope == Function
	input := a.Target.Scope == Parameter || a.Target.Scope == Receiver
	result := a.Target.Scope == Result
	valid, takesArgs := false, false
	switch a.Name {
	case "cold", "noreturn":
		valid = function
	case "memory":
		valid, takesArgs = function, true
	case "access", "capture":
		valid, takesArgs = input, true
	case "nonnull", "nonnegative":
		valid = input || result
	case "range":
		valid, takesArgs = input || result, true
	case "same_as":
		valid, takesArgs = result, true
	default:
		return a, a.Error("unsupported attribute %q", a.Name)
	}
	if !valid {
		return a, a.Error("%s is not supported on %s", a.Name, a.Target.Scope)
	}
	if takesArgs != (a.Args != "") {
		return a, a.Error("invalid arguments for %s", a.Name)
	}
	var err error
	switch a.Name {
	case "memory":
		a.Memory, err = ParseMemoryEffects(a.Args)
		if err == nil {
			a.Args = "args:" + accessString(a.Memory.Args) + ",other:" + accessString(a.Memory.Other)
		}
	case "access":
		a.Access, err = ParseAccessMode(a.Args)
	case "capture":
		switch a.Args {
		case "none":
			a.Capture = CaptureNone
		case "results":
			a.Capture = CaptureResults
		case "any":
			a.Capture = CaptureAny
		default:
			err = fmt.Errorf("unsupported capture effects %q", a.Args)
		}
	case "range":
		var lo, hi *big.Int
		lo, hi, err = bounds(a.Args)
		if err == nil {
			a.Range = &RangeBounds{Lower: lo, Upper: hi}
			a.Args = lo.String() + "," + hi.String()
		}
	case "same_as":
		if a.From == nil || (a.From.Scope != Parameter && a.From.Scope != Receiver) {
			err = fmt.Errorf("same_as requires an input param(...) or receiver selector")
		} else {
			a.Args = a.From.String()
		}
	}
	if err != nil {
		return a, a.Error("%v", err)
	}
	return a, nil
}

// Merge checks repeated declarations deterministically and owns copies of all
// mutable operands. Source spellings are canonicalized before conflict checks.
func Merge(attrs ...[]Attribute) ([]Attribute, error) {
	var out []Attribute
	for _, list := range attrs {
		for _, a := range list {
			var err error
			a, err = normalize(a)
			if err != nil {
				return nil, err
			}
			if a.From != nil {
				from := *a.From
				a.From = &from
			}
			duplicate := false
			for _, prev := range out {
				if !prev.Target.Equal(a.Target) {
					continue
				}
				if prev.Name == a.Name {
					if prev.Args != a.Args {
						return nil, a.Error("conflicting %s (previous declaration at %s)", a.Name, prev.Position)
					}
					duplicate = true
				}
			}
			if !duplicate {
				out = append(out, a)
			}
		}
	}
	slices.SortFunc(out, func(a, b Attribute) int {
		if n := strings.Compare(a.Target.String(), b.Target.String()); n != 0 {
			return n
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

// ResolveTarget returns the Go type of the selected whole input or result.
func ResolveTarget(sig *types.Signature, target Target) (types.Type, error) {
	t := rootType(sig, target)
	if t == nil {
		return nil, fmt.Errorf("selector %s does not exist in this signature", target)
	}
	return t, nil
}

func rootType(sig *types.Signature, target Target) types.Type {
	if sig == nil || target.Index < 0 {
		return nil
	}
	switch target.Scope {
	case Receiver:
		if sig.Recv() != nil && target.Index == 0 {
			return sig.Recv().Type()
		}
	case Parameter:
		if target.Index < sig.Params().Len() {
			return sig.Params().At(target.Index).Type()
		}
	case Result:
		if target.Index < sig.Results().Len() {
			return sig.Results().At(target.Index).Type()
		}
	}
	return nil
}

func valueType(sig *types.Signature, target Target) types.Type {
	t, _ := ResolveTarget(sig, target)
	return t
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

func integer(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}

func unresolved(t types.Type) bool {
	_, ok := types.Unalias(t).(*types.TypeParam)
	return ok
}

// Validate checks concrete source types. Preloading may defer type-dependent
// checks until a generic instance is lowered with concrete type arguments.
func Validate(attrs []Attribute, sig *types.Signature, intBits int, deferTypeParams bool) error {
	for _, a := range attrs {
		if a.Target.Scope == Function {
			continue
		}
		t, err := ResolveTarget(sig, a.Target)
		if err != nil {
			return a.Error("%v", err)
		}
		if unresolved(t) && deferTypeParams {
			continue
		}
		switch a.Name {
		case "nonnull", "access", "capture":
			if !pointer(t) {
				return a.Error("%s requires a pointer, got %s", a.Name, t)
			}
		case "same_as":
			if !pointer(t) && !integer(t) {
				return a.Error("same_as requires an integer or pointer result, got %s", t)
			}
			if a.From == nil {
				return a.Error("same_as requires an input selector")
			}
			from, err := ResolveTarget(sig, *a.From)
			if err != nil {
				return a.Error("same_as input: %v", err)
			}
			if unresolved(from) && deferTypeParams {
				continue
			}
			if !(pointer(t) && pointer(from)) && !types.Identical(types.Unalias(t), types.Unalias(from)) {
				return a.Error("same_as requires compatible pointer types or identical integer source types, got %s and %s", from, t)
			}
		case "range", "nonnegative":
			if _, _, _, err := IntegerRange(a, t, intBits); err != nil {
				return err
			}
			if a.Name == "range" && a.Range != nil && a.Range.Upper.Sign() <= 0 {
				for _, other := range attrs {
					if other.Name == "nonnegative" && other.Target.Equal(a.Target) {
						return a.Error("conflicting range and nonnegative contracts")
					}
				}
			}
		}
	}
	return nil
}

// IntegerRange returns target-width bit-pattern bounds, not source indices.
// full reports that the source promise covers the whole integer domain.
func IntegerRange(a Attribute, t types.Type, intBits int) (bits int, values [2]uint64, full bool, err error) {
	if t == nil || !integer(t) {
		return 0, values, false, a.Error("%s requires an integer value", a.Name)
	}
	b := t.Underlying().(*types.Basic)
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
	if bits != 8 && bits != 16 && bits != 32 && bits != 64 {
		return 0, values, false, a.Error("unsupported target integer width %d", bits)
	}
	unsigned := b.Info()&types.IsUnsigned != 0
	min, max := new(big.Int), new(big.Int).Lsh(big.NewInt(1), uint(bits))
	if !unsigned {
		max.Rsh(max, 1)
		min.Neg(max)
	}
	lo, hi := new(big.Int), new(big.Int).Set(max)
	if a.Name == "range" {
		if a.Range == nil || a.Range.Lower == nil || a.Range.Upper == nil {
			return 0, values, false, a.Error("range has no parsed integer bounds")
		}
		lo, hi = a.Range.Lower, a.Range.Upper
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
