package build

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

const (
	darwinExecutableCwdGuard       = "os.llgoExecutableCwdNeeded"
	darwinExecutableCwdInitializer = "llgoInitExecutableCwd"
)

func rewriteDarwinExecutableCwd(source []byte) []byte {
	original := []byte("\nvar initCwd, initCwdErr = Getwd()\n")
	if bytes.Count(source, original) != 1 {
		return nil
	}
	updated := bytes.Replace(source, original, []byte("\nvar initCwd, initCwdErr = llgoInitExecutableCwd()\n"), 1)
	return append(updated, []byte(`
var llgoExecutableCwdNeeded bool

func llgoInitExecutableCwd() (string, error) {
	if llgoExecutableCwdNeeded {
		return Getwd()
	}
	return "", nil
}
`)...)
}

func configureDarwinExecutableCwd(ctx *context, pkg *ssa.Package, result *rta.Result, isSymbolRoot func(string) bool) bool {
	init := pkg.Func(darwinExecutableCwdInitializer)
	getwd := pkg.Func("Getwd")
	executable := pkg.Func("executable")
	if init == nil || getwd == nil || executable == nil {
		return false
	}
	globals := make(map[ssa.Value]bool)
	for _, name := range []string{"initCwd", "initCwdErr", "getwdCache"} {
		global, ok := pkg.Members[name].(*ssa.Global)
		if !ok || isSymbolRoot("os."+name) {
			return false
		}
		globals[global] = true
	}
	for _, name := range []string{darwinExecutableCwdGuard, "os." + init.Name(), "os.Getwd", "os.Executable", "os.executable"} {
		if isSymbolRoot(name) {
			return false
		}
	}

	if _, reachable := result.Reachable[getwd]; reachable {
		return false
	}
	if _, reachable := result.Reachable[executable]; reachable {
		return false
	}
	pkgInit := pkg.Func("init")
	var operands [8]*ssa.Value
	for fn := range result.Reachable {
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				store, isStore := inst.(*ssa.Store)
				if fn == pkgInit && isStore && globals[store.Addr] {
					continue
				}
				for _, operand := range inst.Operands(operands[:0]) {
					if operand != nil && globals[*operand] {
						return false
					}
				}
			}
		}
	}
	ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, darwinExecutableCwdGuard)
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: omit unobserved initial executable directory")
	}
	return true
}
