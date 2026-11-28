package build

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/quoted"
	"golang.org/x/mod/modfile"
	gopackages "golang.org/x/tools/go/packages"
)

// effectiveGoWork determines the active workspace configuration or file path.
// It checks build flags (-gowork=...), GOFLAGS, GOWORK environment variable,
// and finally searches upward for go.work from dir.
func effectiveGoWork(dir string, cmdEnv commandEnv, buildFlags []string) string {
	for _, f := range buildFlags {
		if f == "-gowork=off" {
			return "off"
		}
		if f == "-gowork=auto" {
			break
		}
		if strings.HasPrefix(f, "-gowork=") {
			return strings.TrimPrefix(f, "-gowork=")
		}
	}
	goflags := commandEnvLookup(cmdEnv, "GOFLAGS")
	if goflags != "" {
		if words, err := quoted.Split(goflags); err == nil {
			for _, f := range words {
				if f == "-gowork=off" {
					return "off"
				}
				if f == "-gowork=auto" {
					break
				}
				if strings.HasPrefix(f, "-gowork=") {
					return strings.TrimPrefix(f, "-gowork=")
				}
			}
		}
	}
	gowork := commandEnvLookup(cmdEnv, "GOWORK")
	if gowork == "off" {
		return "off"
	}
	if gowork != "" && gowork != "auto" {
		return gowork
	}
	if dir == "" {
		dir = cmdEnv.dir
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	cur := filepath.Clean(dir)
	for {
		cand := filepath.Join(cur, "go.work")
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// recordEffectiveGoWork writes the effective workspace state and content hash
// into the hasher h, ensuring caches invalidate whenever the active workspace
// file content changes.
func recordEffectiveGoWork(h io.Writer, dir string, cmdEnv commandEnv, buildFlags []string, overlay map[string][]byte) {
	work := effectiveGoWork(dir, cmdEnv, buildFlags)
	if work == "" {
		fmt.Fprintf(h, "work=none\x00")
		return
	}
	if work == "off" {
		fmt.Fprintf(h, "work=off\x00")
		return
	}
	if !filepath.IsAbs(work) {
		base := dir
		if base == "" {
			base = cmdEnv.dir
		}
		if base != "" {
			work = filepath.Join(base, work)
		}
	}
	work = filepath.Clean(work)
	if overlay != nil {
		if data, ok := overlay[work]; ok {
			fmt.Fprintf(h, "workcontent=%s:%s\x00", work, digestBytes(data))
			return
		}
	}
	if data, err := os.ReadFile(work); err == nil {
		fmt.Fprintf(h, "workcontent=%s:%s\x00", work, digestBytes(data))
		return
	}
	if fi, err := os.Stat(work); err == nil {
		fmt.Fprintf(h, "workstat=%s:%d:%d\x00", work, fi.Size(), fi.ModTime().UnixNano())
		return
	}
	fmt.Fprintf(h, "workmissing=%s\x00", work)
}

// recordEffectiveGoMod records the content digest of the effective go.mod file.
func recordEffectiveGoMod(h io.Writer, dir string, modFile string, overlay map[string][]byte) {
	if modFile == "" {
		if dir == "" {
			dir, _ = os.Getwd()
		}
		cur := filepath.Clean(dir)
		for {
			cand := filepath.Join(cur, "go.mod")
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
				modFile = cand
				break
			}
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}
	if modFile == "" {
		fmt.Fprintf(h, "mod=none\x00")
		return
	}
	modFile = filepath.Clean(modFile)
	if overlay != nil {
		if data, ok := overlay[modFile]; ok {
			fmt.Fprintf(h, "modcontent=%s:%s\x00", modFile, digestBytes(data))
			return
		}
	}
	if data, err := os.ReadFile(modFile); err == nil {
		fmt.Fprintf(h, "modcontent=%s:%s\x00", modFile, digestBytes(data))
		return
	}
	if fi, err := os.Stat(modFile); err == nil {
		fmt.Fprintf(h, "modstat=%s:%d:%d\x00", modFile, fi.Size(), fi.ModTime().UnixNano())
		return
	}
	fmt.Fprintf(h, "modmissing=%s\x00", modFile)
}

// recordEffectiveGoEnv records the content digest of the effective GOENV file.
func recordEffectiveGoEnv(h io.Writer, cmdEnv commandEnv, overlay map[string][]byte) {
	goenv := commandEnvLookup(cmdEnv, "GOENV")
	if goenv == "off" {
		fmt.Fprintf(h, "goenv=off\x00")
		return
	}
	if goenv == "" {
		if cfgDir, err := os.UserConfigDir(); err == nil {
			goenv = filepath.Join(cfgDir, "go", "env")
		}
	}
	if goenv == "" {
		fmt.Fprintf(h, "goenv=none\x00")
		return
	}
	if !filepath.IsAbs(goenv) {
		base := cmdEnv.dir
		if base == "" {
			base, _ = os.Getwd()
		}
		if base != "" {
			sep := string(filepath.Separator)
			goenv = strings.TrimRight(base, sep) + sep + goenv
		}
	}
	if overlay != nil {
		if data, ok := overlay[goenv]; ok {
			fmt.Fprintf(h, "goenvcontent=%s:%s\x00", goenv, digestBytes(data))
			return
		}
		if cleaned := filepath.Clean(goenv); cleaned != goenv {
			if data, ok := overlay[cleaned]; ok {
				fmt.Fprintf(h, "goenvcontent=%s:%s\x00", goenv, digestBytes(data))
				return
			}
		}
		if resolved, err := filepath.EvalSymlinks(goenv); err == nil && resolved != goenv {
			if data, ok := overlay[resolved]; ok {
				fmt.Fprintf(h, "goenvcontent=%s:%s\x00", goenv, digestBytes(data))
				return
			}
		}
	}
	if data, err := os.ReadFile(goenv); err == nil {
		fmt.Fprintf(h, "goenvcontent=%s:%s\x00", goenv, digestBytes(data))
		return
	}
	if fi, err := os.Stat(goenv); err == nil {
		fmt.Fprintf(h, "goenvstat=%s:%d:%d\x00", goenv, fi.Size(), fi.ModTime().UnixNano())
		return
	}
	fmt.Fprintf(h, "goenvmissing=%s\x00", goenv)
}

// isolateNestedModuleWorkspace disables implicit parent go.work discovery when
// building inside a standalone nested Go module that is not listed in that
// workspace's use directives. Explicit GOWORK settings are always respected.
func isolateNestedModuleWorkspace(dir string, environ []string) []string {
	gowork := (commandEnv{environ: environ}).lookup("GOWORK")
	if gowork != "" && gowork != "auto" {
		return environ
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return environ
	}
	var modDir, workFile string
	for cur := absDir; ; cur = filepath.Dir(cur) {
		if modDir == "" {
			if info, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil && !info.IsDir() {
				modDir = cur
			}
		}
		if workFile == "" {
			if info, err := os.Stat(filepath.Join(cur, "go.work")); err == nil && !info.IsDir() {
				workFile = filepath.Join(cur, "go.work")
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur || (modDir != "" && workFile != "") {
			break
		}
	}
	if modDir == "" || workFile == "" {
		return environ
	}
	workDir := filepath.Dir(workFile)
	rel, err := filepath.Rel(workDir, modDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return environ
	}
	data, err := os.ReadFile(workFile)
	if err != nil {
		return environ
	}
	wf, err := modfile.ParseWork(workFile, data, nil)
	if err != nil {
		return environ
	}
	canonDir := func(p string) string {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Clean(resolved)
		}
		return filepath.Clean(p)
	}
	wantMod := canonDir(modDir)
	for _, u := range wf.Use {
		if u == nil || u.Path == "" {
			continue
		}
		useDir := u.Path
		if !filepath.IsAbs(useDir) {
			useDir = filepath.Join(workDir, filepath.FromSlash(useDir))
		}
		if canonDir(useDir) == wantMod {
			return environ
		}
	}
	return withEnv(environ, "GOWORK=off")
}

// go/packages queries release tags with GO111MODULE=off and no build flags.
// A module flag in GOFLAGS breaks that query. Move effective GOFLAGS (including
// persisted GOENV values) into explicit driver flags, before caller flags so
// their precedence is unchanged. A whitespace value suppresses GOENV fallback.
func applyPackageLoadFlags(conf *packages.Config, goflags string) error {
	flags, err := quoted.Split(goflags)
	if err != nil {
		return err
	}
	conf.BuildFlags = append(flags, conf.BuildFlags...)
	conf.Env = withEnv(conf.Env, "GOFLAGS= ")
	return nil
}

// Runtime implementation packages belong to the compiler's module. An
// explicit GOWORK applies even after changing directories, and GOFLAGS can
// select a caller modfile or vendor tree. Override those choices through
// command-line flags, which also take precedence over persistent GOENV flags.
// Only module selection is isolated: retain build tags, source overlays, and
// export package directories so both loads use the same compilation inputs.
func runtimePackageConfig(caller *packages.Config, dir string) packages.Config {
	conf := *caller
	conf.Dir = dir
	conf.Env = withEnv(caller.Env, "GOWORK=off", "GO111MODULE=on")
	conf.BuildFlags = append(slices.Clone(caller.BuildFlags), "-mod=readonly", "-modfile=")
	return conf
}

// List errors are attached to package roots even when Load returns nil error.
// Check before test filtering can discard a failing root. Go frontend export
// diagnostics beginning with '# ' are intentionally deferred: LLGo parses and
// checks patched sources itself and may support sources that gc cannot build.
func initialPackageListErrors(roots []*packages.Package) error {
	var failures []error
	for _, pkg := range roots {
		for _, err := range pkg.Errors {
			if err.Kind == gopackages.ListError && !strings.HasPrefix(err.Msg, "# ") {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}
