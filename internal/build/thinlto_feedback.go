package build

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/goplus/llgo/internal/dcepass"
	"github.com/goplus/llgo/internal/deadcode"
	"github.com/goplus/llgo/internal/meta"
	gllvm "github.com/xgo-dev/llvm"
)

// runThinLTOFeedback performs the opt-in two-link prototype. The first link is
// intentionally real ThinLTO: its backend modules are inspected after
// optimization, then rewritten package bitcode is overlaid ahead of the old
// archives for the final link.
func runThinLTOFeedback(ctx *context, feedbackOutput, outputPath string, linkInputs, linkArgs []string, pkgs []Package, needRuntime bool, firstPlan deadcode.Plan, verbose bool) error {
	if err := linkObjFiles(ctx, feedbackOutput, linkInputs, linkArgs, verbose); err != nil {
		return fmt.Errorf("thin LTO feedback first link: %w", err)
	}
	defer os.Remove(feedbackOutput)
	defer cleanupThinLTOFeedbackOutputTemps(feedbackOutput)

	modulePaths, err := thinLTOFeedbackModulePaths(linkInputs)
	if err != nil {
		return err
	}
	defer cleanupThinLTOFeedbackTemps(linkInputs)
	mods, dispose, err := parseThinLTOFeedbackModules(modulePaths)
	if err != nil {
		return err
	}
	deadFunctions := dcepass.DeadNoInlineFunctionsFromModules(
		mods,
		dceEntryRootCandidates(pkgs, needRuntime),
		thinLTOFeedbackCandidates(pkgs),
	)
	dispose()
	if len(deadFunctions) == 0 {
		return publishThinLTOFeedbackOutput(feedbackOutput, outputPath)
	}

	summaryPlan, err := buildDeadcodePlanWithFeedback(pkgs, needRuntime, deadFunctions)
	if err != nil {
		return fmt.Errorf("thin LTO feedback second plan: %w", err)
	}
	if reflect.DeepEqual(firstPlan.LiveSlots, summaryPlan.LiveSlots) {
		return publishThinLTOFeedbackOutput(feedbackOutput, outputPath)
	}
	overlays, err := materializeThinLTOFeedbackOverlays(ctx, pkgs, summaryPlan, verbose)
	if err != nil {
		return err
	}
	if len(overlays) == 0 {
		return publishThinLTOFeedbackOutput(feedbackOutput, outputPath)
	}
	defer removeThinLTOFeedbackFiles(overlays)
	secondInputs := insertThinLTOOverlays(linkInputs, overlays)
	defer cleanupThinLTOFeedbackTemps(secondInputs)
	defer cleanupThinLTOFeedbackOutputTemps(outputPath)
	if err := linkObjFiles(ctx, outputPath, secondInputs, linkArgs, verbose); err != nil {
		return fmt.Errorf("thin LTO feedback second link: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "llgo: ThinLTO feedback removed %d function facts and relinked %d overlays\n", len(deadFunctions), len(overlays))
	}
	return nil
}

func buildDeadcodePlanWithFeedback(pkgs []Package, needRuntime bool, deadFunctions map[string]struct{}) (deadcode.Plan, error) {
	metas := linkedPackageMetas(pkgs)
	summary, err := meta.NewGlobalSummary(metas)
	if err != nil {
		return deadcode.Plan{}, err
	}
	return deadcode.BuildPlanWithFeedback(summary, dceEntryRootCandidates(pkgs, needRuntime), deadcode.Feedback{DeadFunctions: deadFunctions}), nil
}

func thinLTOFeedbackCandidates(pkgs []Package) []string {
	seen := make(map[string]struct{})
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Meta == nil {
			continue
		}
		for _, name := range pkg.Meta.DemandFunctionNames() {
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func thinLTOFeedbackModulePaths(inputs []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, input := range inputs {
		pattern := input + ".4.opt.bc"
		if strings.HasSuffix(input, ".a") {
			pattern = input + "(*).4.opt.bc"
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob ThinLTO feedback modules %q: %w", pattern, err)
		}
		for _, match := range matches {
			seen[match] = struct{}{}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("ThinLTO feedback produced no .4.opt.bc modules")
	}
	return paths, nil
}

func parseThinLTOFeedbackModules(paths []string) ([]gllvm.Module, func(), error) {
	ctx := gllvm.NewContext()
	mods := make([]gllvm.Module, 0, len(paths))
	dispose := func() {
		for _, mod := range mods {
			mod.Dispose()
		}
		ctx.Dispose()
	}
	for _, path := range paths {
		mod, err := ctx.ParseBitcodeFile(path)
		if err != nil {
			dispose()
			return nil, func() {}, fmt.Errorf("parse ThinLTO feedback module %s: %w", path, err)
		}
		mods = append(mods, mod)
	}
	return mods, dispose, nil
}

func cleanupThinLTOFeedbackTemps(inputs []string) {
	for _, input := range inputs {
		pattern := input + ".*.bc"
		if strings.HasSuffix(input, ".a") {
			pattern = input + "(*).*bc"
		}
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			_ = os.Remove(match)
		}
	}
}

func cleanupThinLTOFeedbackOutputTemps(output string) {
	patterns := []string{
		output + ".*.bc",
		output + ".index.dot",
		output + ".resolution.txt",
		output + ".lto.*.o",
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			_ = os.Remove(match)
		}
	}
}

func removeThinLTOFeedbackFiles(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func materializeThinLTOFeedbackOverlays(ctx *context, pkgs []Package, plan deadcode.Plan, verbose bool) ([]string, error) {
	var overlays []string
	succeeded := false
	defer func() {
		if !succeeded {
			removeThinLTOFeedbackFiles(overlays)
		}
	}()
	for _, aPkg := range pkgs {
		if aPkg == nil || aPkg.LPkg == nil || aPkg.Package == nil || aPkg.Package.ExportFile == "" {
			continue
		}
		dcepass.RewriteTypeMethodTables(aPkg.LPkg.Module(), plan.LiveSlots, verbose)
		exportFile, exportBuffer, err := exportPackageObject(ctx, aPkg.PkgPath, aPkg.Package.ExportFile, aPkg.LPkg)
		if err != nil {
			return nil, fmt.Errorf("export ThinLTO feedback overlay for %s: %w", aPkg.PkgPath, err)
		}
		if exportFile != "" {
			overlays = append(overlays, exportFile)
			continue
		}
		file, err := os.CreateTemp("", "llgo-thinlto-overlay-*.o")
		if err != nil {
			exportBuffer.buffer.Dispose()
			return nil, err
		}
		name := file.Name()
		_, writeErr := file.Write(exportBuffer.buffer.Bytes())
		closeErr := file.Close()
		exportBuffer.buffer.Dispose()
		if writeErr != nil {
			_ = os.Remove(name)
			return nil, writeErr
		}
		if closeErr != nil {
			_ = os.Remove(name)
			return nil, closeErr
		}
		overlays = append(overlays, name)
	}
	succeeded = true
	return overlays, nil
}

func insertThinLTOOverlays(inputs, overlays []string) []string {
	out := make([]string, 0, len(inputs)+len(overlays))
	inserted := false
	for _, input := range inputs {
		if !inserted && strings.HasSuffix(input, ".a") {
			out = append(out, overlays...)
			inserted = true
		}
		out = append(out, input)
	}
	if !inserted {
		out = append(out, overlays...)
	}
	return out
}

func publishThinLTOFeedbackOutput(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("publish ThinLTO feedback output: %w", err)
	}
	return nil
}
