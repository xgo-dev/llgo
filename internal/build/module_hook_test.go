//go:build !llgo
// +build !llgo

package build

import (
	"strings"
	"testing"
)

func TestModuleHookReceivesPostABIPreOptimizationModule(t *testing.T) {
	conf := NewDefaultConf(ModeGen)

	counts := make(map[string]int)
	snapshots := make(map[string]string)
	conf.ModuleHook = func(pkg Package) {
		counts[pkg.PkgPath]++
		if _, ok := snapshots[pkg.PkgPath]; !ok {
			snapshots[pkg.PkgPath] = pkg.LPkg.String()
		}
	}

	pkgs, err := Do([]string{"../../cl/_testgo/localitycodegen"}, conf)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 initial package, got %d", len(pkgs))
	}

	mainPkg := pkgs[0].PkgPath
	if counts[mainPkg] != 1 {
		t.Fatalf("expected hook to fire once for %s, got %d", mainPkg, counts[mainPkg])
	}
	if snapshots[mainPkg] == "" {
		t.Fatalf("expected non-empty module snapshot for %s", mainPkg)
	}
	snapshot := snapshots[mainPkg]
	if !strings.Contains(snapshot, "define void @main.values(ptr sret({ i64, ptr, ptr })") {
		t.Fatalf("hook snapshot does not contain the lowered sret signature for main.values")
	}
	if strings.Contains(snapshot, "define { i64, ptr, ptr } @main.values()") {
		t.Fatalf("hook snapshot still contains the pre-ABI main.values signature")
	}
}
