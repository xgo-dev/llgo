package compile

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/goplus/llgo/internal/clang"
)

type CompileOptions struct {
	CC      string // Compiler to use
	Linker  string
	CCFLAGS []string
	CFLAGS  []string
	LDFLAGS []string
	Verbose bool // Print individual compiler commands
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

func compileJobs() (int, error) {
	if value := os.Getenv("LLGO_COMPILE_JOBS"); value != "" {
		jobs, err := strconv.Atoi(value)
		if err != nil || jobs < 1 {
			return 0, errors.New("invalid LLGO_COMPILE_JOBS " + strconv.Quote(value))
		}
		return jobs, nil
	}
	jobs := runtime.GOMAXPROCS(0)
	if jobs > 16 {
		return 16, nil
	}
	return jobs, nil
}

func compileVerbose(options CompileOptions) bool {
	return options.Verbose || os.Getenv("LLGO_COMPILE_VERBOSE") != ""
}

func groupCompilerConfig(g CompileGroup, options CompileOptions) clang.Config {
	compileLDFlags := append(slices.Clone(options.LDFLAGS), g.LDFlags...)
	compileCCFlags := append(slices.Clone(options.CCFLAGS), g.CCFlags...)
	compileCFFlags := append(slices.Clone(options.CFLAGS), g.CFlags...)
	return clang.NewConfig(options.CC, compileCCFlags, compileCFFlags, compileLDFlags, options.Linker)
}

func compileOneFile(compiler *clang.Cmd, tmpCompileDir string, idx int, file string) (objFile string, err error) {
	objFile = filepath.Join(tmpCompileDir, strconv.Itoa(idx)+".o")

	lang := "c"
	if filepath.Ext(file) == ".S" {
		lang = "assembler-with-cpp"
	}
	if err := compiler.Compile("-o", objFile, "-x", lang, "-c", file); err != nil {
		os.Remove(objFile)
		return "", err
	}
	return objFile, nil
}

func archiveObjects(llvmAr, outputDir string, g CompileGroup, objFiles []string) error {
	archive := filepath.Join(outputDir, filepath.Base(g.OutputFileName))
	args := []string{"rcs", archive}
	args = append(args, objFiles...)

	cmd := exec.Command(llvmAr, args...)
	// TODO(MeteorsLiu): support verbose
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Compile compiles all source files in the group into a static library archive.
// If the archive already exists, compilation is skipped.
func (g CompileGroup) Compile(outputDir string, options CompileOptions) error {
	return CompileConfig{Groups: []CompileGroup{g}}.Compile(outputDir, options)
}

// CompileConfig represents compilation configuration
type CompileConfig struct {
	Groups       []CompileGroup
	ExportCFlags []string
}

type compileGroupState struct {
	group CompileGroup
	cfg   clang.Config
	tmp   string
	objs  []string
	skip  bool
}

type compileTask struct {
	group int
	file  int
}

// Compile compiles all groups with one shared job budget, then archives each
// group in configuration order. Source files are independent, so sharing the
// worker pool removes unnecessary barriers between archive groups while keeping
// final archive and linker-flag order deterministic.
func (cfg CompileConfig) Compile(outputDir string, options CompileOptions) error {
	states := make([]compileGroupState, len(cfg.Groups))
	var tasks []compileTask
	for i, group := range cfg.Groups {
		states[i] = compileGroupState{group: group, cfg: groupCompilerConfig(group, options), objs: make([]string, len(group.Files))}
		if group.IsCompiled(outputDir) {
			states[i].skip = true
			continue
		}
		tmpCompileDir, err := os.MkdirTemp("", "compile-group*")
		if err != nil {
			return err
		}
		states[i].tmp = tmpCompileDir
		defer os.RemoveAll(tmpCompileDir)

		archive := filepath.Join(outputDir, filepath.Base(group.OutputFileName))
		os.Stderr.WriteString("Start to compile group " + group.OutputFileName + " to " + archive + "...\n")
		for j := range group.Files {
			tasks = append(tasks, compileTask{group: i, file: j})
		}
	}

	archiveJobs := 1
	if len(tasks) > 0 {
		jobs, err := compileJobs()
		if err != nil {
			return err
		}
		verbose := compileVerbose(options)
		if jobs > len(tasks) {
			jobs = len(tasks)
		}
		archiveJobs = jobs
		if jobs == 1 {
			for _, task := range tasks {
				state := &states[task.group]
				compiler := clang.NewCompiler(state.cfg)
				compiler.Verbose = verbose
				objFile, err := compileOneFile(compiler, state.tmp, task.file, state.group.Files[task.file])
				if err != nil {
					return err
				}
				state.objs[task.file] = objFile
			}
		} else if group, ok := singleTaskGroup(tasks); ok {
			if err := compileSingleGroupTasks(&states[group], jobs, verbose); err != nil {
				return err
			}
		} else {
			if err := compileTasks(states, tasks, jobs, verbose); err != nil {
				return err
			}
		}
	}

	return archiveGroups(outputDir, options, states, archiveJobs)
}

func archiveGroups(outputDir string, options CompileOptions, states []compileGroupState, jobs int) error {
	llvmAr := filepath.Join(filepath.Dir(options.CC), "llvm-ar")
	var groups []int
	for i := range states {
		if !states[i].skip {
			groups = append(groups, i)
		}
	}
	if jobs > len(groups) {
		jobs = len(groups)
	}
	if jobs <= 1 {
		for _, idx := range groups {
			state := &states[idx]
			if err := archiveObjects(llvmAr, outputDir, state.group, state.objs); err != nil {
				return err
			}
		}
		return nil
	}

	groupCh := make(chan int)
	errs := make([]error, len(states))
	var wg sync.WaitGroup
	for worker := 0; worker < jobs; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range groupCh {
				state := &states[idx]
				errs[idx] = archiveObjects(llvmAr, outputDir, state.group, state.objs)
			}
		}()
	}
	for _, idx := range groups {
		groupCh <- idx
	}
	close(groupCh)
	wg.Wait()
	for _, idx := range groups {
		if errs[idx] != nil {
			return errs[idx]
		}
	}
	return nil
}

