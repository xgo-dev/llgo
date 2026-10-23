package build

import (
	"debug/elf"
	"go/build"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// Keep both original CI counterexamples: these are GCC sources, even though
// the selected runtime/cgo package also owns genuine Go assembler files.
func TestPackageAssemblyOriginalRuntimeCgoObjects(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			selection := build.Default
			selection.GOOS, selection.GOARCH, selection.CgoEnabled = "linux", arch, true
			dir := filepath.Join(runtime.GOROOT(), "src", "runtime", "cgo")
			selected, err := selection.ImportDir(dir, 0)
			if err != nil {
				t.Fatal(err)
			}
			ctx, pkg := assemblyLanguageTestFixture(t, "runtime/cgo", true, false)
			pkg.Dir = dir
			pkg.GoFiles, pkg.OtherFiles = nil, nil
			for _, name := range slices.Concat(selected.GoFiles, selected.CgoFiles) {
				pkg.GoFiles = append(pkg.GoFiles, filepath.Join(dir, name))
			}
			for _, name := range selected.SFiles {
				pkg.OtherFiles = append(pkg.OtherFiles, filepath.Join(dir, name))
			}
			ctx.buildConf.Goarch = arch
			target, machine := "x86_64-unknown-linux-gnu", elf.EM_X86_64
			if arch == "arm64" {
				target, machine = "aarch64-unknown-linux-gnu", elf.EM_AARCH64
			}
			ctx.crossCompile.CCArgs = []string{"--target=" + target}
			inputs, err := llgoPkgFileInputs(ctx, pkg)
			if err != nil {
				t.Fatal(err)
			}
			native := filepath.Join(dir, "gcc_"+arch+".S")
			if len(inputs) != 1 || inputs[0].path != native {
				t.Fatalf("original SDK native ownership = %+v, want %s", inputs, native)
			}
			plan9, err := pkgSFiles(ctx, pkg)
			if err != nil || len(plan9) == 0 || slices.Contains(plan9, native) {
				t.Fatalf("mixed original SDK Plan 9 ownership = %q, %v", plan9, err)
			}
			if _, err := digestLLGoFileInputs(ctx, inputs, nil); err != nil {
				t.Fatal(err)
			}
			object, err := clFile(ctx, inputs[0].compilerArgs, native, pkg.ExportFile, pkg.PkgPath, false)
			if err != nil {
				t.Fatal(err)
			}
			defer removeFiles([]string{object})
			parsed, err := elf.Open(object)
			if err != nil {
				t.Fatal(err)
			}
			actual := parsed.Machine
			if err := parsed.Close(); err != nil {
				t.Fatal(err)
			}
			if actual != machine {
				t.Fatalf("original GCC source object machine = %s, want %s", actual, machine)
			}
		})
	}
}
