package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"go/version"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/goplus/gogen"
	"golang.org/x/tools/go/packages"
)

type fieldCheck struct {
	// owner indexes the package's subjects. Distinct generic instances can
	// print the same typeName when their arguments contain private fields.
	owner                      int
	typeName, field, operation string
	wrapper                    string
	file                       string
	line                       int
	kind                       types.BasicKind
	sizeIndex                  int
}

type fieldLayout struct {
	offset, size, width     int64
	objectAlign, fieldAlign int64
}

type layoutStep struct {
	field *types.Var
	index int64
}

type nestedCheck struct {
	owner                               int
	path                                []layoutStep
	offset, baseAlign, alignment, width int64
	sizeIndex                           int
}

type fieldSizeCheck struct {
	typ   types.Type
	width int64
}

type fieldSizeResult struct {
	Expected      uint64 `json:"expected"`
	Actual        uint64 `json:"actual"`
	Observed      bool   `json:"observed"`
	FirstAddress  string `json:"first_address,omitempty"`
	SecondAddress string `json:"second_address,omitempty"`
}

const accessGuard = 16

type accessResult struct {
	ExpectedValue string `json:"expected_value"`
	ActualValue   string `json:"actual_value,omitempty"`
	ValueMatches  bool   `json:"value_matches"`
	MemoryMatches bool   `json:"memory_matches"`
	ChangedBytes  []int  `json:"unexpected_changed_bytes,omitempty"`
}

type alignmentResult struct {
	ObjectRequired int64  `json:"object_required"`
	FieldRequired  int64  `json:"field_required"`
	ObjectAddress  string `json:"object_address"`
	FieldAddress   string `json:"field_address"`
	ObjectMatches  bool   `json:"object_matches"`
	FieldMatches   bool   `json:"field_matches"`
}

type strideResult struct {
	FirstAddress    string `json:"first_address"`
	SecondAddress   string `json:"second_address"`
	Bytes           uint64 `json:"bytes"`
	GoMatches       bool   `json:"go_matches"`
	MetadataMatches bool   `json:"metadata_matches"`
}

type lldbSession struct {
	input  io.WriteCloser
	output *bufio.Reader
	log    *os.File
	script *os.File
}

func (s *lldbSession) run(command string) (string, error) {
	if _, err := fmt.Fprintln(s.script, command); err != nil {
		return "", err
	}
	// A settings response delimits each request even when LLDB's stdin is a
	// pipe. It does not evaluate an expression or execute code in the target.
	if _, err := fmt.Fprintf(s.input, "%s\nsettings show prompt\n", command); err != nil {
		return "", err
	}
	var output strings.Builder
	for {
		line, err := s.output.ReadString('\n')
		if _, writeErr := io.WriteString(s.log, line); writeErr != nil {
			return output.String(), writeErr
		}
		if strings.TrimSpace(line) == `prompt (string) = "layoutlint>"` {
			break
		}
		output.WriteString(line)
		if err != nil {
			return output.String(), fmt.Errorf("LLDB command %q: %w", command, err)
		}
	}
	text := output.String()
	if strings.HasPrefix(text, "error:") || strings.Contains(text, "\nerror:") {
		return text, fmt.Errorf("LLDB command %q failed; see lldb.log", command)
	}
	return text, nil
}

type offsetResult struct {
	checkIndex     int
	Type           string           `json:"type"`
	Field          string           `json:"field"`
	Operation      string           `json:"operation"`
	File           string           `json:"file"`
	Line           int              `json:"line"`
	Expected       int64            `json:"expected_offset"`
	Actual         int64            `json:"actual_offset"`
	Observed       bool             `json:"observed"`
	AddressCase    int              `json:"address_case"`
	Alignment      *alignmentResult `json:"alignment,omitempty"`
	Access         *accessResult    `json:"access,omitempty"`
	Size           *fieldSizeResult `json:"size"`
	MetadataOffset *uint64          `json:"metadata_offset,omitempty"`
}

func command(directory, logPath string, args ...string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = directory
	output, err := cmd.CombinedOutput()
	log := fmt.Sprintf("%q\n%s", args, output)
	if writeErr := os.WriteFile(logPath, []byte(log), 0644); writeErr != nil {
		return output, writeErr
	}
	if err != nil {
		return output, fmt.Errorf("%s: %w; see %s", args[0], err, logPath)
	}
	return output, nil
}

