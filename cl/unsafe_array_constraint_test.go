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

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestRewriteUnsafeArrayConstraintRepresentation(t *testing.T) {
	const src = `package foo
import "unsafe"
//llgo:cdecl
type Native[T any, A ~[2]T] struct { Items A }
type Arrays[T any] interface { ~[2]T }
type Embedded[T any] interface { Arrays[T] }
type Union[T any] interface { ~[2]T | ~[3]T }
type Mixed[T any] interface { ~[2]T | ~[3]T | ~*[2]T | ~*[3]T | ~[]T }
//llgo:cdecl
type MixedNative[T any, A Mixed[T]] struct { Items A }
func mixedSize[T any, A Mixed[T]](p *MixedNative[T,A], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func mixedAlign[T any, A Mixed[T]](p *MixedNative[T,A], i int) uintptr { return unsafe.Alignof(p.Items[i]) }
func mixedUnevaluated[T any, A Mixed[T]](p *MixedNative[T,A]) uintptr { return unsafe.Sizeof(p.Items[panicIndex()]) }
type ArrayPointers[T any] interface { ~*[2]T | ~*[3]T }
//llgo:cdecl
type PointerNative[T any, A ArrayPointers[T]] struct { Items A }
func pointerSize[T any, A ArrayPointers[T]](p *PointerNative[T,A], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func pointerAlign[T any, A ArrayPointers[T]](p PointerNative[T,A], i int) uintptr { return unsafe.Alignof(p.Items[i]) }
func panicIndex() int { panic("index evaluated") }
func pointerUnevaluated[T any, A ArrayPointers[T]](p *PointerNative[T,A]) uintptr { return unsafe.Sizeof(p.Items[panicIndex()]) }
//llgo:cdecl
type UnionNative[T any, A Union[T]] struct { Items A }
func named[T any, A Arrays[T]](p *Native[T,A], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func embedded[T any, A Embedded[T]](p *Native[T,A], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func union[T any, A Union[T]](p *UnionNative[T,A], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func unionAlign[T any, A Union[T]](p *UnionNative[T,A], i int) uintptr { return unsafe.Alignof(p.Items[i]) }
func element[T any, A ~[2]T](p *Native[T,A], i int) uintptr { return unsafe.Sizeof(p.Items[i]) }
func alignment[T any, A ~[2]T](p Native[T,A], i int) uintptr { return unsafe.Alignof(p.Items[i]) }
func Result() uintptr { var p Native[func(),[2]func()]; var u UnionNative[func(),[2]func()]; var q PointerNative[func(),*[2]func()]; var m2 MixedNative[func(),[2]func()]; var m3 MixedNative[func(),[3]func()]; var mp MixedNative[func(),*[3]func()]; var ms MixedNative[func(),[]func()]; return mixedSize(&m2,1)+mixedSize(&m3,2)+mixedSize(&mp,2)+mixedSize(&ms,1)+mixedAlign(&m2,1)+mixedAlign(&m3,2)+mixedAlign(&mp,2)+mixedAlign(&ms,1)+mixedUnevaluated(&mp)+pointerSize(&q,1)+pointerAlign(q,1)+pointerUnevaluated(&q)+ element(&p, 1)+alignment(p,0)+named(&p,1)+embedded(&p,1)+union(&u,1)+unionAlign(&u,1) }
`
	for _, target := range []llssa.Target{{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			var results [2]map[string]uint64
			for i, raw := range []bool{true, false} {
				_, mod := compileRewrittenUnsafeForTarget(t, src, &target, raw)
				opts := llvm.NewPassBuilderOptions()
				t.Cleanup(opts.Dispose)
				if err := mod.RunPasses("default<O2>", llvm.TargetMachine{}, opts); err != nil {
					t.Fatal(err)
				}
				results[i] = make(map[string]uint64)
				if !raw {
					for _, name := range []string{"foo.pointerUnevaluated[func(),*[2]func()]", "foo.mixedUnevaluated[func(),*[3]func()]"} {
						fn := mod.NamedFunction(name)
						if fn.IsNil() {
							t.Fatalf("missing %s", name)
						}
						if strings.Contains(fn.String(), "panicIndex") || strings.Contains(fn.String(), "PanicIndex") || strings.Contains(fn.String(), "AssertNilDeref") {
							t.Fatalf("operand evaluated: %s", fn.String())
						}
					}
				}
				for _, name := range []string{"element[func(),[2]func()]", "alignment[func(),[2]func()]", "named[func(),[2]func()]", "embedded[func(),[2]func()]", "union[func(),[2]func()]", "unionAlign[func(),[2]func()]", "pointerSize[func(),*[2]func()]", "pointerAlign[func(),*[2]func()]", "mixedSize[func(),[2]func()]", "mixedSize[func(),[3]func()]", "mixedSize[func(),*[3]func()]", "mixedSize[func(),[]func()]", "mixedAlign[func(),[2]func()]", "mixedAlign[func(),[3]func()]", "mixedAlign[func(),*[3]func()]", "mixedAlign[func(),[]func()]"} {
					fn := mod.NamedFunction("foo." + name)
					if fn.IsNil() {
						t.Fatalf("missing %s: %s", name, mod.String())
					}
					found := false
					for _, block := range fn.BasicBlocks() {
						ret := block.LastInstruction()
						if ret.IsAReturnInst().IsNil() {
							continue
						}
						v := ret.Operand(0).IsAConstantInt()
						if v.IsNil() {
							t.Fatalf("%s: %s", name, fn.String())
						}
						results[i][name] = v.ZExtValue()
						found = true
					}
					if !found {
						t.Fatalf("missing return: %s", name)
					}
				}
			}
			for name, want := range results[0] {
				if got := results[1][name]; got != want {
					t.Errorf("%s: rewritten %d, raw %d", name, got, want)
				}
			}
		})
	}
}

func TestRewriteUnsafeRecursiveAndGenericAlias(t *testing.T) {
	const src = `package foo
import "unsafe"
type Link struct { Next *Link }
type Alias[T any] = struct { Value T; Tail byte }
func trap[T any](v Alias[T]) Alias[T] { panic("operand evaluated") }
func size[T any](v Alias[T]) uintptr { return unsafe.Sizeof(trap(v)) }
func align[T any](v Alias[T]) uintptr { return unsafe.Alignof(trap(v)) }
func offset[T any](v Alias[T]) uintptr { return unsafe.Offsetof(trap(v).Tail) }
func recursive[T any](v struct { Link Link; Value T }) uintptr { return unsafe.Sizeof(v) }
func Result() uintptr { var v Alias[uint64]; return size(v)+align(v)+offset(v)+recursive(struct { Link Link; Value uint64 }{}) }
`
	_, module := compileRewrittenUnsafe(t, src)
	for _, name := range []string{"size[uint64]", "align[uint64]", "offset[uint64]", "recursive[uint64]"} {
		fn := module.NamedFunction("foo." + name)
		if fn.IsNil() {
			t.Fatalf("missing %s", name)
		}
		if strings.Contains(fn.String(), "foo.trap") {
			t.Errorf("%s evaluates operand: %s", name, fn.String())
		}
	}
}
