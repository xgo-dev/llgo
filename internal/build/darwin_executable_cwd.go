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

func configureDarwinExecutableCwd(ctx *context, pkg *ssa.Package, roots []*ssa.Function, isSymbolRoot func(string) bool) {
	init := pkg.Func(darwinExecutableCwdInitializer)
	getwd := pkg.Func("Getwd")
	executable := pkg.Func("executable")
	if init == nil || getwd == nil || executable == nil {
		return
	}
	globals := make(map[ssa.Value]bool)
	for _, name := range []string{"initCwd", "initCwdErr", "getwdCache"} {
		global, ok := pkg.Members[name].(*ssa.Global)
		if !ok || isSymbolRoot("os."+name) {
			return
		}
		globals[global] = true
	}
	for _, name := range []string{darwinExecutableCwdGuard, "os." + init.Name(), "os.Getwd", "os.Executable", "os.executable"} {
		if isSymbolRoot(name) {
			return
		}
	}

	// Ignore the startup lookup when checking for observers. Getwd itself
	// must be unobserved because startup may populate its directory cache.
	// Restore the body before emitting the shared archive's guarded lookup.
	blocks := init.Blocks
	init.Blocks = nil
	result := rta.Analyze(roots, false)
	init.Blocks = blocks
	if _, reachable := result.Reachable[getwd]; reachable {
		return
	}
	if _, reachable := result.Reachable[executable]; reachable {
		return
	}
	pkgInit := pkg.Func("init")
	for fn := range result.Reachable {
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				store, isStore := inst.(*ssa.Store)
				if fn == pkgInit && isStore && globals[store.Addr] {
					continue
				}
				for _, operand := range inst.Operands(nil) {
					if operand != nil && globals[*operand] {
						return
					}
				}
			}
		}
	}
	ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, darwinExecutableCwdGuard)
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: omit unobserved initial executable directory")
	}
}
