package dcepass

import (
	"sort"
	"strings"

	"github.com/xgo-dev/llvm"
)

const (
	reflectMethodByNameCallAttr  = "llgo.reflect.methodbyname"
	reflectMethodByNameNamesAttr = "llgo.reflect.methodbyname.names"
)

// RefinedMethodNamesFromModules returns finite MethodByName name sets proved
// by an LLVM optimization pass for function-scoped semantic demand owners.
//
// The result is deliberately all-or-nothing per owner. If even one marked call
// in a function lacks a refinement, that owner is omitted and the Go planner
// retains its conservative dynamic-reflection behavior.
func RefinedMethodNamesFromModules(mods []llvm.Module, candidates []string) map[string][]string {
	wanted := make(map[string]struct{}, len(candidates))
	for _, name := range candidates {
		wanted[name] = struct{}{}
	}

	result := make(map[string][]string)
	for _, mod := range mods {
		for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
			if _, ok := wanted[fn.Name()]; !ok || fn.IsDeclaration() {
				continue
			}

			found := false
			known := true
			names := make(map[string]struct{})
			for _, block := range fn.BasicBlocks() {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
						continue
					}
					if inst.GetCallSiteStringAttribute(-1, reflectMethodByNameCallAttr).IsNil() {
						continue
					}
					found = true
					attr := inst.GetCallSiteStringAttribute(-1, reflectMethodByNameNamesAttr)
					if attr.IsNil() || attr.GetStringValue() == "" {
						known = false
						continue
					}
					for _, name := range strings.Split(attr.GetStringValue(), ",") {
						if name == "" {
							known = false
							break
						}
						names[name] = struct{}{}
					}
				}
			}
			if !found || !known || len(names) == 0 {
				continue
			}
			out := make([]string, 0, len(names))
			for name := range names {
				out = append(out, name)
			}
			sort.Strings(out)
			result[fn.Name()] = out
		}
	}
	return result
}
