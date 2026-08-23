package build

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/internal/dcepass"
	"github.com/xgo-dev/llgo/internal/deadcode"
	"github.com/xgo-dev/llgo/internal/meta"
	gllvm "github.com/xgo-dev/llvm"
)

const thinLTOFeedbackMaxRounds = 3

// runThinLTOFeedback performs the opt-in bounded feedback loop. Each link is
// intentionally real ThinLTO: its backend modules are inspected after
// optimization, then rewritten package bitcode is overlaid ahead of the old
// archives for the next link.
func runThinLTOFeedback(ctx *context, feedbackOutput, outputPath string, linkInputs, linkArgs []string, pkgs []Package, needRuntime bool, firstPlan deadcode.Plan, knownDefinitions map[string]struct{}, verbose bool) error {
	defer os.Remove(feedbackOutput)
	defer cleanupThinLTOFeedbackOutputTemps(feedbackOutput)
	baseInputs := append([]string(nil), linkInputs...)
	currentInputs := baseInputs
	currentPlan := firstPlan
	knownDead := make(map[string]struct{})
	knownRefinedNames := make(map[string][]string)
	candidates := thinLTOFeedbackCandidates(pkgs)
	var overlays []string
	defer func() { removeThinLTOFeedbackFiles(overlays) }()
	finish := func() error {
		if len(knownDefinitions) == 0 {
			return publishThinLTOFeedbackOutput(feedbackOutput, outputPath)
		}
		removeThinLTOFeedbackFiles(overlays)
		overlays = nil
		unmarked := unmarkThinLTOFeedbackFunctions(pkgs)
		finalOverlays, err := materializeThinLTOFeedbackOverlays(ctx, pkgs, currentPlan, verbose)
		if err != nil {
			return err
		}
		overlays = finalOverlays
		if len(overlays) == 0 {
			return publishThinLTOFeedbackOutput(feedbackOutput, outputPath)
		}
		finalInputs := insertThinLTOOverlays(baseInputs, overlays)
		cleanupThinLTOFeedbackOutputTemps(feedbackOutput)
		if err := linkObjFiles(ctx, feedbackOutput, finalInputs, linkArgs, verbose); err != nil {
			return fmt.Errorf("thin LTO feedback final link: %w", err)
		}
		cleanupThinLTOFeedbackTemps(finalInputs)
		if verbose {
			fmt.Fprintf(os.Stderr, "llgo: ThinLTO feedback final link removed %d temporary noinline attributes and used %d overlays\n", unmarked, len(overlays))
		}
		return publishThinLTOFeedbackOutput(feedbackOutput, outputPath)
	}

	for round := 0; round < thinLTOFeedbackMaxRounds; round++ {
		cleanupThinLTOFeedbackOutputTemps(feedbackOutput)
		if err := linkObjFiles(ctx, feedbackOutput, currentInputs, linkArgs, verbose); err != nil {
			return fmt.Errorf("thin LTO feedback link round %d: %w", round+1, err)
		}
		modulePaths, err := thinLTOFeedbackModulePaths(currentInputs)
		if err != nil {
			return err
		}
		mods, dispose, err := parseThinLTOFeedbackModules(modulePaths)
		if err != nil {
			return err
		}
		roundDead := dcepass.DeadNoInlineFunctionsFromModulesWithDefinitions(
			mods,
			dceEntryRootCandidates(pkgs, needRuntime),
			candidates,
			knownDefinitions,
		)
		roundRefinedNames := dcepass.RefinedMethodNamesFromModules(mods, candidates)
		dispose()
		cleanupThinLTOFeedbackTemps(currentInputs)

		newFacts := 0
		for name := range roundDead {
			if _, seen := knownDead[name]; seen {
				continue
			}
			knownDead[name] = struct{}{}
			newFacts++
		}
		newRefinements := 0
		for owner, names := range roundRefinedNames {
			if reflect.DeepEqual(knownRefinedNames[owner], names) {
				continue
			}
			knownRefinedNames[owner] = append([]string(nil), names...)
			newRefinements++
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "llgo: ThinLTO feedback round %d found %d new dead function facts (%d cumulative) and %d refined MethodByName owners (%d cumulative)\n", round+1, newFacts, len(knownDead), newRefinements, len(knownRefinedNames))
			for owner, names := range roundRefinedNames {
				fmt.Fprintf(os.Stderr, "llgo: ThinLTO MethodByName refinement owner=%s names=%s\n", owner, strings.Join(names, ","))
			}
			unrefined, err := thinLTOFeedbackUnrefinedReflectOwners(pkgs, knownDead, knownRefinedNames)
			if err != nil {
				return fmt.Errorf("ThinLTO feedback reflection diagnostics: %w", err)
			}
			if len(unrefined) > 0 {
				fmt.Fprintf(os.Stderr, "llgo: ThinLTO feedback has %d remaining unrefined reflect owners: %s\n", len(unrefined), strings.Join(unrefined, ","))
			}
		}
		if newFacts == 0 && newRefinements == 0 {
			return finish()
		}

		nextPlan, err := buildDeadcodePlanWithFeedback(pkgs, needRuntime, knownDead, knownRefinedNames)
		if err != nil {
			return fmt.Errorf("thin LTO feedback round %d plan: %w", round+1, err)
		}
		if verbose && len(knownRefinedNames) > 0 {
			withoutRefinement, err := buildDeadcodePlanWithFeedback(pkgs, needRuntime, knownDead, nil)
			if err != nil {
				return fmt.Errorf("thin LTO feedback round %d comparison plan: %w", round+1, err)
			}
			fmt.Fprintf(os.Stderr, "llgo: ThinLTO MethodByName refinement changed live method slots from %d to %d\n", liveMethodSlotCount(withoutRefinement), liveMethodSlotCount(nextPlan))
		}
		if reflect.DeepEqual(currentPlan.LiveSlots, nextPlan.LiveSlots) {
			return finish()
		}
		if round+1 == thinLTOFeedbackMaxRounds {
			if verbose {
				fmt.Fprintf(os.Stderr, "llgo: ThinLTO feedback reached %d rounds with %d dead function facts\n", thinLTOFeedbackMaxRounds, len(knownDead))
			}
			return finish()
		}

		removeThinLTOFeedbackFiles(overlays)
		overlays, err = materializeThinLTOFeedbackOverlays(ctx, pkgs, nextPlan, verbose)
		if err != nil {
			return err
		}
		if len(overlays) == 0 {
			return finish()
		}
		currentInputs = insertThinLTOOverlays(baseInputs, overlays)
		currentPlan = nextPlan
		if verbose {
			fmt.Fprintf(os.Stderr, "llgo: ThinLTO feedback round %d removed %d new function facts and relinked %d overlays\n", round+1, newFacts, len(overlays))
		}
	}
	return finish()
}

