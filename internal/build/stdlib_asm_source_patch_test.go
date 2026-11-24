//go:build go1.27

package build

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

func TestStdlibAsmSourcePatchesApplyToGo127SDK(t *testing.T) {
	runtimeDir := env.LLGoRuntimeDir()
	goroot := runtime.GOROOT()
	version := runtime.Version()

	cases := []struct {
		pkg      string
		goos     string
		goarch   string
		replaced []string
	}{
		{
			pkg:      "internal/runtime/syscall/windows",
			goos:     "windows",
			goarch:   "386",
			replaced: []string{"asm_windows_386.s"},
		},
		{
			pkg:      "internal/runtime/syscall/windows",
			goos:     "windows",
			goarch:   "amd64",
			replaced: []string{"asm_windows_amd64.s"},
		},
		{
			pkg:      "internal/runtime/syscall/windows",
			goos:     "windows",
			goarch:   "arm64",
			replaced: []string{"asm_windows_arm64.s"},
		},
		{
			pkg:      "internal/runtime/atomic",
			goos:     "windows",
			goarch:   "386",
			replaced: []string{"atomic_386.s"},
		},
		{
			pkg:      "internal/runtime/atomic",
			goos:     "linux",
			goarch:   "386",
			replaced: []string{"atomic_386.s"},
		},
		{
			pkg:      "internal/runtime/atomic",
			goos:     "windows",
			goarch:   "arm64",
			replaced: []string{"atomic_arm64.s"},
		},
		{
			pkg:      "internal/runtime/atomic",
			goos:     "darwin",
			goarch:   "arm64",
			replaced: []string{"atomic_arm64.s"},
		},
		{
			pkg:      "internal/runtime/gc/scan",
			goos:     "linux",
			goarch:   "amd64",
			replaced: []string{"expand_amd64.s", "scan_amd64.s", "filter_amd64.s"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.pkg+"/"+tc.goos+"/"+tc.goarch, func(t *testing.T) {
			ctx := sourcePatchBuildContext{goos: tc.goos, goarch: tc.goarch, goversion: version}
			changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, tc.pkg, ctx)
			if err != nil {
				t.Fatalf("applySourcePatchForPkg(%s): %v", tc.pkg, err)
			}
			if !changed {
				t.Fatalf("applySourcePatchForPkg(%s) reported no change", tc.pkg)
			}
			srcDir := filepath.Join(goroot, "src", filepath.FromSlash(tc.pkg))
			for _, name := range tc.replaced {
				got := string(overlay[filepath.Join(srcDir, name)])
				if got != "// replaced by LLGo source patch\n" {
					t.Fatalf("%s was not replaced by source patch: got %q", name, got)
				}
			}
			// Reapplying onto an existing overlay must re-validate cleanly.
			changed2, _, _, err := applySourcePatchForPkg(overlay, nil, runtimeDir, goroot, tc.pkg, ctx)
			if err != nil || !changed2 {
				t.Fatalf("reapplySourcePatchForPkg(%s): changed=%v err=%v", tc.pkg, changed2, err)
			}
		})
	}
}