func goOffsets(root, directory, baselineDirectory string, pkg *packages.Package, tests bool, patterns []string, checks []fieldCheck, retained []metadataCheck, nested []nestedCheck, dependencies map[string]string, ownerBuild bool) ([]fieldLayout, []goMetadata, error) {
	// Load export paths separately: asking for dependencies and syntax in the
	// same load would retain the syntax trees of the entire dependency graph.
	metadata, err := packages.Load(&packages.Config{
		Dir: root, Tests: tests,
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedCompiledGoFiles,
	}, patterns...)
	if err != nil {
		return nil, nil, err
	}
	if packages.PrintErrors(metadata) != 0 {
		return nil, nil, fmt.Errorf("cannot load Go dependency archives")
	}
	var input *packages.Package
	for _, candidate := range metadata {
		if candidate.ID == pkg.ID {
			input = candidate
			break
		}
	}
	if input == nil {
		return nil, nil, fmt.Errorf("Go export data missing for %s", pkg.ID)
	}
	imports := make(map[string]string)
	seen := make(map[string]bool)
	var visit func(*packages.Package)
	visit = func(current *packages.Package) {
		if seen[current.ID] {
			return
		}
		seen[current.ID] = true
		if current.ExportFile != "" {
			imports[current.PkgPath] = current.ExportFile
		}
		for _, dependency := range current.Imports {
			visit(dependency)
		}
	}
	for _, dependency := range input.Imports {
		visit(dependency)
	}
	var names []string
	for name := range imports {
		names = append(names, name)
	}
	slices.Sort(names)
	var importcfg strings.Builder
	for _, name := range names {
		fmt.Fprintf(&importcfg, "packagefile %s=%s\n", name, imports[name])
	}
	for path, dependency := range input.Imports {
		if path != dependency.PkgPath {
			fmt.Fprintf(&importcfg, "importmap %s=%s\n", path, dependency.PkgPath)
		}
	}
	configPath := filepath.Join(directory, "importcfg")
	if err := os.WriteFile(configPath, []byte(importcfg.String()), 0644); err != nil {
		return nil, nil, err
	}
	// Go emits the reference and storage-model descriptors used by every
	// layout check. No generated expression supplies a separate layout fact.
	models, layoutNames, err := goLayoutModels(pkg, retained)
	if err != nil {
		return nil, nil, err
	}
	modelSource := ""
	if models != "" {
		modelSource = filepath.Join(directory, "go-models.go")
		if err := os.WriteFile(modelSource, []byte(models), 0644); err != nil {
			return nil, nil, err
		}
	}
	var declarations strings.Builder
	fmt.Fprintf(&declarations, "package %s\n", pkg.Name)
	for i, check := range retained {
		name := fmt.Sprintf("layoutlintABIReference%d", i)
		if pkg.Types.Scope().Lookup(name) != nil {
			return nil, nil, fmt.Errorf("%s: generated reference %s already exists", pkg.PkgPath, name)
		}
		// A named interface global gives aliases an exact descriptor relocation;
		// no Go executable or reflection getter is needed to identify T.
		fmt.Fprintf(&declarations, "var %s interface{} = (*%s)(nil)\n", name, check.identifier)
		if layoutNames[i] != check.identifier {
			name = fmt.Sprintf("layoutlintStorageReference%d", i)
			if pkg.Types.Scope().Lookup(name) != nil {
				return nil, nil, fmt.Errorf("%s: generated reference %s already exists", pkg.PkgPath, name)
			}
			fmt.Fprintf(&declarations, "var %s interface{} = (*%s)(nil)\n", name, layoutNames[i])
		}
	}
	referenceSource := filepath.Join(directory, "go-references.go")
	if err := os.WriteFile(referenceSource, []byte(declarations.String()), 0644); err != nil {
		return nil, nil, err
	}
	args := []string{"go", "tool", "compile", "-S", "-S", "-p", pkg.PkgPath, "-importcfg", configPath, "-o", filepath.Join(directory, "reference.a")}
	if pkg.Module != nil && pkg.Module.GoVersion != "" {
		args = append(args, "-lang="+version.Lang("go"+pkg.Module.GoVersion))
	}
	sourceFiles := make(map[string]bool, len(pkg.GoFiles))
	for _, file := range pkg.GoFiles {
		sourceFiles[file] = true
	}
	rewrittenSource := make(map[string]bool)
	for _, file := range pkg.Syntax {
		if original := pkg.Fset.PositionFor(file.Pos(), true); sourceFiles[original.Filename] {
			rewrittenSource[pkg.Fset.PositionFor(file.Pos(), false).Filename] = true
		}
	}
	for i, file := range pkg.CompiledGoFiles {
		if !sourceFiles[file] {
			// Support files need _cgo_ for their directives. Processed user
			// files must keep ordinary names: _cgo_ makes their types non-local
			// and rejects methods such as Foo.Print in cl/_testrt/struct.
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, nil, err
			}
			name := fmt.Sprintf("_cgo_layoutlint_%d.go", i)
			if rewrittenSource[file] {
				name = fmt.Sprintf("layoutlint_cgo_source_%d.go", i)
			}
			file = filepath.Join(directory, name)
			if err := os.WriteFile(file, data, 0644); err != nil {
				return nil, nil, err
			}
		}
		args = append(args, file)
	}
	args = append(args, referenceSource)
	if modelSource != "" {
		args = append(args, modelSource)
	}
	if ownerBuild {
		// The defining package can contain assembly (reflect is one example).
		// Let cmd/go provide symabis and the package's other compiler settings.
		replacements := make(map[string]string)
		for _, path := range []string{referenceSource, modelSource} {
			if path == "" {
				continue
			}
			logical := filepath.Join(filepath.Dir(pkg.GoFiles[0]), "layoutlint_"+strings.ReplaceAll(filepath.Base(path), "-", "_"))
			if _, err := os.Stat(logical); !os.IsNotExist(err) {
				return nil, nil, fmt.Errorf("Go baseline overlay destination must not exist: %s", logical)
			}
			replacements[logical] = path
		}
		data, err := json.Marshal(struct{ Replace map[string]string }{replacements})
		if err != nil {
			return nil, nil, err
		}
		path := filepath.Join(directory, "go-overlay.json")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, nil, err
		}
		args = []string{"go", "build", "-buildmode=archive", "-overlay=" + path, "-gcflags=" + pkg.PkgPath + "=-S -S", "-o", filepath.Join(directory, "reference.a"), pkg.PkgPath}
	}
	fmt.Println("  Go layout and ABI metadata:", pkg.PkgPath)
	logPath := filepath.Join(directory, "go-compile.log")
	log, err := os.Create(logPath)
	if err != nil {
		return nil, nil, err
	}
	if _, err := fmt.Fprintf(log, "%q\n", args); err != nil {
		log.Close()
		return nil, nil, err
	}
	// Assembly listings can be much larger than the descriptors we need.
	// Stream the compiler output to disk instead of retaining it in memory.
	compiler := exec.Command(args[0], args[1:]...)
	compiler.Dir = root
	compiler.Stdout, compiler.Stderr = log, log
	compileErr := compiler.Run()
	closeErr := log.Close()
	if compileErr != nil {
		return nil, nil, fmt.Errorf("Go baseline compilation: %w; see %s", compileErr, logPath)
	}
	if closeErr != nil {
		return nil, nil, closeErr
	}
	typeMetadata, err := readGoMetadata(root, directory, baselineDirectory, pkg.PkgPath, retained, layoutNames, dependencies)
	if err != nil {
		return nil, nil, err
	}
	for i := range nested {
		check := &nested[i]
		current := typeMetadata[check.owner].storage
		alignment := current.header.Align
		for _, step := range check.path {
			check.baseAlign = int64(alignment)
			if step.field != nil {
				index := slices.IndexFunc(current.fields, func(field goMetadataField) bool { return field.Name == step.field.Name() })
				if index == -1 {
					return nil, nil, fmt.Errorf("Go metadata %s: missing field %s", retained[check.owner].name, step.field.Name())
				}
				field := &current.fields[index]
				check.offset = int64(field.Offset)
				current = field.storage
				alignment = current.header.FieldAlign
			} else {
				if current.elem == nil {
					return nil, nil, fmt.Errorf("Go metadata %s: missing array element", retained[check.owner].name)
				}
				current = current.elem
				check.offset = step.index * int64(current.header.Size)
				alignment = current.header.Align
			}
		}
		check.alignment = int64(alignment)
		check.width = int64(current.header.Size)
	}
	layouts := make([]fieldLayout, len(checks))
	for i, check := range checks {
		storage := typeMetadata[check.owner].storage
		index := slices.IndexFunc(storage.fields, func(field goMetadataField) bool { return field.Name == check.field })
		if index == -1 {
			return nil, nil, fmt.Errorf("Go metadata %s: missing field %s", check.typeName, check.field)
		}
		field := &storage.fields[index]
		layouts[i] = fieldLayout{
			offset: int64(field.Offset), size: int64(storage.header.Size), width: int64(field.storage.header.Size),
			objectAlign: int64(storage.header.Align), fieldAlign: int64(field.storage.header.FieldAlign),
		}
		if check.kind != types.Invalid {
			if layouts[i].width <= 0 || layouts[i].offset > layouts[i].size-layouts[i].width {
				return nil, nil, fmt.Errorf("incomplete Go size/width for %s.%s", check.typeName, check.field)
			}
		}
	}
	return layouts, typeMetadata, nil
}

