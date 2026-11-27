// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

package ssa

import (
	"go/ast"
	"strings"

	"github.com/xgo-dev/llgo/types"
)

// Adapted from golang.org/x/tools/internal/versions plus Go 1.20 slice/AST
// helpers so github.com/xgo-dev/llgo/ssa remains strictly Go 1.20+ compatible.

const (
	verGo1_22 = "go1.22"
	verFuture = ""
)

func verLang(x string) string {
	v := goverLang(verStripGo(x))
	if v == "" {
		return ""
	}
	return x[:2+len(v)]
}

func verCompare(x, y string) int { return goverCompare(verStripGo(x), verStripGo(y)) }
func verIsValid(x string) bool   { return goverIsValid(verStripGo(x)) }

func verAtLeast(v, release string) bool {
	if v == verFuture {
		return true
	}
	return verCompare(verLang(v), verLang(release)) >= 0
}

func verFileVersion(info *types.Info, file *ast.File) string {
	if info != nil {
		if v := info.FileVersions[file]; verIsValid(v) {
			return v
		}
	}
	return verFuture
}

func verStripGo(v string) string {
	v, _, _ = strings.Cut(v, "-")
	if len(v) < 2 || v[:2] != "go" {
		return ""
	}
	return v[2:]
}

type gover struct {
	major string
	minor string
	patch string
	kind  string
	pre   string
}

func goverCompare(x, y string) int {
	vx := goverParse(x)
	vy := goverParse(y)
	if c := goverCmpInt(vx.major, vy.major); c != 0 {
		return c
	}
	if c := goverCmpInt(vx.minor, vy.minor); c != 0 {
		return c
	}
	if c := goverCmpInt(vx.patch, vy.patch); c != 0 {
		return c
	}
	if c := strings.Compare(vx.kind, vy.kind); c != 0 {
		return c
	}
	if c := goverCmpInt(vx.pre, vy.pre); c != 0 {
		return c
	}
	return 0
}

func goverLang(x string) string {
	v := goverParse(x)
	if v.minor == "" || v.major == "1" && v.minor == "0" {
		return v.major
	}
	return v.major + "." + v.minor
}

func goverIsValid(x string) bool { return goverParse(x) != (gover{}) }

func goverParse(x string) gover {
	var v gover
	var ok bool
	v.major, x, ok = goverCutInt(x)
	if !ok {
		return gover{}
	}
	if x == "" {
		v.minor = "0"
		v.patch = "0"
		return v
	}
	if x[0] != '.' {
		return gover{}
	}
	v.minor, x, ok = goverCutInt(x[1:])
	if !ok {
		return gover{}
	}
	if x == "" {
		if goverCmpInt(v.minor, "21") < 0 {
			v.patch = "0"
		}
		return v
	}
	if x[0] == '.' {
		v.patch, x, ok = goverCutInt(x[1:])
		if !ok || x != "" {
			return gover{}
		}
		return v
	}
	i := 0
	for i < len(x) && (x[i] < '0' || '9' < x[i]) {
		if x[i] < 'a' || 'z' < x[i] {
			return gover{}
		}
		i++
	}
	if i == 0 {
		return gover{}
	}
	v.kind, x = x[:i], x[i:]
	if x == "" {
		return v
	}
	v.pre, x, ok = goverCutInt(x)
	if !ok || x != "" {
		return gover{}
	}
	return v
}

func goverCutInt(x string) (n, rest string, ok bool) {
	i := 0
	for i < len(x) && '0' <= x[i] && x[i] <= '9' {
		i++
	}
	if i == 0 || x[0] == '0' && i != 1 {
		return "", "", false
	}
	return x[:i], x[i:], true
}

func goverCmpInt(x, y string) int {
	if x == y {
		return 0
	}
	if len(x) < len(y) {
		return -1
	}
	if len(x) > len(y) {
		return +1
	}
	if x < y {
		return -1
	}
	return +1
}

// --- Go 1.20 stdlib compatibility helpers ---

func astUnparen(e ast.Expr) ast.Expr {
	for {
		paren, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = paren.X
	}
}

func sliceClone[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	return append(S([]E{}), s...)
}

func sliceConcat[S ~[]E, E any](slices ...S) S {
	size := 0
	for _, s := range slices {
		size += len(s)
	}
	if size == 0 {
		return nil
	}
	out := make(S, 0, size)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func sliceContains[S ~[]E, E comparable](s S, v E) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

func sliceContainsFunc[S ~[]E, E any](s S, f func(E) bool) bool {
	for _, e := range s {
		if f(e) {
			return true
		}
	}
	return false
}

func sliceDeleteFunc[S ~[]E, E any](s S, del func(E) bool) S {
	i := 0
	for _, e := range s {
		if !del(e) {
			s[i] = e
			i++
		}
	}
	var zero E
	for j := i; j < len(s); j++ {
		s[j] = zero
	}
	return s[:i]
}
