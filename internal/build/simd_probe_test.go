//go:build !llgo && llgo_simd_probe

package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/simd"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

const simdProbePath = "github.com/xgo-dev/llgo/internal/build/testdata/simd-probe"

type simdProbeLayout struct {
	Size    uint64   `json:"size"`
	Align   uint64   `json:"align"`
	Offsets []uint64 `json:"field_offsets,omitempty"`
}

type simdProbeType struct {
	GoType        string           `json:"go_type"`
	Descriptor    *simd.Descriptor `json:"simd_descriptor,omitempty"`
	GoNative      simdProbeLayout  `json:"go_native_layout"`
	GoLoaded      simdProbeLayout  `json:"go_loaded_layout"`
	LLGo          simdProbeLayout  `json:"llgo_layout"`
	LLVMType      string           `json:"llvm_storage_type"`
	LLVMStorage   simdProbeLayout  `json:"llvm_storage_layout"`
	LLVMNatural   *simdProbeLayout `json:"llvm_natural_vector_layout,omitempty"`
	LLVMNaturalTy string           `json:"llvm_natural_vector_type,omitempty"`
}

type simdProbeReport struct {
	Head             string                   `json:"head"`
	GoVersion        string                   `json:"go_version"`
	GOROOT           string                   `json:"goroot"`
	GOOS             string                   `json:"goos"`
	GOARCH           string                   `json:"goarch"`
	GOEXPERIMENT     string                   `json:"goexperiment"`
	Triple           string                   `json:"llvm_triple"`
	Features         string                   `json:"llvm_features"`
	DataLayout       string                   `json:"llvm_data_layout"`
	SourceSHA256     map[string]string        `json:"source_sha256"`
	NativeCompiled   bool                     `json:"native_compiled"`
	Types            map[string]simdProbeType `json:"types"`
	FunctionsBefore  map[string]string        `json:"functions_before_abi"`
	FunctionsAfter   map[string]string        `json:"functions_after_abi"`
	VectorAddLowered map[string]bool          `json:"vector_add_lowered"`
	// This is an observation across generated modules, not a reachability or
	// final-link proof: external objects may still supply some declarations.
	InitDirectCallees       []string            `json:"archsimd_init_direct_callees"`
	InitCalleesWithoutBody  []string            `json:"archsimd_init_callees_without_observed_body"`
	ArchsimdWithoutBody     []string            `json:"archsimd_declarations_without_observed_body"`
	InitHelperDirectCallees map[string][]string `json:"archsimd_init_helper_direct_callees"`
	InitHelperWithoutBody   map[string][]string `json:"archsimd_init_helper_callees_without_observed_body"`
	MissingSupport          []string            `json:"missing_support"`
}

