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

package build

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/pclnpost"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func canPruneEmbeddedPCLN(ctx *context) bool {
	conf := ctx.buildConf
	if conf.PCLNMode != PCLNEmbedded || conf.BuildMode != BuildModeExe || os.Getenv("LLGO_PCLNPOST") == "0" {
		return false
	}
	if canPruneWasmPCLN(ctx) {
		return true
	}
	return conf.Target == "" &&
		(conf.Goos == "linux" || conf.Goos == "darwin") &&
		(conf.Goarch == "amd64" || conf.Goarch == "arm64") &&
		shouldEmitRuntimeEntrySites(ctx)
}

// linkedPCLNData joins metadata to ownership-checked final sites. Both packing
// modes use exactly the same liveness and wrapper/alias rules.
func linkedPCLNData(original pclnmap.Data, analysis pclnpost.ExternalAnalysis) (pclnmap.Data, error) {
	if analysis.PointerSize != original.PointerSize {
		return pclnmap.Data{}, fmt.Errorf("pclntab pointer size changed from %d to %d", original.PointerSize, analysis.PointerSize)
	}
	data := original
	data.ImageBase, data.TextStart, data.TextEnd = analysis.ImageBase, analysis.TextStart, analysis.TextEnd
	data.Identity = analysis.Identity
	data.EntrySites = make([]pclnmap.Site, len(analysis.EntrySites))
	for i, site := range analysis.EntrySites {
		data.EntrySites[i] = pclnmap.Site{PCOffset: site.PCOffset, ID: site.ID}
	}
	if original.GOARCH == "wasm" {
		if len(original.Table.PCLines) != 0 || len(analysis.PCLineSites) != 0 {
			return pclnmap.Data{}, fmt.Errorf("unexpected native PC-line records in Wasm pclntab")
		}
		// A fully inlined program can legitimately have no named Go bodies.
		return pclnmap.Prune(data)
	}
	data.PCSites = make([]pclnmap.Site, len(analysis.PCLineSites))
	owners := make([]string, len(analysis.PCLineSites))
	for i, site := range analysis.PCLineSites {
		data.PCSites[i] = pclnmap.Site{PCOffset: site.PCOffset, ID: site.ID}
		owners[i] = site.OwnerSymbol
	}
	if err := filterExternalPCLNJoins(&data, owners); err != nil {
		return pclnmap.Data{}, err
	}
	return pclnmap.Prune(data)
}

// pruneEmbeddedPCLN relinks only the synthetic entry object. Metadata contains
// strings and integer indexes, not references to Go bodies, so its replacement
// cannot root otherwise dead functions. Relinking lets the native linker reclaim
// the removed bytes and relocate every reference, including LTO constants and
// Mach-O chained fixups. Runtime table formats and lookup costs stay unchanged.
func pruneEmbeddedPCLN(ctx *context, plan *mainLinkPlan, output string, verbose bool) error {
	analyze := pclnpost.AnalyzeExternal
	if plan.embeddedPCLN != nil && plan.entryTarget.GOARCH == "wasm" {
		analyze = func(path string) (pclnpost.ExternalAnalysis, error) { return analyzeWasmPCLN(path, *plan.embeddedPCLN) }
	}
	return pruneEmbeddedPCLNWith(plan, output, verbose, embeddedPCLNLinker{
		analyze: analyze,
		compile: func(ir, object string) error { return compileEmbeddedEntry(ctx, plan, ir, object) },
		link: func(candidate string, inputs []string) error {
			return linkObjFiles(ctx, candidate, inputs, plan.linkArgs, verbose)
		},
		publish: replaceFile,
	})
}

// The staged linker keeps publication separate from fallible compilation and
// verification. Failure tests can exercise this transaction without invoking a
// native linker or modifying a valid output.
type embeddedPCLNLinker struct {
	analyze func(string) (pclnpost.ExternalAnalysis, error)
	compile func(ir, object string) error
	link    func(candidate string, inputs []string) error
	publish func(candidate, output string) error
}

