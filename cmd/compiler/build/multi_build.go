package build

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

// prepareBuildOutput implements the directory form of cmd/go's -o contract.
// A plain non-directory remains a valid output file for exactly one package;
// a directory output requires at least one main package.
func prepareBuildOutput(root, output string, multiple bool, pkgs []*packages.Package) (string, error) {
	if output == "" {
		return "", nil
	}
	resolved := resolvePath(root, output)
	info, statErr := os.Stat(resolved)
	isDirectory := statErr == nil && info.IsDir()
	trailingSeparator := strings.HasSuffix(output, "/") || strings.HasSuffix(output, `\`)
	if !isDirectory && !trailingSeparator && multiple {
		return "", fmt.Errorf("cannot write multiple packages to non-directory %s", output)
	}
	if !isDirectory && !trailingSeparator {
		return "", nil
	}
	hasMain := false
	for _, pkg := range pkgs {
		hasMain = hasMain || pkg.Name == "main"
	}
	if !hasMain {
		return "", errors.New("no main packages to build")
	}
	// On non-Windows hosts a final backslash is still accepted as the
	// documented directory marker rather than becoming part of the name.
	if !isDirectory && filepath.Separator != '\\' && strings.HasSuffix(output, `\`) {
		trimmed := strings.TrimSuffix(output, `\`)
		if trimmed == "" {
			resolved = root
		} else {
			resolved = resolvePath(root, trimmed)
		}
	}
	resolved = filepath.Clean(resolved)
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return "", fmt.Errorf("create build output directory %s: %w", resolved, err)
	}
	return resolved, nil
}

func defaultExecutableName(pkgPath string) string {
	name := path.Base(pkgPath)
	if len(name) > 1 && name[0] == 'v' && name[1] != '0' {
		if major, err := strconv.Atoi(name[1:]); err == nil && major >= 2 {
			name = path.Base(path.Dir(pkgPath))
		}
	}
	return name
}

// multiBuildFallback is used only after the shared graph fails. The normal
// path loads and compiles the union graph once; isolating roots here lets an
// unrelated good command still produce its output and makes every failure
// visible, at the acceptable cost of repeated failure-path work.
type multiBuildFallback struct {
	conf      *Config
	pkgs      []*packages.Package
	root      string
	outputDir string
}

func newMultiBuildFallback(conf *Config, pkgs []*packages.Package, root, outputDir string) *multiBuildFallback {
	return &multiBuildFallback{conf: conf.clone(), pkgs: pkgs, root: root, outputDir: outputDir}
}

func (fallback *multiBuildFallback) run() ([]Package, error) {
	results := make([][]Package, len(fallback.pkgs))
	errs := make([]error, len(fallback.pkgs))
	jobs := make(chan int, len(fallback.pkgs))
	workers := min(fallback.conf.parallelism(), len(fallback.pkgs))
	var outputLocks sync.Map
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				pkg := fallback.pkgs[index]
				conf := fallback.conf.clone()
				conf.BuildParallelism = 1
				conf.BuildTrace = ""
				conf.OutFile = ""
				compileOnly := pkg.Name != "main"
				discardOutput := ""
				if pkg.Name == "main" {
					if fallback.outputDir != "" {
						conf.OutFile = filepath.Join(fallback.outputDir, defaultExecutableName(pkg.PkgPath)+conf.AppExt)
					} else {
						var err error
						discardOutput, err = genTempOutputFile(defaultExecutableName(pkg.PkgPath), conf.AppExt)
						if err != nil {
							errs[index] = fmt.Errorf("%s: create temporary output: %w", pkg.PkgPath, err)
							continue
						}
						conf.OutFile = discardOutput
					}
				}
				var outputLock *sync.Mutex
				if conf.OutFile != "" {
					value, _ := outputLocks.LoadOrStore(conf.OutFile, new(sync.Mutex))
					outputLock = value.(*sync.Mutex)
					outputLock.Lock()
				}
				func() {
					if discardOutput != "" {
						defer removeTemporaryBuildOutputs(discardOutput)
					}
					if outputLock != nil {
						defer outputLock.Unlock()
					}
					errs[index] = runPackageJob(index, func(int) error {
						var err error
						results[index], err = Build(Invocation{
							Args: []string{pkg.PkgPath}, Config: conf, Dir: fallback.root,
							compileOnly: compileOnly, disableMultiFallback: true,
						})
						return err
					})
				}()
				if errs[index] != nil {
					errs[index] = fmt.Errorf("%s: %w", pkg.PkgPath, errs[index])
				}
			}
		}()
	}
	for index := range fallback.pkgs {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	var all []Package
	var failures []error
	for index := range results {
		all = append(all, results[index]...)
		if errs[index] != nil {
			failures = append(failures, errs[index])
		}
	}
	return all, errors.Join(failures...)
}

func removeTemporaryBuildOutputs(output string) {
	base := strings.TrimSuffix(output, filepath.Ext(output))
	removeOutFmts(&OutFmtDetails{
		Out: output, PCLN: pclnSidecarPath(output),
		Bin: base + ".bin", Hex: base + ".hex", Img: base + ".img",
		Uf2: base + ".uf2", Zip: base + ".zip",
	})
}
