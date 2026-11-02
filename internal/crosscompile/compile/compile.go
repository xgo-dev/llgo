package compile

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/clang"
)

const maxCompileWorkers = 4

type CompileOptions struct {
	CC      string // Compiler to use
	Linker  string
	CCFLAGS []string
	CFLAGS  []string
	LDFLAGS []string
}

type CompileGroup struct {
	OutputFileName string
	Files          []string // List of source files to compile
	CFlags         []string // C compiler flags
	CCFlags        []string
	LDFlags        []string // Linker flags
}

// IsCompiled checks if the compile group has already been compiled by verifying
// if the output archive file exists in the specified directory
func (g CompileGroup) IsCompiled(outputDir string) bool {
	archive := filepath.Join(outputDir, filepath.Base(g.OutputFileName))
	_, err := os.Stat(archive)
	return err == nil
}

// Compile compiles all source files in the group into a static library archive
// If the archive already exists, compilation is skipped
func (g CompileGroup) Compile(
	outputDir string, options CompileOptions,
) (err error) {
	if g.IsCompiled(outputDir) {
		return
	}
	tmpCompileDir, err := os.MkdirTemp("", "compile-group*")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmpCompileDir)

	compileLDFlags := append(slices.Clone(options.LDFLAGS), g.LDFlags...)
	compileCCFlags := append(slices.Clone(options.CCFLAGS), g.CCFlags...)
	compileCFFlags := append(slices.Clone(options.CFLAGS), g.CFlags...)

	cfg := clang.NewConfig(options.CC, compileCCFlags, compileCFFlags, compileLDFlags, options.Linker)

	// Keep archive members in source order even when compiles finish out of
	// order. Duplicate symbol resolution can depend on archive member order.
	// Each worker writes one distinct index; the backing array is never resized
	// and compileFiles joins every worker before the archive reads it.
	objFiles := make([]string, len(g.Files))

	archive := filepath.Join(outputDir, filepath.Base(g.OutputFileName))
	fmt.Fprintf(os.Stderr, "Start to compile group %s to %s...\n", g.OutputFileName, archive)

	err = compileFiles(len(g.Files), func(index int) error {
		file := g.Files[index]
		tempObjFile, err := os.CreateTemp(tmpCompileDir, objectFilePattern(file))
		if err != nil {
			return err
		}
		tempObjName := tempObjFile.Name()
		if err = tempObjFile.Close(); err != nil {
			return err
		}

		lang := "c"
		if filepath.Ext(file) == ".S" {
			lang = "assembler-with-cpp"
		}
		compiler := clang.NewCompiler(cfg)
		compiler.Verbose = true
		// Clang writes diagnostics in multiple chunks. Buffer each subprocess
		// so concurrent compiles cannot split another file's diagnostic lines.
		var stdout, stderr bytes.Buffer
		compiler.Stdout = &stdout
		compiler.Stderr = &stderr
		err = compiler.Compile("-o", tempObjName, "-x", lang, "-c", file)
		os.Stdout.Write(stdout.Bytes())
		os.Stderr.Write(stderr.Bytes())
		if err != nil {
			return fmt.Errorf("compile %s: %w", file, err)
		}

		objFiles[index] = tempObjName
		return nil
	})
	if err != nil {
		return err
	}

	ccDir := filepath.Dir(options.CC)
	llvmAr := filepath.Join(ccDir, "llvm-ar")

	responseFile, err := writeArchiveResponseFile(tmpCompileDir, objFiles)
	if err != nil {
		return err
	}
	// newlib contains hundreds of object files, whose expanded paths exceed
	// Windows' CreateProcess command-line limit. LLVM tools support response
	// files on every host, so keep the object list out of the process command
	// line rather than splitting one archive update into platform-only batches.
	cmd := exec.Command(llvmAr, "rcs", archive, "@"+responseFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("llvm-ar rcs %s: %w", archive, err)
	}
	return nil
}

// compileFiles bounds the out-of-process Clang workload independently of the
// number of library sources. Stop assigning work on failure, but join every
// active compile before the caller removes its temporary object directory.
func compileFiles(count int, compile func(int) error) error {
	var mu sync.Mutex
	var firstErr error
	next := 0
	var workers sync.WaitGroup
	// Respect restricted Go runtimes and avoid launching one Clang per CPU on
	// large hosts, where other compiler stages or CI jobs may also be active.
	for range min(count, runtime.GOMAXPROCS(0), maxCompileWorkers) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				mu.Lock()
				if firstErr != nil || next == count {
					mu.Unlock()
					return
				}
				index := next
				next++
				mu.Unlock()
				if err := compile(index); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			}
		}()
	}
	workers.Wait()
	return firstErr
}

func writeArchiveResponseFile(dir string, objFiles []string) (string, error) {
	var contents strings.Builder
	for _, objFile := range objFiles {
		// LLVM's response-file parser accepts forward slashes on Windows. Quote
		// each argument so temporary roots containing spaces remain one path.
		contents.WriteByte('"')
		contents.WriteString(filepath.ToSlash(objFile))
		contents.WriteString("\"\n")
	}
	responseFile := filepath.Join(dir, "objects.rsp")
	if err := os.WriteFile(responseFile, []byte(contents.String()), 0o600); err != nil {
		return "", err
	}
	return responseFile, nil
}

func objectFilePattern(source string) string {
	name := filepath.Base(source)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	return fmt.Sprintf("%s-*.o", name)
}

// CompileConfig represents compilation configuration
type CompileConfig struct {
	Groups       []CompileGroup
	ExportCFlags []string
}

type LibConfig struct {
	Url            string
	Name           string // Library name (e.g., "picolibc", "musl", "glibc")
	Version        string
	ResourceSubDir string
}

// String returns a string representation of the library configuration
// in the format "name-version"
func (cfg LibConfig) String() string {
	return fmt.Sprintf("%s-%s", cfg.Name, cfg.Version)
}
