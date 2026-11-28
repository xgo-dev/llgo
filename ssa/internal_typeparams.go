// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

package ssa

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xgo-dev/llgo/types"
)

// Adapted from golang.org/x/tools/internal/typeparams for pure Go 1.20+
// compatibility operating natively on github.com/xgo-dev/llgo/types.

var errEmptyTypeSet = errors.New("empty type set")

func tpIsTypeParam(t types.Type) bool {
	_, ok := types.Unalias(t).(*types.TypeParam)
	return ok
}

func tpCoreType(T types.Type) types.Type {
	U := T.Underlying()
	if _, ok := U.(*types.Interface); !ok {
		return U
	}
	terms, err := tpNormalTerms(U)
	if len(terms) == 0 || err != nil {
		return nil
	}
	U = terms[0].Type().Underlying()
	var identical int
	for identical = 1; identical < len(terms); identical++ {
		if !types.Identical(U, terms[identical].Type().Underlying()) {
			break
		}
	}
	if identical == len(terms) {
		return U
	}
	ch, ok := U.(*types.Chan)
	if !ok {
		return nil
	}
	for chans := identical; chans < len(terms); chans++ {
		curr, ok := terms[chans].Type().Underlying().(*types.Chan)
		if !ok {
			return nil
		}
		if !types.Identical(ch.Elem(), curr.Elem()) {
			return nil
		}
		if ch.Dir() == types.SendRecv {
			ch = curr
		} else if curr.Dir() != types.SendRecv && ch.Dir() != curr.Dir() {
			return nil
		}
	}
	return ch
}