func checkOffsets(root, directory, source string, pkg *packages.Package, tests bool, checks []fieldCheck, layouts []fieldLayout, metadata []metadataCheck, goMetadata []goMetadata, nested []nestedCheck, sizes []fieldSizeCheck, private []privateLayout) (err error) {
	var results []offsetResult
	sizeResults := make([]fieldSizeResult, len(sizes))
	for i, check := range sizes {
		sizeResults[i].Expected = uint64(check.width)
	}
	nestedResults := make([]offsetResult, len(nested))
	for i, check := range nested {
		var path strings.Builder
		var position token.Pos
		operation := "nested-field"
		for _, step := range check.path {
			if step.field != nil {
				if path.Len() != 0 {
					path.WriteByte('.')
				}
				path.WriteString(step.field.Name())
				position = step.field.Pos()
				if operation == "array-element" {
					operation = "array-field"
				}
			} else {
				fmt.Fprintf(&path, "[%d]", step.index)
				operation = "array-element"
			}
		}
		location := pkg.Fset.PositionFor(position, false)
		nestedResults[i] = offsetResult{Type: metadata[check.owner].name, Field: path.String(), Operation: operation, File: location.Filename, Line: location.Line, Expected: check.offset, Size: &sizeResults[check.sizeIndex]}
	}
	metadataResults := make([]metadataResult, len(metadata))
	for i := range private {
		plan := &private[i]
		plan.Metadata = metadataResult{Type: plan.pkg.PkgPath + "." + plan.subject.name, File: plan.subject.file, Line: plan.subject.line, Go: &plan.baseline}
		for _, field := range plan.baseline.Fields {
			plan.Metadata.Fields = append(plan.Metadata.Fields, metadataFieldResult{Name: field.Name, Expected: field.Offset})
		}
		for _, check := range plan.paths {
			var name strings.Builder
			var position token.Pos
			for _, step := range check.path {
				if step.field != nil {
					if name.Len() != 0 {
						name.WriteByte('.')
					}
					name.WriteString(step.field.Name())
					position = step.field.Pos()
				} else {
					fmt.Fprintf(&name, "[%d]", step.index)
				}
			}
			location := plan.pkg.Fset.PositionFor(position, true)
			operation := "private-field"
			if check.path[len(check.path)-1].field == nil {
				operation = "array-element"
			}
			plan.Fields = append(plan.Fields, offsetResult{Type: plan.Metadata.Type, Field: name.String(), Operation: operation, File: location.Filename, Line: location.Line, Expected: check.offset, Size: &fieldSizeResult{Expected: uint64(check.width)}})
		}
	}
	metadataFields := make([][]metadataFieldCheck, len(metadata))
	for i, check := range metadata {
		metadataResults[i] = metadataResult{Type: check.name, File: check.file, Line: check.line, Go: &goMetadata[i]}
		for _, field := range goMetadata[i].Fields {
			metadataResults[i].Fields = append(metadataResults[i].Fields, metadataFieldResult{Name: field.Name, Expected: field.Offset})
		}
		st := types.Unalias(check.typ).Underlying().(*types.Struct)
		for pair := 0; pair < len(checks); pair += 2 {
			if checks[pair].owner != i {
				continue
			}
			for field := 0; field < st.NumFields(); field++ {
				if st.Field(field).Name() == checks[pair].field {
					metadataFields[i] = append(metadataFields[i], metadataFieldCheck{field: checks[pair].field, path: []layoutStep{{field: st.Field(field)}}})
					break
				}
			}
		}
	}
	for i, check := range nested {
		if check.path[len(check.path)-1].field != nil {
			metadataFields[check.owner] = append(metadataFields[check.owner], metadataFieldCheck{field: nestedResults[i].Field, path: check.path})
		}
	}
	// Each pair is one field's reader and writer. Scalar pairs run at two
	// addresses; their wrapper IDs remain the same in both address cases.
	for pair := 0; pair < len(checks); pair += 2 {
		addressCases := 1
		if checks[pair].kind != types.Invalid {
			addressCases = 2
		}
		for addressCase := 0; addressCase < addressCases; addressCase++ {
			for i := pair; i < pair+2; i++ {
				check := checks[i]
				results = append(results, offsetResult{checkIndex: i, Type: check.typeName, Field: check.field, Operation: check.operation, File: check.file, Line: check.line, Expected: layouts[i].offset, AddressCase: addressCase, Size: &sizeResults[check.sizeIndex]})
			}
		}
	}
	defer func() {
		if err == nil {
			return
		}
		report := struct {
			Package  string           `json:"package"`
			Error    string           `json:"error,omitempty"`
			Results  []offsetResult   `json:"results"`
			Arrays   []offsetResult   `json:"array_elements,omitempty"`
			Nested   []offsetResult   `json:"nested_fields,omitempty"`
			Metadata []metadataResult `json:"metadata"`
			Private  []privateLayout  `json:"private_layouts,omitempty"`
		}{Package: pkg.PkgPath, Results: results, Metadata: metadataResults, Private: private}
		for _, result := range nestedResults {
			if result.Operation == "nested-field" {
				report.Nested = append(report.Nested, result)
			} else {
				report.Arrays = append(report.Arrays, result)
			}
		}
		if err != nil {
			report.Error = err.Error()
		}
		data, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr == nil {
			marshalErr = os.WriteFile(filepath.Join(directory, "offsets.json"), append(data, '\n'), 0644)
		}
		if marshalErr != nil {
			err = fmt.Errorf("%w; write offsets.json: %v", err, marshalErr)
		}
		if annotationErr := writeAnnotations(directory, source, pkg, checks, results, nestedResults, metadataResults, private); annotationErr != nil {
			err = fmt.Errorf("%w; annotate source: %v", err, annotationErr)
		}
	}()

	logical := filepath.Join(filepath.Dir(pkg.GoFiles[0]), filepath.Base(source))
	if _, err := os.Stat(logical); !os.IsNotExist(err) {
		return fmt.Errorf("generated source destination must not exist: %s", logical)
	}
	replacements := map[string]string{logical: source}
	for _, plan := range private {
		for _, name := range plan.logical {
			if _, err := os.Stat(name); !os.IsNotExist(err) {
				return fmt.Errorf("generated private source destination must not exist: %s", name)
			}
			replacements[name] = plan.source
		}
	}
	overlay, err := json.Marshal(struct{ Replace map[string]string }{replacements})
	if err != nil {
		return err
	}
	overlayPath := filepath.Join(directory, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0644); err != nil {
		return err
	}
	llgo := os.Getenv("LLGO")
	if llgo == "" {
		llgo = "llgo"
	}
	executable := filepath.Join(directory, "observed")
	args := []string{llgo, "build"}
	if tests {
		args = []string{llgo, "test", "-c"}
	} else if pkg.Name != "main" {
		return fmt.Errorf("%s is a library; use -test to build a test executable", pkg.PkgPath)
	}
	args = append(args, "-O0", "-ldflags=-w=false", "-overlay="+overlayPath, "-o", executable)
	if pkg.PkgPath == "command-line-arguments" || pkg.ForTest == "command-line-arguments" {
		args = append(args, pkg.GoFiles...)
		args = append(args, logical)
	} else if pkg.ForTest != "" {
		args = append(args, pkg.ForTest)
	} else {
		args = append(args, pkg.PkgPath)
	}
	fmt.Println("  LLGo access wrappers:", pkg.PkgPath)
	if _, err := command(root, filepath.Join(directory, "llgo-build.log"), args...); err != nil {
		return err
	}

	lldb := os.Getenv("LLGO_LLDB")
	if lldb == "" {
		lldb = "lldb"
	}
	log, err := os.Create(filepath.Join(directory, "lldb.log"))
	if err != nil {
		return err
	}
	defer log.Close()
	script, err := os.Create(filepath.Join(directory, "offsets.lldb"))
	if err != nil {
		return err
	}
	defer script.Close()
	launchPath := filepath.Join(directory, "launch.lldb")
	if err := os.WriteFile(launchPath, []byte("process launch --no-stdio\n"), 0644); err != nil {
		return err
	}
	debugger := exec.Command(lldb, "--no-lldbinit", "--no-use-colors", "-O", "settings set prompt layoutlint>", "--", executable)
	debugger.Dir = root
	input, err := debugger.StdinPipe()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := debugger.StdoutPipe()
	if err != nil {
		return err
	}
	debugger.Stderr = debugger.Stdout
	if err := debugger.Start(); err != nil {
		return err
	}
	session := lldbSession{input: input, output: bufio.NewReader(output), log: log, script: script}
	defer func() {
		// A failed bounds check must exit with the inferior still stopped,
		// before its pending load or store can touch unowned memory.
		fmt.Fprintln(input, "quit")
		input.Close()
		io.Copy(log, session.output)
		if waitErr := debugger.Wait(); err == nil {
			err = waitErr
		}
	}()
	setup := []string{
		"settings set auto-confirm true",
		"settings set symbols.enable-external-lookup false",
		`settings set thread-format "layoutlint-thread ${thread.index}|{${frame.pc}}|{${thread.stop-reason}}\n"`,
		`breakpoint set --func-regex '\.layoutlint(Checkpoint|PrivateCheckpoint[0-9]+)$' --skip-prologue false`,
	}
	if runtime.GOOS == "linux" {
		// Boehm suspends/resumes threads with SIGPWR/SIGXCPU; LLGo uses
		// SIGURG for thread snapshots. Deliver these runtime handshakes.
		setup = append(setup, "process handle --pass true --stop false --notify false SIGPWR SIGXCPU SIGURG")
	}
	// command source with stop-on-continue=false runs synchronously. Let
	// LLDB consume the loader's initial stop (dyld SIGSTOP on macOS, for
	// example) and resume to the first checkpoint before reading arguments.
	setup = append(setup, fmt.Sprintf("command source --stop-on-continue false --stop-on-error true %q", launchPath))
	for _, command := range setup {
		if _, err := session.run(command); err != nil {
			return err
		}
	}
	fmt.Printf("  LLDB: %d field accesses\n", len(results))
	register := regexp.MustCompile(`(?m)^\s*[a-zA-Z][a-zA-Z0-9]*\s*=\s*(0x[0-9a-fA-F]+)(?:\s|$)`)
	stopReason := regexp.MustCompile(`^breakpoint (1\.[0-9]+)$`)
	breakpointLocation := regexp.MustCompile(`(?m)^\s*(1\.[0-9]+):.* address = (0x[0-9a-fA-F]+), resolved, hit count = ([0-9]+)\b`)
	readArguments := func() ([3]uint64, error) {
		var arguments [3]uint64
		// A synchronous continue can return on a signal or exception. In
		// particular, libc's x2 at a GC handshake is not a checkpoint ID.
		threads, err := session.run("thread list")
		if err != nil {
			return arguments, err
		}
		var thread, location string
		var pc uint64
		selected := false
		for _, line := range strings.Split(threads, "\n") {
			line = strings.TrimSpace(line)
			current := strings.HasPrefix(line, "* ")
			line = strings.TrimPrefix(line, "* ")
			line, ok := strings.CutPrefix(line, "layoutlint-thread ")
			if !ok {
				continue
			}
			parts := strings.SplitN(line, "|", 3)
			if len(parts) != 3 {
				return arguments, fmt.Errorf("invalid LLDB thread status %q", line)
			}
			if parts[2] == "" {
				continue
			}
			reason := stopReason.FindStringSubmatch(parts[2])
			if reason == nil {
				return arguments, fmt.Errorf("unexpected LLDB stop: thread %s at %s: %s", parts[0], parts[1], parts[2])
			}
			if thread != "" {
				return arguments, fmt.Errorf("multiple threads stopped at layout checkpoints; see lldb.log")
			}
			if _, err := strconv.ParseUint(parts[0], 10, 32); err != nil {
				return arguments, fmt.Errorf("invalid LLDB thread index %q", parts[0])
			}
			pc, err = strconv.ParseUint(parts[1], 0, 64)
			if err != nil {
				return arguments, fmt.Errorf("invalid LLDB checkpoint address %q", parts[1])
			}
			thread, location, selected = parts[0], reason[1], current
		}
		if thread == "" {
			return arguments, fmt.Errorf("LLDB did not stop at a layout checkpoint: %s", strings.TrimSpace(threads))
		}
		locations, err := session.run("breakpoint list 1")
		if err != nil {
			return arguments, err
		}
		matched := false
		for _, entry := range breakpointLocation.FindAllStringSubmatch(locations, -1) {
			if entry[1] != location {
				continue
			}
			address, err := strconv.ParseUint(entry[2], 0, 64)
			if err != nil {
				return arguments, err
			}
			matched = address == pc && entry[3] != "0"
			break
		}
		if !matched {
			return arguments, fmt.Errorf("thread %s PC 0x%x does not match hit checkpoint %s; see lldb.log", thread, pc, location)
		}
		if !selected {
			if _, err := session.run("thread select " + thread); err != nil {
				return arguments, err
			}
		}
		if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
			// Both MSVC and MinGW pass these three words on the stack. The
			// breakpoint skips no prologue, so ESP still points to the return
			// address; base, address and id are at ESP+4, ESP+8 and ESP+12.
			output, err := session.run("register read -f hex esp")
			if err != nil {
				return arguments, err
			}
			values := register.FindAllStringSubmatch(output, -1)
			if len(values) != 1 {
				return arguments, fmt.Errorf("LLDB returned %d stack-pointer values, want 1; see lldb.log", len(values))
			}
			stack, err := strconv.ParseUint(values[0][1], 0, 32)
			if err != nil {
				return arguments, err
			}
			path := filepath.Join(directory, "checkpoint-arguments.bin")
			if _, err := session.run(fmt.Sprintf("memory read --binary --force --size 1 --count 12 --outfile %q 0x%x", path, stack+4)); err != nil {
				return arguments, err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return arguments, err
			}
			if len(data) != 12 {
				return arguments, fmt.Errorf("LLDB returned %d argument bytes, want 12", len(data))
			}
			for i := range arguments {
				arguments[i] = uint64(binary.LittleEndian.Uint32(data[4*i:]))
			}
			return arguments, nil
		}
		output, err := session.run("register read -f hex arg1 arg2 arg3")
		if err != nil {
			return arguments, err
		}
		values := register.FindAllStringSubmatch(output, -1)
		if len(values) != len(arguments) {
			return arguments, fmt.Errorf("LLDB returned %d register values, want 3; see lldb.log", len(values))
		}
		for j := range arguments {
			value, err := strconv.ParseUint(values[j][1], 0, 64)
			if err != nil {
				return arguments, err
			}
			arguments[j] = value
		}
		return arguments, nil
	}
	checkpoint := func(want uint64) ([3]uint64, error) {
		arguments, err := readArguments()
		if err != nil {
			return arguments, err
		}
		if arguments[2] != want {
			return arguments, fmt.Errorf("unexpected access ID %d, want %d", arguments[2], want)
		}
		return arguments, nil
	}
	failed, err := observePrivateLayouts(&session, private, readArguments)
	if err != nil {
		return err
	}
	fmt.Printf("  LLDB: %d retained type descriptors\n", len(metadata))
	for i := range metadata {
		arguments, err := checkpoint(uint64(2*len(checks) + i))
		if err != nil {
			return fmt.Errorf("metadata %s: %w", metadata[i].name, err)
		}
		if arguments[1] != 0 {
			return fmt.Errorf("metadata %s: unexpected checkpoint payload 0x%x", metadata[i].name, arguments[1])
		}
		result := &metadataResults[i]
		readErr := readMetadata(&session, directory, i, arguments[0], metadata[i].typ, metadataFields[i], result)
		for j := range results {
			observation := &results[j]
			if checks[observation.checkIndex].owner == i {
				if offset, ok := result.fieldOffsets[observation.Field]; ok {
					observation.MetadataOffset = &offset
				}
			}
		}
		for j := range nestedResults {
			if nested[j].owner == i {
				if offset, ok := result.fieldOffsets[nestedResults[j].Field]; ok {
					nestedResults[j].MetadataOffset = &offset
				}
			}
		}
		// Only storage facts come from the closure-adjusted model. Keep the
		// original Go Kind and pointer header, and never mutate raw evidence.
		expectedType := *result.Go.TypeHeader
		expectedType.Size = result.Go.Layout.Size
		expectedType.PtrBytes = result.Go.Layout.PtrBytes
		expectedType.Align = result.Go.Layout.Align
		expectedType.FieldAlign = result.Go.Layout.FieldAlign
		for _, header := range []struct {
			descriptor string
			typeName   string
			expected   *goMetadataHeader
			actual     *metadataHeader
		}{
			{"pointer", "*" + result.Type, result.Go.PointerHeader, result.PointerHeader},
			{"type", result.Type, &expectedType, result.TypeHeader},
		} {
			if header.actual == nil {
				// A failed read may leave only the pointer header available.
				// Report its differences before returning the capture error.
				continue
			}
			for _, field := range []struct {
				name             string
				expected, actual uint64
			}{
				{"Size_", header.expected.Size, header.actual.Size},
				{"PtrBytes", header.expected.PtrBytes, header.actual.PtrBytes},
				{"Align_", uint64(header.expected.Align), uint64(header.actual.Align)},
				{"FieldAlign_", uint64(header.expected.FieldAlign), uint64(header.actual.FieldAlign)},
				// LLGo carries DirectIface/GC flags above the low five Kind bits.
				{"Kind", uint64(header.expected.Kind & 31), uint64(header.actual.Kind & 31)},
			} {
				if field.actual != field.expected {
					failed = true
					result.Differences = append(result.Differences, metadataDifference{header.descriptor, field.name, field.expected, field.actual})
					fmt.Printf("FAIL %s: metadata.%s: expected %d, got %d\n", header.typeName, field.name, field.expected, field.actual)
				}
			}
		}
		for index, field := range result.Fields {
			if field.Observed && field.Actual != field.Expected {
				failed = true
				name := fmt.Sprintf("Fields[%d](%s).Offset", index, field.Name)
				result.Differences = append(result.Differences, metadataDifference{"type", name, field.Expected, field.Actual})
				fmt.Printf("FAIL %s: metadata.%s: expected %d, got %d\n", result.Type, name, field.Expected, field.Actual)
			}
		}
		if readErr != nil {
			return fmt.Errorf("metadata %s: %w", metadata[i].name, readErr)
		}
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return err
		}
	}
	fmt.Printf("  LLDB: %d array strides\n", len(metadata))
	for i, check := range metadata {
		arguments, err := checkpoint(uint64(2*len(checks) + len(metadata) + i))
		if err != nil {
			return fmt.Errorf("array stride %s: %w", check.name, err)
		}
		stride := &strideResult{
			FirstAddress: fmt.Sprintf("0x%x", arguments[0]), SecondAddress: fmt.Sprintf("0x%x", arguments[1]),
		}
		metadataResults[i].Stride = stride
		if arguments[1] < arguments[0] {
			return fmt.Errorf("%s: array element 1 at %s precedes element 0 at %s", check.name, stride.SecondAddress, stride.FirstAddress)
		}
		stride.Bytes = arguments[1] - arguments[0]
		stride.GoMatches = stride.Bytes == goMetadata[i].Layout.Size
		stride.MetadataMatches = stride.Bytes == metadataResults[i].TypeHeader.Size
		if !stride.GoMatches {
			failed = true
			fmt.Printf("FAIL %s: array stride %d, Go layout size %d\n", check.name, stride.Bytes, goMetadata[i].Layout.Size)
		}
		if !stride.MetadataMatches {
			failed = true
			fmt.Printf("FAIL %s: LLGo metadata size %d disagrees with array stride %d\n", check.name, metadataResults[i].TypeHeader.Size, stride.Bytes)
		}
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return err
		}
	}
	fmt.Printf("  LLDB: %d field type sizes\n", len(sizes))
	for i, check := range sizes {
		arguments, err := checkpoint(uint64(2*len(checks) + 2*len(metadata) + len(nested) + i))
		if err != nil {
			return fmt.Errorf("field size %s: %w", check.typ, err)
		}
		result := &sizeResults[i]
		result.FirstAddress, result.SecondAddress = fmt.Sprintf("0x%x", arguments[0]), fmt.Sprintf("0x%x", arguments[1])
		if arguments[1] < arguments[0] {
			return fmt.Errorf("field size %s: element 1 precedes element 0", check.typ)
		}
		result.Actual = arguments[1] - arguments[0]
		result.Observed = true
		if result.Actual != result.Expected {
			failed = true
			fmt.Printf("FAIL field size %s: LLGo %d, Go expects %d\n", check.typ, result.Actual, result.Expected)
		}
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return err
		}
	}
	if len(nested) != 0 {
		fmt.Printf("  LLDB: %d nested layout observations\n", len(nested))
	}
	for i, check := range nested {
		arguments, err := checkpoint(uint64(2*len(checks) + 2*len(metadata) + i))
		if err != nil {
			return err
		}
		result := &nestedResults[i]
		result.Actual = int64(arguments[1]) - int64(arguments[0])
		result.Observed = true
		result.Alignment = &alignmentResult{
			ObjectRequired: check.baseAlign, FieldRequired: check.alignment,
			ObjectAddress: fmt.Sprintf("0x%x", arguments[0]), FieldAddress: fmt.Sprintf("0x%x", arguments[1]),
			ObjectMatches: arguments[0]%uint64(check.baseAlign) == 0,
			FieldMatches:  arguments[1]%uint64(check.alignment) == 0,
		}
		if result.Actual != result.Expected {
			failed = true
			fmt.Printf("FAIL %s.%s: nested layout offset %d, Go expects %d\n", result.Type, result.Field, result.Actual, result.Expected)
		}
		if check.path[len(check.path)-1].field != nil {
			if result.MetadataOffset == nil {
				return fmt.Errorf("%s.%s: metadata field offset was not observed", result.Type, result.Field)
			}
			if *result.MetadataOffset != uint64(result.Expected) || *result.MetadataOffset != uint64(result.Actual) {
				failed = true
				fmt.Printf("FAIL %s.%s: Go offset %d, LLGo offset %d, metadata offset %d\n", result.Type, result.Field, result.Expected, result.Actual, *result.MetadataOffset)
			}
		}
		if !result.Alignment.ObjectMatches || !result.Alignment.FieldMatches {
			return fmt.Errorf("%s.%s: parent address %s requires alignment %d, selected address %s requires alignment %d", result.Type, result.Field, result.Alignment.ObjectAddress, check.baseAlign, result.Alignment.FieldAddress, check.alignment)
		}
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return err
		}
	}
	for observation := range results {
		result := &results[observation]
		i := result.checkIndex
		check := checks[i]
		layout := layouts[i]
		arguments, err := checkpoint(uint64(2 * i))
		if err != nil {
			return err
		}
		result.Actual = int64(arguments[1]) - int64(arguments[0])
		result.Observed = true
		result.Alignment = &alignmentResult{
			ObjectRequired: layout.objectAlign, FieldRequired: layout.fieldAlign,
			ObjectAddress: fmt.Sprintf("0x%x", arguments[0]), FieldAddress: fmt.Sprintf("0x%x", arguments[1]),
			ObjectMatches: arguments[0]%uint64(layout.objectAlign) == 0,
			FieldMatches:  arguments[1]%uint64(layout.fieldAlign) == 0,
		}
		if result.Actual != result.Expected {
			failed = true
			fmt.Printf("FAIL %s.%s %s address case %d: offset %d, Go expects %d\n", result.Type, result.Field, result.Operation, result.AddressCase, result.Actual, result.Expected)
		}
		if result.MetadataOffset == nil {
			return fmt.Errorf("%s.%s: metadata field offset was not observed", result.Type, result.Field)
		}
		if *result.MetadataOffset != uint64(result.Expected) || *result.MetadataOffset != uint64(result.Actual) {
			failed = true
			fmt.Printf("FAIL %s.%s %s address case %d: Go offset %d, LLGo offset %d, metadata offset %d\n", result.Type, result.Field, result.Operation, result.AddressCase, result.Expected, result.Actual, *result.MetadataOffset)
		}
		if !result.Alignment.ObjectMatches || !result.Alignment.FieldMatches {
			return fmt.Errorf("%s.%s %s address case %d: object %s requires alignment %d, field %s requires alignment %d; stopped before access", check.typeName, check.field, check.operation, result.AddressCase, result.Alignment.ObjectAddress, layout.objectAlign, result.Alignment.FieldAddress, layout.fieldAlign)
		}
		if check.kind == types.Invalid {
			if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
				return err
			}
			continue
		}
		storageBytes := (layout.size + 2*accessGuard + 7) / 8 * 8
		base := arguments[0]
		if remainder := base % uint64(2*layout.objectAlign); remainder != uint64(result.AddressCase)*uint64(layout.objectAlign) {
			return fmt.Errorf("%s.%s address case %d: base address modulo %d is %d, want %d", check.typeName, check.field, result.AddressCase, 2*layout.objectAlign, remainder, int64(result.AddressCase)*layout.objectAlign)
		}
		if base < accessGuard || arguments[1] < base-accessGuard || arguments[1]-(base-accessGuard) > uint64(storageBytes-layout.width) {
			return fmt.Errorf("%s.%s %s: field offset %d plus width %d is outside the buffer [%d,%d); stopped before access", check.typeName, check.field, check.operation, result.Actual, layout.width, -accessGuard, storageBytes-accessGuard)
		}
		_, expected, err := scalarPattern(check.kind, layout.width)
		if err != nil {
			return err
		}
		result.Access = &accessResult{ExpectedValue: fmt.Sprintf("%x", expected)}
		seed := make([]byte, storageBytes)
		for j := range seed {
			seed[j] = byte(0xa5 ^ (j * 37))
		}
		for j, b := range expected {
			if check.operation == "write" {
				b = ^b
			}
			seed[accessGuard+layout.offset+int64(j)] = b
		}
		prefix := filepath.Join(directory, fmt.Sprintf("access-%d", observation))
		if err := os.WriteFile(prefix+"-seed.bin", seed, 0644); err != nil {
			return err
		}
		if _, err := session.run(fmt.Sprintf("memory write --infile %q 0x%x", prefix+"-seed.bin", base-accessGuard)); err != nil {
			return err
		}
		if _, err := session.run(fmt.Sprintf("memory read --binary --force --size 1 --count %d --outfile %q 0x%x", len(seed), prefix+"-before.bin", base-accessGuard)); err != nil {
			return err
		}
		before, err := os.ReadFile(prefix + "-before.bin")
		if err != nil {
			return err
		}
		if !bytes.Equal(seed, before) {
			return fmt.Errorf("LLDB did not seed access %d with the requested bytes", i)
		}
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return err
		}
		arguments, err = checkpoint(uint64(2*i + 1))
		if err != nil {
			return err
		}
		if arguments[0] != base {
			return fmt.Errorf("access %d changed its object address", i)
		}
		if check.operation == "read" {
			if _, err := session.run(fmt.Sprintf("memory read --binary --force --size 1 --count %d --outfile %q 0x%x", layout.width, prefix+"-value.bin", arguments[1])); err != nil {
				return err
			}
		}
		if _, err := session.run(fmt.Sprintf("memory read --binary --force --size 1 --count %d --outfile %q 0x%x", len(seed), prefix+"-after.bin", base-accessGuard)); err != nil {
			return err
		}
		after, err := os.ReadFile(prefix + "-after.bin")
		if err != nil {
			return err
		}
		if len(after) != len(seed) {
			return fmt.Errorf("access %d: got %d memory bytes, want %d", i, len(after), len(seed))
		}
		actual := after[accessGuard+layout.offset : accessGuard+layout.offset+layout.width]
		if check.operation == "read" {
			actual, err = os.ReadFile(prefix + "-value.bin")
			if err != nil {
				return err
			}
		} else {
			copy(seed[accessGuard+layout.offset:], expected)
		}
		access := result.Access
		access.ActualValue = fmt.Sprintf("%x", actual)
		access.ValueMatches = bytes.Equal(expected, actual)
		access.MemoryMatches = bytes.Equal(seed, after)
		for j, b := range after {
			if b != seed[j] {
				access.ChangedBytes = append(access.ChangedBytes, j-accessGuard)
			}
		}
		if !access.ValueMatches || !access.MemoryMatches {
			failed = true
			fmt.Printf("FAIL %s.%s %s address case %d: value %x, want %x; memory matches: %t\n", check.typeName, check.field, check.operation, result.AddressCase, actual, expected, access.MemoryMatches)
		}
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return err
		}
	}
	status, err := session.run("process status")
	if err != nil {
		return err
	}
	if !strings.Contains(status, "exited with status = 0 (0x00000000)") {
		return fmt.Errorf("LLDB did not report a successful process exit; see lldb.log")
	}
	if failed {
		return fmt.Errorf("field offset/size, access, nested layout, array stride or ABI metadata checks failed; see %s", filepath.Join(directory, "offsets.json"))
	}
	metadataFieldsChecked, privateFieldsChecked := 0, 0
	for _, result := range metadataResults {
		metadataFieldsChecked += len(result.Fields)
	}
	for _, plan := range private {
		metadataFieldsChecked += len(plan.Metadata.Fields)
		privateFieldsChecked += len(plan.Fields)
	}
	fmt.Printf("PASS %s: %d concrete struct types, %d field accesses, %d array strides, %d nested layout observations, %d owner-package field observations, %d metadata field offsets, %d checked type headers\n", pkg.PkgPath, len(metadataResults)+len(private), len(results), len(metadataResults)+len(private), len(nestedResults), privateFieldsChecked, metadataFieldsChecked, 2*(len(metadataResults)+len(private)))
	return nil
}