func singleTaskGroup(tasks []compileTask) (int, bool) {
	if len(tasks) == 0 {
		return 0, false
	}
	group := tasks[0].group
	for _, task := range tasks[1:] {
		if task.group != group {
			return 0, false
		}
	}
	return group, true
}

func compileSingleGroupTasks(state *compileGroupState, jobs int, verbose bool) error {
	taskCh := make(chan int)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	setErr := func(err error) {
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	for worker := 0; worker < jobs; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			compiler := clang.NewCompiler(state.cfg)
			compiler.Verbose = verbose
			for idx := range taskCh {
				objFile, err := compileOneFile(compiler, state.tmp, idx, state.group.Files[idx])
				if err != nil {
					setErr(err)
					continue
				}
				state.objs[idx] = objFile
			}
		}()
	}
	for idx := range state.group.Files {
		errMu.Lock()
		stopped := firstErr != nil
		errMu.Unlock()
		if stopped {
			break
		}
		taskCh <- idx
	}
	close(taskCh)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return nil
}

func compileTasks(states []compileGroupState, tasks []compileTask, jobs int, verbose bool) error {
	taskCh := make(chan compileTask)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	setErr := func(err error) {
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	for worker := 0; worker < jobs; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			compilers := make([]*clang.Cmd, len(states))
			for task := range taskCh {
				state := &states[task.group]
				compiler := compilers[task.group]
				if compiler == nil {
					compiler = clang.NewCompiler(state.cfg)
					compiler.Verbose = verbose
					compilers[task.group] = compiler
				}
				objFile, err := compileOneFile(compiler, state.tmp, task.file, state.group.Files[task.file])
				if err != nil {
					setErr(err)
					continue
				}
				state.objs[task.file] = objFile
			}
		}()
	}
	for _, task := range tasks {
		errMu.Lock()
		stopped := firstErr != nil
		errMu.Unlock()
		if stopped {
			break
		}
		taskCh <- task
	}
	close(taskCh)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return nil
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
	return cfg.Name + "-" + cfg.Version
}
