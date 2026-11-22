// The WASI initializer bodies used for matching come from the Go standard
// library. See runtime/LICENSES/Go-BSD-3-Clause.txt.

package build

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// The native startup reads the same Preview 1 descriptors as these Go
// initializers. If the standard library changes either operation, keep the
// ordinary entry until the replacement can account for it.
func matchesWASIProcessContract(pkg *ssa.Package, preopen *ssa.Function) bool {
	legacyPreopen := strings.NewReplacer(
		"&prestat)", "unsafe.Pointer(&prestat))",
		"&dirNameBuf[0],", "unsafe.Pointer(&dirNameBuf[0]),",
	).Replace(wasiCompactPreopenBody)
	if (!matchesWASIStartupBody(preopen, wasiCompactPreopenBody) && !matchesWASIStartupBody(preopen, legacyPreopen)) ||
		!matchesWASIStartupBody(pkg.Func("SetNonblock"), wasiCompactSetNonblockBody) {
		return false
	}
	flags := pkg.Func("fd_fdstat_get_flags")
	if !matchesWASIStartupBody(flags, wasiCompactGetFlagsBody) &&
		!matchesWASIStartupBody(flags, strings.Replace(wasiCompactGetFlagsBody, "&stat", "unsafe.Pointer(&stat)", 1)) {
		return false
	}
	count := 0
	for _, member := range pkg.Members {
		fn, _ := member.(*ssa.Function)
		if fn == nil || !strings.HasPrefix(fn.Name(), "init#") || fn == preopen {
			continue
		}
		if !matchesWASIStartupBody(fn, wasiCompactStdioBody) {
			return false
		}
		count++
	}
	return count == 1
}

func matchesWASIStartupBody(fn *ssa.Function, expected string) bool {
	if fn == nil {
		return false
	}
	syntax, _ := fn.Syntax().(*ast.FuncDecl)
	if syntax == nil || syntax.Body == nil {
		return false
	}
	return matchesWASIStartupBlock(syntax.Body, expected)
}

func matchesWASIStartupBlock(block *ast.BlockStmt, expected string) bool {
	// Fresh positions ignore source comments and line numbers.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", "package p\nfunc f() "+expected, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	var body, want bytes.Buffer
	match := file.Decls[0].(*ast.FuncDecl).Body
	if format.Node(&body, token.NewFileSet(), block) != nil ||
		format.Node(&want, token.NewFileSet(), match) != nil {
		return false
	}
	return bytes.Equal(body.Bytes(), want.Bytes())
}

const wasiCompactStdioBody = `{
	SetNonblock(0, true)
	SetNonblock(1, true)
	SetNonblock(2, true)
}`

const wasiCompactSetNonblockBody = `{
	flags, err := fd_fdstat_get_flags(fd)
	if err != nil {
		return err
	}
	if nonblocking {
		flags |= FDFLAG_NONBLOCK
	} else {
		flags &^= FDFLAG_NONBLOCK
	}
	errno := fd_fdstat_set_flags(int32(fd), flags)
	return errnoErr(errno)
}`

const wasiCompactGetFlagsBody = `{
	var stat fdstat
	errno := fd_fdstat_get(int32(fd), &stat)
	return uint32(stat.fdflags), errnoErr(errno)
}`

const wasiCompactPreopenBody = `{
	dirNameBuf := make([]byte, 256)
	for preopenFd := int32(3); ; preopenFd++ {
		var prestat prestat
		errno := fd_prestat_get(preopenFd, &prestat)
		if errno == EBADF {
			break
		}
		if errno == ENOTDIR || prestat.typ != preopentypeDir {
			continue
		}
		if errno != 0 {
			panic("fd_prestat: " + errno.Error())
		}
		if int(prestat.dir.prNameLen) > len(dirNameBuf) {
			dirNameBuf = make([]byte, prestat.dir.prNameLen)
		}
		errno = fd_prestat_dir_name(preopenFd, &dirNameBuf[0], prestat.dir.prNameLen)
		if errno != 0 {
			panic("fd_prestat_dir_name: " + errno.Error())
		}
		if ` + wasiPathStateMember + ` { ` + wasiSaveMount + ` }
	}
	if ` + wasiPathStateMember + ` { ` + wasiSaveCwd + ` }
}`

const wasiStartupTestlogGuard = "llgoWASIUseOrdinaryStartup"

func rewriteWASIStartupTestlog(source []byte) []byte {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "log.go", source, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != "testlog" {
		return nil
	}
	var imports token.Pos
	var logger, setLogger *ast.BlockStmt
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				imports = d.End()
			}
		case *ast.FuncDecl:
			if d.Recv == nil {
				switch d.Name.Name {
				case "Logger":
					logger = d.Body
				case "SetLogger":
					setLogger = d.Body
				}
			}
		}
	}
	legacyLogger := strings.Replace(wasiTestlogLoggerBody, "return *impl", "return *impl.(*Interface)", 1)
	if !imports.IsValid() || logger == nil || setLogger == nil ||
		(!matchesWASIStartupBlock(logger, wasiTestlogLoggerBody) && !matchesWASIStartupBlock(logger, legacyLogger)) {
		return nil
	}
	offset := files.File(file.Pos()).Offset
	first, second, third := offset(imports), offset(logger.Lbrace)+1, offset(setLogger.Lbrace)+1
	type edit struct {
		at   int
		text string
	}
	edits := []edit{
		{first, "\nimport _ \"unsafe\""},
		{second, " if !" + wasiStartupTestlogGuard + " { return nil };"},
		{third, " llgoRequireWASIStartup();"},
		{len(source), "\n//go:linkname " + wasiStartupTestlogGuard + " C.__llgo_wasi_ordinary_startup\nvar " + wasiStartupTestlogGuard + " bool\n" +
			"\n//go:linkname llgoRequireWASIStartup C.__llgo_wasi_requires_managed_startup\nfunc llgoRequireWASIStartup()\n"},
	}
	slices.SortFunc(edits, func(a, b edit) int { return a.at - b.at })
	var output bytes.Buffer
	at := 0
	for _, entry := range edits {
		output.Write(source[at:entry.at])
		output.WriteString(entry.text)
		at = entry.at
	}
	return output.Bytes()
}

const wasiTestlogLoggerBody = `{
	impl := logger.Load()
	if impl == nil {
		return nil
	}
	return *impl
}`
