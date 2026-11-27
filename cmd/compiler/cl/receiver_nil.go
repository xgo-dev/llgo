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
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
)

func (p *context) checkMethodCallReceiver(b llvmgen.Builder, ins ssa.Instruction, call *ssa.CallCommon) {
	if len(call.Args) == 0 || p.optPlan == nil {
		return
	}
	if nc, ok := p.optPlan.NilChecks[ins]; ok {
		p.emitPlannedReceiverNilCheck(b, call.Args[0], nc)
	}
}

func isPointerMethodWrapperCall(owner *ssa.Function, call *ssa.CallCommon) bool {
	return ssaopt.IsPointerMethodWrapperCall(owner, call)
}

func isMethodReceiverWrapper(fn *ssa.Function) bool {
	return ssaopt.IsMethodReceiverWrapper(fn)
}

func (p *context) checkBoundMethodReceiver(b llvmgen.Builder, closure *ssa.MakeClosure) {
	if len(closure.Bindings) == 0 || p.optPlan == nil {
		return
	}
	if nc, ok := p.optPlan.NilChecks[closure]; ok {
		p.emitPlannedReceiverNilCheck(b, closure.Bindings[0], nc)
	}
}

func (p *context) emitPlannedReceiverNilCheck(b llvmgen.Builder, receiver ssa.Value, nc ssaopt.NilCheckPlan) {
	if nc.Elided || (!nc.NeedBaseCheck && !nc.NeedAddrCheck) {
		return
	}
	p.recordPanicSite(b, nc.Pos)
	if nc.NeedBaseCheck {
		p.emitNilDerefBaseCheck(b, receiver)
	}
	if nc.NeedAddrCheck {
		b.AssertNilDeref(p.compileValue(b, receiver))
	}
}

func methodReceiverHasUncheckedBase(receiver ssa.Value) bool {
	return ssaopt.MethodReceiverHasUncheckedBase(receiver)
}
