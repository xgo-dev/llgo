/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package ssa

import "github.com/xgo-dev/llvm"

const (
	FuncInfoMetadataName = "llgo.funcinfo"
	PCLineMetadataName   = "llgo.pcline"
	funcInfoVersion      = 1
	pcLineVersion        = 1
)

// EnableFuncInfoMetadata controls emission of DCE-safe function source
// metadata. The metadata intentionally stores symbol names as strings instead
// of function pointer operands, so it can be consumed before materializing a
// final runtime line/func table without keeping otherwise-dead functions alive.
func (p Program) EnableFuncInfoMetadata(enable bool) {
	p.enableFuncInfoMetadata = enable
}

func (p Program) FuncInfoMetadataEnabled() bool {
	return p.enableFuncInfoMetadata
}

// EnableFuncInfoSites controls emission of the per-function site records
// (entry/stub/pc-line inline-asm fragments inside function bodies). They are
// gated separately from the funcinfo metadata tables because the
// body-embedded anchors shift instruction/scope layout enough to confuse
// debuggers; debug builds keep the tables (FuncForPC name/FileLine fidelity
// via the dlsym path) but drop the sites.
func (p Program) EnableFuncInfoSites(enable bool) {
	p.enableFuncInfoSites = enable
}

func (p Program) FuncInfoSitesEnabled() bool {
	return p.enableFuncInfoSites
}

// EmitFuncInfo records a function's linker symbol, Go name, and declaration
// source position as LLVM named metadata. The row layout is:
//
//	!{i32 version, !"symbol", !"go.name", !"file", i32 line, i32 column}
func (p Package) EmitFuncInfo(symbol, name, file string, line, column int) {
	if symbol == "" {
		return
	}
	if line < 0 {
		line = 0
	}
	if column < 0 {
		column = 0
	}
	i32 := p.Prog.Int32().ll
	p.mod.AddNamedMetadataOperand(FuncInfoMetadataName,
		p.Prog.ctx.MDNode([]llvm.Metadata{
			llvm.ConstInt(i32, funcInfoVersion, false).ConstantAsMetadata(),
			p.Prog.ctx.MDString(symbol),
			p.Prog.ctx.MDString(name),
			p.Prog.ctx.MDString(file),
			llvm.ConstInt(i32, uint64(line), false).ConstantAsMetadata(),
			llvm.ConstInt(i32, uint64(column), false).ConstantAsMetadata(),
		}),
	)
}

// EmitPCLineInfo records a PC label id and its source position. The id names a
// zero-byte label emitted in the function body; keeping the metadata string-only
// lets dead functions be removed without the line table holding address
// references to them.
func (p Package) EmitPCLineInfo(id uint64, symbol, file string, line, column int) {
	if id == 0 || symbol == "" {
		return
	}
	if line < 0 {
		line = 0
	}
	if column < 0 {
		column = 0
	}
	i32 := p.Prog.Int32().ll
	i64 := p.Prog.Int64().ll
	p.mod.AddNamedMetadataOperand(PCLineMetadataName,
		p.Prog.ctx.MDNode([]llvm.Metadata{
			llvm.ConstInt(i32, pcLineVersion, false).ConstantAsMetadata(),
			llvm.ConstInt(i64, id, false).ConstantAsMetadata(),
			p.Prog.ctx.MDString(symbol),
			p.Prog.ctx.MDString(file),
			llvm.ConstInt(i32, uint64(line), false).ConstantAsMetadata(),
			llvm.ConstInt(i32, uint64(column), false).ConstantAsMetadata(),
		}),
	)
}

// NeedsFramePointer reports whether functions should keep the frame-pointer
// chain: only on OS targets where the runtime's FP unwinder runs.
func (p Program) NeedsFramePointer() bool {
	target := p.Target()
	if target == nil {
		return true
	}
	if target.Target != "" || target.GOARCH == "wasm" {
		return false
	}
	switch target.GOOS {
	case "linux", "darwin", "":
		return true
	}
	return false
}