func tpDeref(t types.Type) types.Type {
	if ptr, ok := tpCoreType(t).(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

func tpMustDeref(t types.Type) types.Type {
	if ptr, ok := tpCoreType(t).(*types.Pointer); ok {
		return ptr.Elem()
	}
	panic(fmt.Sprintf("%v is not a pointer", t))
}

func tpNormalTerms(T types.Type) ([]*types.Term, error) {
	typ := types.Unalias(T)
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}
	switch typ := typ.(type) {
	case *types.TypeParam:
		return tpStructuralTerms(typ)
	case *types.Union:
		return tpComputeTermSet(typ)
	case *types.Interface:
		return tpComputeTermSet(typ)
	default:
		return []*types.Term{types.NewTerm(false, T)}, nil
	}
}

func tpStructuralTerms(tparam *types.TypeParam) ([]*types.Term, error) {
	constraint := tparam.Constraint()
	if constraint == nil {
		return nil, fmt.Errorf("%s has nil constraint", tparam)
	}
	iface, _ := constraint.Underlying().(*types.Interface)
	if iface == nil {
		return nil, fmt.Errorf("constraint is %T, not *types.Interface", constraint.Underlying())
	}
	return tpComputeTermSet(iface)
}

func tpComputeTermSet(typ types.Type) ([]*types.Term, error) {
	tset, err := tpComputeTermSetInternal(typ, make(map[types.Type]*tpTermSet), 0)
	if err != nil {
		return nil, err
	}
	if tset.terms.isEmpty() {
		return nil, errEmptyTypeSet
	}
	if tset.terms.isAll() {
		return nil, nil
	}
	var terms []*types.Term
	for _, term := range tset.terms {
		terms = append(terms, types.NewTerm(term.tilde, term.typ))
	}
	return terms, nil
}

type tpTermSet struct {
	complete bool
	terms    tpTermlist
}

func tpComputeTermSetInternal(t types.Type, seen map[types.Type]*tpTermSet, depth int) (res *tpTermSet, err error) {
	if t == nil {
		panic("nil type")
	}
	const maxTermCount = 100
	if tset, ok := seen[t]; ok {
		if !tset.complete {
			return nil, fmt.Errorf("cycle detected in the declaration of %s", t)
		}
		return tset, nil
	}
	tset := new(tpTermSet)
	defer func() {
		tset.complete = true
	}()
	seen[t] = tset

	switch u := t.Underlying().(type) {
	case *types.Interface:
		tset.terms = tpAllTermlist
		for i := 0; i < u.NumEmbeddeds(); i++ {
			embedded := u.EmbeddedType(i)
			if _, ok := embedded.Underlying().(*types.TypeParam); ok {
				return nil, fmt.Errorf("invalid embedded type %T", embedded)
			}
			tset2, err := tpComputeTermSetInternal(embedded, seen, depth+1)
			if err != nil {
				return nil, err
			}
			tset.terms = tset.terms.intersect(tset2.terms)
		}
	case *types.Union:
		tset.terms = nil
		for i := 0; i < u.Len(); i++ {
			ut := u.Term(i)
			var terms tpTermlist
			switch ut.Type().Underlying().(type) {
			case *types.Interface:
				tset2, err := tpComputeTermSetInternal(ut.Type(), seen, depth+1)
				if err != nil {
					return nil, err
				}
				terms = tset2.terms
			case *types.TypeParam, *types.Union:
				return nil, fmt.Errorf("invalid union term %T", ut)
			default:
				if ut.Type() == types.Typ[types.Invalid] {
					continue
				}
				terms = tpTermlist{{ut.Tilde(), ut.Type()}}
			}
			tset.terms = tset.terms.union(terms)
			if len(tset.terms) > maxTermCount {
				return nil, fmt.Errorf("exceeded max term count %d", maxTermCount)
			}
		}
	case *types.TypeParam:
		panic("unreachable")
	default:
		if u != types.Typ[types.Invalid] {
			tset.terms = tpTermlist{{false, t}}
		}
	}
	return tset, nil
}

type tpTerm struct {
	tilde bool
	typ   types.Type
}

func (x *tpTerm) String() string {
	switch {
	case x == nil:
		return "∅"
	case x.typ == nil:
		return "𝓤"
	case x.tilde:
		return "~" + x.typ.String()
	default:
		return x.typ.String()
	}
}

func (x *tpTerm) union(y *tpTerm) (_, _ *tpTerm) {
	switch {
	case x == nil && y == nil:
		return nil, nil
	case x == nil:
		return y, nil
	case y == nil:
		return x, nil
	case x.typ == nil:
		return x, nil
	case y.typ == nil:
		return y, nil
	}
	if x.disjoint(y) {
		return x, y
	}
	if x.tilde || !y.tilde {
		return x, nil
	}
	return y, nil
}

func (x *tpTerm) intersect(y *tpTerm) *tpTerm {
	switch {
	case x == nil || y == nil:
		return nil
	case x.typ == nil:
		return y
	case y.typ == nil:
		return x
	}
	if x.disjoint(y) {
		return nil
	}
	if !x.tilde || y.tilde {
		return x
	}
	return y
}

func (x *tpTerm) disjoint(y *tpTerm) bool {
	ux := x.typ
	if y.tilde {
		ux = ux.Underlying()
	}
	uy := y.typ
	if x.tilde {
		uy = uy.Underlying()
	}
	return !types.Identical(ux, uy)
}

type tpTermlist []*tpTerm

var tpAllTermlist = tpTermlist{new(tpTerm)}

func (xl tpTermlist) String() string {
	if len(xl) == 0 {
		return "∅"
	}
	var buf strings.Builder
	for i, x := range xl {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(x.String())
	}
	return buf.String()
}

func (xl tpTermlist) isEmpty() bool {
	for _, x := range xl {
		if x != nil {
			return false
		}
	}
	return true
}

func (xl tpTermlist) isAll() bool {
	for _, x := range xl {
		if x != nil && x.typ == nil {
			return true
		}
	}
	return false
}

func (xl tpTermlist) norm() tpTermlist {
	used := make([]bool, len(xl))
	var rl tpTermlist
	for i, xi := range xl {
		if xi == nil || used[i] {
			continue
		}
		for j := i + 1; j < len(xl); j++ {
			xj := xl[j]
			if xj == nil || used[j] {
				continue
			}
			if u1, u2 := xi.union(xj); u2 == nil {
				if u1.typ == nil {
					return tpAllTermlist
				}
				xi = u1
				used[j] = true
			}
		}
		rl = append(rl, xi)
	}
	return rl
}

func (xl tpTermlist) union(yl tpTermlist) tpTermlist {
	return append(xl, yl...).norm()
}

func (xl tpTermlist) intersect(yl tpTermlist) tpTermlist {
	if xl.isEmpty() || yl.isEmpty() {
		return nil
	}
	var rl tpTermlist
	for _, x := range xl {
		for _, y := range yl {
			if r := x.intersect(y); r != nil {
				rl = append(rl, r)
			}
		}
	}
	return rl.norm()
}

// tpFree memoizes the set of free type parameters within a type.
type tpFree struct {
	seen map[types.Type]bool
}

func (w *tpFree) Has(typ types.Type) (res bool) {
	if x, ok := w.seen[typ]; ok {
		return x
	}
	if w.seen == nil {
		w.seen = make(map[types.Type]bool)
	}
	w.seen[typ] = false
	defer func() {
		w.seen[typ] = res
	}()

	switch t := typ.(type) {
	case nil, *types.Basic, *types.SIMDVec:
		break
	case *types.Alias:
		if t.TypeParams().Len() > t.TypeArgs().Len() {
			return true
		}
		return w.Has(types.Unalias(t))
	case *types.Array:
		return w.Has(t.Elem())
	case *types.Slice:
		return w.Has(t.Elem())
	case *types.Struct:
		for i, n := 0, t.NumFields(); i < n; i++ {
			if w.Has(t.Field(i).Type()) {
				return true
			}
		}
	case *types.Pointer:
		return w.Has(t.Elem())
	case *types.Tuple:
		for i, n := 0, t.Len(); i < n; i++ {
			if w.Has(t.At(i).Type()) {
				return true
			}
		}
	case *types.Signature:
		return w.Has(t.Params()) || w.Has(t.Results())
	case *types.Interface:
		for i, n := 0, t.NumMethods(); i < n; i++ {
			if w.Has(t.Method(i).Type()) {
				return true
			}
		}
		terms, err := tpComputeTermSet(t)
		if err != nil {
			return false
		}
		for _, term := range terms {
			if w.Has(term.Type()) {
				return true
			}
		}
	case *types.Map:
		return w.Has(t.Key()) || w.Has(t.Elem())
	case *types.Chan:
		return w.Has(t.Elem())
	case *types.Named:
		args := t.TypeArgs()
		if params := t.TypeParams(); params.Len() > args.Len() {
			return true
		}
		for i, n := 0, args.Len(); i < n; i++ {
			if w.Has(args.At(i)) {
				return true
			}
		}
		return w.Has(t.Underlying())
	case *types.TypeParam:
		return true
	default:
		panic(t)
	}
	return false
}
