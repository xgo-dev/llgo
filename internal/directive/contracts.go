// Package funcattrs validates attributes on whole Go parameters and results.
package directive

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
)

type Scope string

const (
	FunctionScope Scope = "function"
	Parameter     Scope = "parameter"
	Result        Scope = "result"
	Receiver      Scope = "receiver"
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
	Pos      token.Pos
	Position token.Position
	Range    *RangeBounds
	From     *Target // Ordinary parameter entry value used by sameas.
}

func (a Attribute) Error(format string, args ...any) error {
	return &ContractError{Pos: a.Pos, Position: a.Position, Message: fmt.Sprintf(format, args...)}
}

// IsSourceDirective distinguishes attributes from existing LLGo directives.
func IsSourceDirective(d Directive) bool {
	name := strings.TrimPrefix(d.Name, "llgo:")
	if name == d.Name {
		return false
	}
	if i := strings.IndexAny(name, "(."); i >= 0 {
		name = name[:i]
	}
	switch name {
	case "param", "result", "receiver", "nonnull", "range", "nonnegative", "sameas":
		return true
	}
	return false
}

// Parse resolves source selectors before imported signatures can lose parameter
// names. Type validation is separate so generic instances use concrete types.
func ParseContracts(fset *token.FileSet, decl *ast.FuncDecl, items []Directive) ([]Attribute, error) {
	var attrs []Attribute
	for _, d := range items {
		if !IsSourceDirective(d) {
			continue
		}
		base := Attribute{Target: Target{Scope: FunctionScope}, Pos: d.Pos, Position: contractPosition(fset, d.Pos)}

		words, err := split(strings.TrimPrefix(d.Name, "llgo:") + " " + d.Args)
		if err != nil {
			return nil, base.Error("%v", err)
		}
		if isSelector(words[0]) {
			base.Target, err = parseTarget(decl, words[0])
			if err != nil {
				return nil, base.Error("%v", err)
			}

			words = words[1:]
			if len(words) == 0 {
				return nil, base.Error("expected an attribute after selector")
			}
		} else if len(words) != 1 {
			return nil, base.Error("each function attribute must be written on its own //llgo: line")
		}
		for _, word := range words {
			a := base
			a.Name, a.Args, err = expression(word)
			if err != nil {
				return nil, a.Error("%v", err)
			}

			if a.Name == "sameas" {
				if !token.IsIdentifier(a.Args) || a.Args == "_" {
					return nil, a.Error("sameas expects a parameter name")
				}
				index, err := selectIndex(decl.Type.Params, a.Args)
				if err != nil {
					return nil, a.Error("sameas: %v", err)
				}
				a.From = &Target{Scope: Parameter, Index: index}
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

// expression separates an attribute name from its parenthesized arguments.
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
	if s == "result" {
		if fieldCount(decl.Type.Results) != 1 {
			return target, fmt.Errorf("result shorthand requires exactly one source result")
		}
		return Target{Scope: Result}, nil
	}
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

// normalize is the only string-to-contract translation point. It also populates
// typed operands for programmatically constructed attributes passed to Merge.
func normalize(a Attribute) (Attribute, error) {
	a.Args = strings.TrimSpace(a.Args)
	input := a.Target.Scope == Parameter || a.Target.Scope == Receiver
	result := a.Target.Scope == Result
	valid, takesArgs := false, false
	switch a.Name {
	case "nonnull", "nonnegative":
		valid = input || result
	case "range":
		valid, takesArgs = input || result, true
	case "sameas":
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
	case "range":
		var lo, hi *big.Int
		lo, hi, err = bounds(a.Args)
		if err == nil {
			a.Range = &RangeBounds{Lower: lo, Upper: hi}
			a.Args = lo.String() + "," + hi.String()
		}

	case "sameas":
		if a.From == nil || a.From.Scope != Parameter {
			err = fmt.Errorf("sameas requires a parameter name")
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
		if a.Target.Scope == FunctionScope {
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
		case "nonnull":
			if !pointer(t) {
				return a.Error("%s requires a pointer, got %s", a.Name, t)
			}

		case "sameas":
			if !pointer(t) && !integer(t) {
				return a.Error("sameas requires an integer or pointer result, got %s", t)
			}
			if a.From == nil {
				return a.Error("sameas requires an input selector")
			}
			from, err := ResolveTarget(sig, *a.From)
			if err != nil {
				return a.Error("sameas input: %v", err)
			}
			if unresolved(from) && deferTypeParams {
				continue
			}
			if !(pointer(t) && pointer(from)) && !types.Identical(types.Unalias(t), types.Unalias(from)) {
				return a.Error("sameas requires compatible pointer types or identical integer source types, got %s and %s", from, t)
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

// ContractError retains token identity until a compilation supplies its FileSet.
type ContractError struct {
	Pos      token.Pos
	Position token.Position
	Message  string
}

func (e *ContractError) Error() string { return fmt.Sprintf("%s: llgo: %s", e.Position, e.Message) }
func contractPosition(fset *token.FileSet, pos token.Pos) token.Position {
	if fset == nil {
		return token.Position{}
	}
	return fset.Position(pos)
}

// WithPositions returns a view with diagnostic positions. Published records
// remain immutable and contain no LLVM values.
func (f Function) WithPositions(fset *token.FileSet) Function {
	f.Values = slices.Clone(f.Values)
	for i := range f.Values {
		f.Values[i].Position = contractPosition(fset, f.Values[i].Pos)
	}
	if e, ok := f.ContractError.(*ContractError); ok {
		copy := *e
		copy.Position = contractPosition(fset, e.Pos)
		f.ContractError = &copy
	}
	return f
}
func functionRecord(decl *ast.FuncDecl, group *Group) Function {
	f := group.Function
	f.Values, f.ContractError = ParseContracts(nil, decl, group.Items)
	return f
}
