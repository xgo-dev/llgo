//go:build !llgo

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

package cl_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl/cltest"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestRecoverUsesActivationToken(t *testing.T) {
	const src = `package foo

func recursive(depth int) any {
	if depth > 0 {
		return recursive(depth - 1)
	}
	return recover()
}
`
	ir := cltest.CompileIREx(t, src, "foo.go", false, nil)

	bind := regexp.MustCompile(`BindRecoverFrame"\(ptr @foo\.recursive, ptr (%[0-9]+)\)`).FindStringSubmatch(ir)
	if bind == nil {
		t.Fatalf("recover function does not bind an activation token:\n%s", ir)
	}
	if want := `Recover"(ptr ` + bind[1] + `)`; !strings.Contains(ir, want) {
		t.Fatalf("recover does not use bound activation token %s:\n%s", bind[1], ir)
	}
	if !strings.Contains(ir, `noinline`) || !strings.Contains(ir, `"disable-tail-calls"="true"`) {
		t.Fatalf("recover function must preserve its activation frame:\n%s", ir)
	}
}

func TestRecoverInterfaceMethodIR(t *testing.T) {
	const src = `package foo

type I interface { recoverValue() }
type T int
func (T) recoverValue() { recover() }
	func call(v I) { defer v.recoverValue() }
`
	ir := cltest.CompileIREx(t, src, "foo.go", false, nil)
	for _, want := range []string{
		`BindRecoverFrame"(ptr @foo.T.recoverValue`,
		`StartRecoverFrameAlias"(ptr @"foo.(*T).recoverValue", ptr @foo.T.recoverValue)`,
		`StartRecoverFrame"(ptr %`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing recover wrapper handoff %q:\n%s", want, ir)
		}
	}
}

func TestRecoverInterfaceMethodGlobalDCEIR(t *testing.T) {
	const src = `package foo

import "reflect"

func call(method func(reflect.Type, int) reflect.Method, typ reflect.Type) reflect.Method {
	return method(typ, 0)
}

func use(typ reflect.Type) {
	method := reflect.Type.Method
	_ = call(method, typ)
}
`
	ir := cltest.CompileIREx(t, src, "foo.go", false, func(prog llssa.Program) {
		prog.EnableGoGlobalDCE(true)
	})
	start := strings.Index(ir, `define %reflect.Method @"foo.Type.Method$thunk"`)
	if start < 0 {
		t.Fatalf("missing reflect.Type.Method thunk:\n%s", ir)
	}
	end := strings.Index(ir[start:], "\n}\n")
	if end < 0 {
		t.Fatalf("unterminated reflect.Type.Method thunk:\n%s", ir[start:])
	}
	thunk := ir[start : start+end]
	rawLoad := strings.Index(thunk, "load ptr, ptr")
	checkedLoad := strings.Index(thunk, `metadata !"go.method.Method:func(int) reflect.Method"`)
	alias := strings.Index(thunk, "StartRecoverFrameAlias")
	resultCheck := strings.Index(thunk, `metadata !"go.method.type.reflect"`)
	if rawLoad < 0 || checkedLoad < 0 || alias < 0 || resultCheck < 0 ||
		rawLoad > checkedLoad || checkedLoad > alias || alias > resultCheck {
		t.Fatalf("recover-aware method thunk lost raw token or GlobalDCE checks:\n%s", thunk)
	}
}