// scalarPattern builds a Go constant and its native bytes from the same value.
// No LLGo value, size, reflection result or target debugger type is involved.
func scalarPattern(kind types.BasicKind, width int64) (*gogen.Element, []byte, error) {
	var literal string
	var value constant.Value
	var untyped types.BasicKind
	var data []byte
	switch kind {
	case types.Bool:
		literal, value, untyped = "true", constant.MakeBool(true), types.UntypedBool
		data = []byte{1}
	case types.Float32, types.Float64, types.Complex64, types.Complex128:
		partWidth := width
		complex := kind == types.Complex64 || kind == types.Complex128
		if complex {
			partWidth /= 2
		}
		var numbers []float64
		count := 1
		if complex {
			count = 2
		}
		for i := 0; i < count; i++ {
			if partWidth == 4 {
				bits := uint32(0xc12a4b6d) ^ uint32(i)*0x80123456
				numbers = append(numbers, float64(math.Float32frombits(bits)))
				data = binary.NativeEndian.AppendUint32(data, bits)
			} else if partWidth == 8 {
				bits := uint64(0xc02468ace13579bd) ^ uint64(i)*0x800123456789abcd
				numbers = append(numbers, math.Float64frombits(bits))
				data = binary.NativeEndian.AppendUint64(data, bits)
			} else {
				return nil, nil, fmt.Errorf("unsupported Go floating-point width %d", width)
			}
		}
		literal = strconv.FormatFloat(numbers[0], 'g', -1, 64)
		value, untyped = constant.MakeFloat64(numbers[0]), types.UntypedFloat
		if complex {
			literal = "(" + literal + "+" + strconv.FormatFloat(numbers[1], 'g', -1, 64) + "i)"
			value = constant.BinaryOp(value, token.ADD, constant.MakeImag(constant.MakeFloat64(numbers[1])))
			untyped = types.UntypedComplex
		}
	default:
		bits := uint64(0x91b7d35f6a2c4e89)
		switch width {
		case 1:
			bits &= 0xff
			data = []byte{byte(bits)}
		case 2:
			bits &= 0xffff
			data = binary.NativeEndian.AppendUint16(data, uint16(bits))
		case 4:
			bits &= 0xffffffff
			data = binary.NativeEndian.AppendUint32(data, uint32(bits))
		case 8:
			data = binary.NativeEndian.AppendUint64(data, bits)
		default:
			return nil, nil, fmt.Errorf("unsupported Go integer width %d", width)
		}
		untyped = types.UntypedInt
		if types.Typ[kind].Info()&types.IsUnsigned != 0 {
			literal, value = strconv.FormatUint(bits, 10), constant.MakeUint64(bits)
		} else {
			shift := uint(64 - 8*width)
			signed := int64(bits<<shift) >> shift
			literal, value = strconv.FormatInt(signed, 10), constant.MakeInt64(signed)
		}
	}
	if int64(len(data)) != width {
		return nil, nil, fmt.Errorf("Go scalar width %d disagrees with %d pattern bytes", width, len(data))
	}
	expression, err := parser.ParseExpr(literal)
	if err != nil {
		return nil, nil, err
	}
	return &gogen.Element{Val: expression, Type: types.Typ[untyped], CVal: value}, data, nil
}