func liveMethodSlotCount(plan deadcode.Plan) int {
	total := 0
	for _, slots := range plan.LiveSlots {
		total += len(slots)
	}
	return total
}

func thinLTOFeedbackUnrefinedReflectOwners(pkgs []Package, deadFunctions map[string]struct{}, refinedMethodNames map[string][]string) ([]string, error) {
	summary, err := meta.NewGlobalSummary(linkedPackageMetas(pkgs))
	if err != nil {
		return nil, err
	}
	var owners []string
	for _, name := range thinLTOFeedbackCandidates(pkgs) {
		if _, dead := deadFunctions[name]; dead {
			continue
		}
		if _, refined := refinedMethodNames[name]; refined {
			continue
		}
		sym, ok := summary.LookupSymbol(name)
		if !ok {
			continue
		}
		for _, demand := range summary.FuncDemands(sym) {
			if demand.Kind == meta.DemandReflectMethod {
				owners = append(owners, name)
				break
			}
		}
	}
	return owners, nil
}

func buildDeadcodePlanWithFeedback(pkgs []Package, needRuntime bool, deadFunctions map[string]struct{}, refinedMethodNames map[string][]string) (deadcode.Plan, error) {
	metas := linkedPackageMetas(pkgs)
	summary, err := meta.NewGlobalSummary(metas)
	if err != nil {
		return deadcode.Plan{}, err
	}
	return deadcode.BuildPlanWithFeedback(summary, dceEntryRootCandidates(pkgs, needRuntime), deadcode.Feedback{
		DeadFunctions:      deadFunctions,
		RefinedMethodNames: refinedMethodNames,
	}), nil
}

func thinLTOFeedbackKnownDefinitions(pkgs []Package) map[string]struct{} {
	known := make(map[string]struct{})
	noInlineKind := gllvm.AttributeKindID("noinline")
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Meta == nil || pkg.LPkg == nil {
			continue
		}
		mod := pkg.LPkg.Module()
		for _, name := range pkg.Meta.DemandFunctionNames() {
			fn := mod.NamedFunction(name)
			if fn.IsNil() || fn.IsDeclaration() || fn.GetEnumFunctionAttribute(noInlineKind).IsNil() {
				continue
			}
			known[name] = struct{}{}
		}
	}
	return known
}

func unmarkThinLTOFeedbackFunctions(pkgs []Package) int {
	unmarked := 0
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Meta == nil || pkg.LPkg == nil {
			continue
		}
		unmarked += dcepass.UnmarkNoInlineFunctions(pkg.LPkg.Module(), pkg.Meta.DemandFunctionNames())
	}
	return unmarked
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