func pruneEmbeddedPCLNWith(plan *mainLinkPlan, output string, verbose bool, linker embeddedPCLNLinker) error {
	if plan.embeddedPCLN == nil {
		return nil
	}
	analysis, err := linker.analyze(output)
	if err != nil {
		return fmt.Errorf("analyze embedded pclntab: %w", err)
	}
	pruned, err := linkedPCLNData(*plan.embeddedPCLN, analysis)
	if err != nil {
		return fmt.Errorf("prune embedded pclntab: %w", err)
	}
	if len(pruned.Table.Records) == len(plan.embeddedPCLN.Table.Records) &&
		len(pruned.Table.PCLines) == len(plan.embeddedPCLN.Table.PCLines) {
		return nil
	}
	dir, err := os.MkdirTemp(filepath.Dir(output), ".llgo-pcln-link-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	ir := filepath.Join(dir, "entry.ll")
	if err := os.WriteFile(ir, []byte(plan.entryIR), 0o600); err != nil {
		return err
	}
	if err := rewriteEmbeddedPCLNIR(ir, pruned); err != nil {
		return err
	}
	object := filepath.Join(dir, "entry.o")
	if err := linker.compile(ir, object); err != nil {
		return fmt.Errorf("compile compact embedded pclntab: %w", err)
	}
	inputs := slices.Clone(plan.linkInputs)
	inputs[0] = object
	candidate := filepath.Join(dir, filepath.Base(output))
	if err := linker.link(candidate, inputs); err != nil {
		return fmt.Errorf("link compact embedded pclntab: %w", err)
	}
	final, err := linker.analyze(candidate)
	if err != nil {
		return fmt.Errorf("verify compact embedded pclntab: %w", err)
	}
	// Recompute from the original catalog: using only the pruned catalog here
	// would hide a newly-live function whose metadata we had removed.
	verified, err := linkedPCLNData(*plan.embeddedPCLN, final)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(pruned.Table, verified.Table) || !reflect.DeepEqual(pruned.SymbolIndex, verified.SymbolIndex) {
		return fmt.Errorf("embedded pclntab liveness changed during metadata relink")
	}
	if err := linker.publish(candidate, output); err != nil {
		return fmt.Errorf("publish compact embedded pclntab: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "llgo: embedded pclntab: %d -> %d functions, %d -> %d PC lines\n",
			len(plan.embeddedPCLN.Table.Records), len(pruned.Table.Records),
			len(plan.embeddedPCLN.Table.PCLines), len(pruned.Table.PCLines))
	}
	return nil
}

func compileEmbeddedEntry(ctx *context, plan *mainLinkPlan, ir, object string) error {
	if !useInMemoryNativeCodegen(ctx) {
		return ctx.irCompiler().Compile("-o", object, "-c", ir, "-Wno-override-module")
	}
	// Match initial entry emission exactly. In particular, running Clang's IR
	// optimizer only on the second pass would change more than metadata.
	prog := llssa.NewProgram(&plan.entryTarget)
	defer prog.Dispose()
	lc := llvm.NewContext()
	defer lc.Dispose()
	input, err := llvm.NewMemoryBufferFromFile(ir)
	if err != nil {
		return err
	}
	mod, err := lc.ParseIR(input)
	if err != nil {
		return err
	}
	defer mod.Dispose()
	var buffer llvm.MemoryBuffer
	switch ctx.buildConf.ltoMode() {
	case lto.Full:
		buffer = llvm.WriteFullLTOBitcodeToMemoryBuffer(mod, ctx.buildConf.Goos != "darwin")
	case lto.Thin:
		buffer = llvm.WriteThinLTOBitcodeToMemoryBuffer(mod)
	default:
		buffer, err = prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
		if err != nil {
			return err
		}
	}
	defer buffer.Dispose()
	return os.WriteFile(object, buffer.Bytes(), 0o600)
}

func rewriteEmbeddedPCLNIR(path string, data pclnmap.Data) error {
	lc := llvm.NewContext()
	defer lc.Dispose()
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		return err
	}
	mod, err := lc.ParseIR(buf)
	if err != nil {
		return err
	}
	defer mod.Dispose()
	setArray := func(name string, values func(llvm.Type) []llvm.Value) error {
		old := mod.NamedGlobal(name)
		if old.IsNil() {
			return fmt.Errorf("embedded pclntab array %s is missing", name)
		}
		elem := old.GlobalValueType().ElementType()
		items := values(elem)
		old.SetName(name + ".unpruned")
		array := llvm.AddGlobal(mod, llvm.ArrayType(elem, len(items)), name)
		array.SetInitializer(llvm.ConstArray(elem, items))
		array.SetLinkage(llvm.PrivateLinkage)
		array.SetGlobalConstant(true)
		array.SetUnnamedAddr(true)
		array.SetAlignment(old.Alignment())
		old.ReplaceAllUsesWith(array)
		old.EraseFromParentAsGlobal()
		return nil
	}
	ints := func(typ llvm.Type, fields ...uint64) llvm.Value {
		types := typ.StructElementTypes()
		values := make([]llvm.Value, len(fields))
		for i, field := range fields {
			values[i] = llvm.ConstInt(types[i], field, false)
		}
		return llvm.ConstNamedStruct(typ, values)
	}
	arrays := []struct {
		name   string
		values func(llvm.Type) []llvm.Value
	}{
		{funcInfoDataSymbol, func(t llvm.Type) (v []llvm.Value) {
			for _, r := range data.Table.Records {
				v = append(v, ints(t, uint64(r.SymbolPkg), uint64(r.SymbolName), uint64(r.NamePkg), uint64(r.NameName), uint64(r.FileRoot), uint64(r.FileName), uint64(r.Line)))
			}
			return
		}},
		{pcLineDataSymbol, func(t llvm.Type) (v []llvm.Value) {
			for _, r := range data.Table.PCLines {
				v = append(v, ints(t, r.ID, uint64(r.Func), uint64(r.FileRoot), uint64(r.FileName), uint64(r.Line)))
			}
			return
		}},
		{funcInfoStringsDataSymbol, func(t llvm.Type) (v []llvm.Value) {
			for _, b := range data.Table.Strings {
				v = append(v, llvm.ConstInt(t, uint64(b), false))
			}
			return
		}},
		{funcInfoStringOffsetsDataSymbol, func(t llvm.Type) (v []llvm.Value) {
			for _, b := range data.Table.StringOffsets {
				v = append(v, llvm.ConstInt(t, uint64(b), false))
			}
			return
		}},
		{funcInfoHashDataSymbol, func(t llvm.Type) (v []llvm.Value) {
			for _, b := range data.Table.Hash {
				v = append(v, llvm.ConstInt(t, uint64(b), false))
			}
			return
		}},
		{funcInfoSymbolIndexDataSymbol, func(t llvm.Type) (v []llvm.Value) {
			for _, r := range data.SymbolIndex {
				v = append(v, ints(t, r.SymbolID, uint64(r.FuncIndex)))
			}
			return
		}},
	}
	for _, array := range arrays {
		if array.name == pcLineDataSymbol && len(data.Table.PCLines) == 0 && mod.NamedGlobal(array.name).IsNil() {
			continue
		}
		if err := setArray(array.name, array.values); err != nil {
			return err
		}
	}
	for name, count := range map[string]int{
		funcInfoCountSymbol: len(data.Table.Records), pcLineCountSymbol: len(data.Table.PCLines),
		funcInfoStringCountSymbol: len(data.Table.StringOffsets), funcInfoSymbolIndexCountSymbol: len(data.SymbolIndex),
		funcInfoHashMaskSymbol: max(0, len(data.Table.Hash)-1),
	} {
		global := mod.NamedGlobal(name)
		if global.IsNil() {
			return fmt.Errorf("embedded pclntab count %s is missing", name)
		}
		global.SetInitializer(llvm.ConstInt(global.GlobalValueType(), uint64(count), false))
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(mod.String()), 0o600)
}