// TestSIMDCompilerProbe is an opt-in investigation, not part of the default
// test suite. Run with -tags=llgo_simd_probe and LLGO_SIMD_PROBE_OUT set to a
// persistent artifact directory. Missing implementation must fail this probe;
// it is never an expected successful baseline and never calls t.Skip.
func TestSIMDCompilerProbe(t *testing.T) {
	out := os.Getenv("LLGO_SIMD_PROBE_OUT")
	if out == "" || !filepath.IsAbs(out) {
		t.Fatal("set LLGO_SIMD_PROBE_OUT to an absolute artifact directory")
	}
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"linux", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.arch, func(t *testing.T) {
			dir := filepath.Join(out, target.arch)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			report := simdProbeReport{
				GOOS: target.os, GOARCH: target.arch, GOEXPERIMENT: "simd",
				SourceSHA256: make(map[string]string), Types: make(map[string]simdProbeType),
				VectorAddLowered: make(map[string]bool),
			}
			defer func() {
				data, err := json.MarshalIndent(report, "", "  ")
				if err == nil {
					err = os.WriteFile(filepath.Join(dir, "report.json"), append(data, '\n'), 0644)
				}
				if err != nil {
					t.Error(err)
				}
				t.Logf("compiler probe artifacts: %s", dir)
				for _, missing := range report.MissingSupport {
					t.Error("missing-support: " + missing)
				}
			}()
			if head, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
				report.Head = strings.TrimSpace(string(head))
			} else {
				t.Fatal(err)
			}
			// The native compiler is a code-generation control, not a runtime
			// test and not an ABI that LLGo must duplicate. Both compile the
			// real experiment-gated fixture with the selected source toolchain.
			env := withEnv(os.Environ(), "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=simd",
				"GOOS="+target.os, "GOARCH="+target.arch, "GOAMD64=v1", "GOARM64=v8.0")
			version := exec.Command("go", "version")
			version.Env = env
			if data, err := version.Output(); err == nil {
				report.GoVersion = strings.TrimSpace(string(data))
			} else {
				t.Fatal(err)
			}
			goroot := exec.Command("go", "env", "GOROOT")
			goroot.Env = env
			if data, err := goroot.Output(); err == nil {
				report.GOROOT = strings.TrimSpace(string(data))
			} else {
				t.Fatal(err)
			}
			native := exec.Command("go", "build", "-gcflags=-S", "./testdata/simd-probe")
			native.Env = env
			assembly, nativeErr := native.CombinedOutput()
			if err := os.WriteFile(filepath.Join(dir, "native.s"), assembly, 0644); err != nil {
				t.Fatal(err)
			}
			report.NativeCompiled = nativeErr == nil
			if nativeErr != nil {
				report.MissingSupport = append(report.MissingSupport, "native source control failed: "+nativeErr.Error())
			}

			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			conf.GOAMD64, conf.GOARM64, conf.OptLevel = "v1", "v8.0", optlevel.O0
			conf.BuildParallelism = 1
			definitions, declarations, initCallees := map[string]bool{}, map[string]bool{}, map[string]bool{}
			archsimdBodies, archsimdCalls := map[string]string{}, map[string][]string{}
			var initIR strings.Builder
			conf.ModuleHook = func(pkg Package) {
				for fn := pkg.LPkg.Module().FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
					if fn.IsDeclaration() {
						declarations[fn.Name()] = true
						continue
					}
					definitions[fn.Name()] = true
					if pkg.PkgPath == "simd/archsimd" {
						archsimdBodies[fn.Name()] = fn.String()
						archsimdCalls[fn.Name()] = simdProbeDirectCallees(fn)
					}
					if strings.HasPrefix(fn.Name(), "simd/archsimd.init") {
						initIR.WriteString(fn.String())
						initIR.WriteByte('\n')
						for _, name := range archsimdCalls[fn.Name()] {
							initCallees[name] = true
						}
					}
				}
				if pkg.PkgPath != simdProbePath {
					return
				}
				if err := os.WriteFile(filepath.Join(dir, "before-abi.ll"), []byte(pkg.LPkg.String()), 0644); err != nil {
					t.Error(err)
				}
				report.FunctionsBefore = simdProbeFunctions(pkg.LPkg.Module())
				simdProbeTypes(t, pkg, &report)
			}
			pkgs, err := Do([]string{"./testdata/simd-probe"}, conf)
			for name := range declarations {
				if strings.HasPrefix(name, "simd/archsimd.") && !definitions[name] {
					report.ArchsimdWithoutBody = append(report.ArchsimdWithoutBody, name)
				}
			}
			for name := range initCallees {
				report.InitDirectCallees = append(report.InitDirectCallees, name)
				if strings.HasPrefix(name, "simd/archsimd.") && !definitions[name] {
					report.InitCalleesWithoutBody = append(report.InitCalleesWithoutBody, name)
				}
			}
			slices.Sort(report.ArchsimdWithoutBody)
			slices.Sort(report.InitDirectCallees)
			slices.Sort(report.InitCalleesWithoutBody)
			report.InitHelperDirectCallees, report.InitHelperWithoutBody = map[string][]string{}, map[string][]string{}
			// Record one helper layer explicitly. This exposes initializer
			// requirements such as init -> new64x2 -> Uint64x2.SetElem without
			// pretending to compute complete program reachability.
			for _, name := range report.InitDirectCallees {
				if body, ok := archsimdBodies[name]; ok {
					initIR.WriteString(body)
					initIR.WriteByte('\n')
					report.InitHelperDirectCallees[name] = archsimdCalls[name]
					for _, callee := range archsimdCalls[name] {
						if strings.HasPrefix(callee, "simd/archsimd.") && !definitions[callee] {
							report.InitHelperWithoutBody[name] = append(report.InitHelperWithoutBody[name], callee)
						}
					}
				}
			}
			if writeErr := os.WriteFile(filepath.Join(dir, "archsimd-init-functions.ll"), []byte(initIR.String()), 0644); writeErr != nil {
				t.Error(writeErr)
			}
			if err != nil {
				report.MissingSupport = append(report.MissingSupport, "fixture compilation failed: "+err.Error())
				return
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				report.MissingSupport = append(report.MissingSupport, "fixture module was not generated")
				return
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			if err := os.WriteFile(filepath.Join(dir, "after-abi.ll"), []byte(pkgs[0].LPkg.String()), 0644); err != nil {
				t.Fatal(err)
			}
			mod := pkgs[0].LPkg.Module()
			report.FunctionsAfter = simdProbeFunctions(mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				report.MissingSupport = append(report.MissingSupport, "LLVM verification failed: "+err.Error())
			}
			widths := []int{128}
			if target.arch == "amd64" {
				widths = []int{128, 256, 512}
			}
			for _, width := range widths {
				name := fmt.Sprintf("Add%d", width)
				fn := mod.NamedFunction(simdProbePath + "." + name)
				lowered := false
				if !fn.IsNil() {
					for _, bb := range fn.BasicBlocks() {
						for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
							if inst.InstructionOpcode() == llvm.FAdd && inst.Type().TypeKind() == llvm.VectorTypeKind &&
								inst.Type().VectorSize() == width/32 && inst.Type().ElementType().TypeKind() == llvm.FloatTypeKind {
								lowered = true
							}
						}
					}
				}
				report.VectorAddLowered[name] = lowered
				if !lowered {
					report.MissingSupport = append(report.MissingSupport, fmt.Sprintf("%s has no natural <%d x float> fadd", name, width/32))
				}
			}
		})
	}
}

