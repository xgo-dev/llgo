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

package llvmattr

import (
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llvm"
)

const runtimePrefix = env.LLGoRuntimePkg + "/internal/runtime."

// runtimeFunctionModel is the single source of audited runtime contracts.
// Indices describe the logical LLVM signature before C ABI lowering: result
// is index 0, parameters start at 1. Never put pointer properties of a string
// or slice's data field on the aggregate itself (or on its later byval/sret
// storage). internal/cabi transports attributes when it rewrites this signature.
// Argument-dependent guarantees belong on individual calls, not in this table.
type runtimeFunctionModel struct {
	function []runtimeAttribute
	result   []runtimeAttribute
	params   map[int][]runtimeAttribute
	// A nonzero entry is the one-based size argument of an allocator that
	// returns fresh storage for positive byte counts, but zerobase for zero.
	allocationSizeParam int
	// Compiler-maintained root chains add writes and publish argument pointers.
	// These contracts are only used when that instrumentation is disabled.
	uninstrumented bool
}

type runtimeAttribute struct {
	name  string
	value uint64
}

var runtimeFunctionModels = func() map[string]runtimeFunctionModel {
	attr := func(names ...string) []runtimeAttribute {
		a := make([]runtimeAttribute, len(names))
		for i, name := range names {
			a[i].name = name
		}
		return a
	}
	models := make(map[string]runtimeFunctionModel)
	add := func(names []string, model runtimeFunctionModel) {
		for _, name := range names {
			models[name] = model
		}
	}
	// All allocator implementations return zerobase for size 0. Consequently
	// neither noalias nor an allocation-family/allockind contract is unconditional.
	add([]string{"AllocU", "AllocZ", "AllocRoot"}, runtimeFunctionModel{
		result:              attr("nonnull", "noundef"),
		allocationSizeParam: 1,
		function:            []runtimeAttribute{{"allocsize", AllocSize(0, AllocSizeNoCount)}},
	})
	add([]string{"New", "newobject", "newarray", "makemap"}, runtimeFunctionModel{result: attr("nonnull", "noundef")})
	// These helpers always allocate a nonzero object, even for empty strings,
	// maps, and unbuffered channels. makemap itself may reuse a supplied header
	// and must not inherit the stronger contract of MakeMap's nil-header call.
	add([]string{"CStrDup", "NewStringIter", "NewMapIter", "MakeSmallMap", "MakeMap", "makemap_small", "NewChan", "NewChan64"},
		runtimeFunctionModel{result: attr("noalias", "nonnull", "noundef")})
	// A nil input must still enter the recoverable panic path. Only the returned
	// pointer is nonnull; putting nonnull on the parameter would erase the check.
	models["AssertNilDerefPtr"] = runtimeFunctionModel{
		result: attr("nonnull", "noundef"), params: map[int][]runtimeAttribute{1: attr("returned")},
	}
	// Assert*/PanicWrapNilPointer have returning paths. throw is currently a
	// printing stub. Do not infer noreturn from a name prefix.
	add([]string{"Panic", "Goexit", "PanicErrorString", "PanicIndex", "PanicIndexU",
		"PanicExtendIndex", "PanicExtendIndexU", "PanicSliceConvert", "PanicTypeAssert",
		"PanicTypeAssertionError", "PanicSIMDImmediate", "PanicSIMDUnimplemented",
		"PanicSignal", "panicBounds", "panicMakeChanSize", "panicSendOnClosedChan",
		"panicmakeslicelen", "panicmakeslicecap", "panicgrowslicelen"},
		runtimeFunctionModel{function: attr("noreturn")})
	// These operations neither allocate nor invoke user code. In contrast,
	// ChanLen locks a mutex, interface equality can panic/call user code, and
	// floating-point hashing calls the random-number generator for NaNs.
	// memory(read) covers all locations, including hashkey and nested string data.
	read := runtimeFunctionModel{
		function:       append(attr("nofree", "nosync", "nounwind", "willreturn"), runtimeAttribute{"memory", MemoryRead}),
		uninstrumented: true,
	}
	add([]string{"StringEqual", "StringLess", "Complex128Div"}, read)
	read.params = map[int][]runtimeAttribute{1: attr("readonly")}
	add([]string{"ChanCap", "MapLen", "memhash", "Memhash", "memhash32", "Memhash32", "memhash64", "Memhash64",
		"memhash8", "memhash16", "memhash128", "strhash"}, read)
	read.params = map[int][]runtimeAttribute{1: attr("readonly"), 2: attr("readonly")}
	add([]string{"memequal", "memequalptr", "memequal8", "memequal16", "memequal32", "memequal64", "memequal128",
		"f32equal", "f64equal", "c64equal", "c128equal", "strequal"}, read)
	add([]string{"memequal0", "memhash0"}, runtimeFunctionModel{
		function:       append(attr("nofree", "nosync", "nounwind", "willreturn"), runtimeAttribute{"memory", MemoryNone}),
		uninstrumented: true,
	})
	// The destination escapes through the return value, so no captures(none).
	// Its pointee is written, while the String parameter is an aggregate value.
	models["CStrCopy"] = runtimeFunctionModel{params: map[int][]runtimeAttribute{1: attr("returned", "writeonly")}, uninstrumented: true}
	models["StringFrom"] = runtimeFunctionModel{params: map[int][]runtimeAttribute{1: attr("readonly")}, uninstrumented: true}
	return models
}()

// ApplyRuntimeFunction attaches unconditional contracts to a logical runtime entry.
// Entries with compiler-owned environment parameters must not use this mapping.
func ApplyRuntimeFunction(ctx llvm.Context, fn llvm.Value, name string, gcRoots bool) {
	name, ok := strings.CutPrefix(name, runtimePrefix)
	if !ok {
		return
	}
	model, ok := runtimeFunctionModels[name]
	if !ok || model.uninstrumented && gcRoots {
		return
	}
	apply := func(index int, attrs []runtimeAttribute) {
		for _, attr := range attrs {
			fn.AddAttributeAtIndex(index, ctx.CreateEnumAttribute(llvm.AttributeKindID(attr.name), attr.value))
		}
	}
	apply(-1, model.function)
	apply(0, model.result)
	for i, attrs := range model.params {
		apply(i, attrs)
	}
}

// ApplyRuntimeCall adds only argument-dependent facts. The shared
// declaration must stay valid for zero-sized and dynamically sized calls too.
func ApplyRuntimeCall(ctx llvm.Context, call, callee llvm.Value, args []llvm.Value) {
	if callee.IsAFunction().IsNil() {
		return
	}
	name, ok := strings.CutPrefix(callee.Name(), runtimePrefix)
	if !ok {
		return
	}
	model, ok := runtimeFunctionModels[name]
	index := model.allocationSizeParam - 1
	if !ok || index < 0 || index >= len(args) {
		return
	}
	size := args[index].IsAConstantInt()
	if size.IsNil() || size.ZExtValue() == 0 {
		return
	}
	// Restrict to ordinary positive sizes representable as a Go int. This also
	// avoids making extent claims for wrapped/oversized allocation requests.
	maxSize := (uint64(1) << (size.Type().IntTypeWidth() - 1)) - 1
	if size.ZExtValue() > maxSize {
		return
	}
	call.AddCallSiteAttribute(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("noalias"), 0))
	call.AddCallSiteAttribute(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("dereferenceable"), size.ZExtValue()))
}