func simdProbeFunctions(mod llvm.Module) map[string]string {
	functions := make(map[string]string)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if name, ok := strings.CutPrefix(fn.Name(), simdProbePath+"."); ok && !fn.IsDeclaration() {
			for _, line := range strings.Split(fn.String(), "\n") {
				if strings.HasPrefix(line, "define ") {
					functions[name] = line
					break
				}
			}
		}
	}
	return functions
}

func simdProbeDirectCallees(fn llvm.Value) []string {
	seen := map[string]bool{}
	for _, bb := range fn.BasicBlocks() {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if op := inst.InstructionOpcode(); op == llvm.Call || op == llvm.Invoke {
				if name := inst.CalledValue().Name(); name != "" {
					seen[name] = true
				}
			}
		}
	}
	callees := make([]string, 0, len(seen))
	for name := range seen {
		callees = append(callees, name)
	}
	slices.Sort(callees)
	return callees
}

func simdProbeTypes(t *testing.T, pkg Package, report *simdProbeReport) {
	t.Helper()
	prog, mod := pkg.LPkg.Prog, pkg.LPkg.Module()
	td := prog.TargetData()
	report.Triple, report.DataLayout = mod.Target(), mod.DataLayout()
	report.Features = prog.Target().Spec().Features
	archsimd := pkg.Imports["simd/archsimd"]
	if archsimd == nil {
		report.MissingSupport = append(report.MissingSupport, "real archsimd dependency was not loaded")
		return
	}
	stdDir, err := filepath.EvalSymlinks(filepath.Join(report.GOROOT, "src", "simd", "archsimd"))
	if err != nil {
		report.MissingSupport = append(report.MissingSupport, "cannot resolve selected archsimd source: "+err.Error())
		return
	}
	for _, path := range archsimd.GoFiles {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || filepath.Dir(resolved) != stdDir {
			report.MissingSupport = append(report.MissingSupport, "archsimd source is outside selected GOROOT: "+path)
			return
		}
	}
	// The probe records the actual loader paths/hashes rather than manufacturing
	// a second package called simd/archsimd with matching type names.
	for _, path := range append(append([]string(nil), pkg.GoFiles...), archsimd.GoFiles...) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Error(err)
			continue
		}
		report.SourceSHA256[path] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	classifier, err := simd.NewClassifier(report.GOARCH, archsimd.Types)
	if err != nil {
		report.MissingSupport = append(report.MissingSupport, "archsimd markers: "+err.Error())
		return
	}
	for _, name := range []string{"Scalar128", "Vector128", "Container128", "Vector256", "Container256", "Vector512", "Container512"} {
		obj := pkg.Types.Scope().Lookup(name)
		if obj == nil {
			continue
		}
		raw := obj.Type()
		global := mod.NamedGlobal(simdProbePath + "." + name)
		if global.IsNil() {
			report.MissingSupport = append(report.MissingSupport, name+" storage global is missing")
			continue
		}
		lltyp := global.GlobalValueType()
		translated := prog.Type(raw, llssa.InGo)
		entry := simdProbeType{
			GoType:   raw.String(),
			GoNative: simdProbeGoLayout(types.SizesFor("gc", report.GOARCH), raw),
			GoLoaded: simdProbeGoLayout(pkg.TypesSizes, raw),
			LLGo:     simdProbeLayout{Size: prog.SizeOf(translated), Align: prog.AlignOf(translated)},
			LLVMType: lltyp.String(), LLVMStorage: simdProbeLLVMLayout(td, lltyp),
		}
		if desc, ok := classifier.Classify(raw); ok {
			entry.Descriptor = &desc
			natural := llvm.VectorType(mod.Context().FloatType(), desc.Lanes)
			layout := simdProbeLLVMLayout(td, natural)
			entry.LLVMNatural, entry.LLVMNaturalTy = &layout, natural.String()
			if entry.LLVMStorage.Size != uint64(desc.GoLayout.Size) || entry.LLVMStorage.Align != uint64(desc.GoLayout.Align) {
				report.MissingSupport = append(report.MissingSupport,
					fmt.Sprintf("%s LLVM storage size/alignment %d/%d differs from Go SIMD %d/%d", name,
						entry.LLVMStorage.Size, entry.LLVMStorage.Align, desc.GoLayout.Size, desc.GoLayout.Align))
			}
		}
		if strings.HasPrefix(name, "Container") {
			if entry.LLVMStorage.Size != entry.GoNative.Size || entry.LLVMStorage.Align != entry.GoNative.Align ||
				len(entry.LLVMStorage.Offsets) != 3 || entry.LLVMStorage.Offsets[1] != 8 {
				report.MissingSupport = append(report.MissingSupport, name+" does not retain native Go SIMD field layout")
			}
		}
		report.Types[name] = entry
	}
}

func simdProbeGoLayout(sizes types.Sizes, typ types.Type) simdProbeLayout {
	if sizes == nil {
		return simdProbeLayout{}
	}
	layout := simdProbeLayout{Size: uint64(sizes.Sizeof(typ)), Align: uint64(sizes.Alignof(typ))}
	if st, ok := typ.Underlying().(*types.Struct); ok {
		fields := make([]*types.Var, st.NumFields())
		for i := range fields {
			fields[i] = st.Field(i)
		}
		for _, offset := range sizes.Offsetsof(fields) {
			layout.Offsets = append(layout.Offsets, uint64(offset))
		}
	}
	return layout
}

func simdProbeLLVMLayout(td llvm.TargetData, typ llvm.Type) simdProbeLayout {
	layout := simdProbeLayout{Size: td.TypeAllocSize(typ), Align: uint64(td.ABITypeAlignment(typ))}
	if typ.TypeKind() == llvm.StructTypeKind {
		for i := range typ.StructElementTypes() {
			layout.Offsets = append(layout.Offsets, td.ElementOffset(typ, i))
		}
	}
	return layout
}
