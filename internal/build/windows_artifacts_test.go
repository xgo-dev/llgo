package build

import (
	"debug/pe"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestWindowsNativeArtifacts(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native PE/COFF artifact test")
	}
	t.Run("executable", testWindowsExecutableArtifact)
	t.Run("c-archive", testWindowsCArchiveArtifact)
	t.Run("c-shared", testWindowsCSharedArtifact)
}

func TestWindowsConsumesMSVCLibrary(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native MSVC interoperability test")
	}
	cl, clErr := exec.LookPath("cl.exe")
	if clErr != nil {
		if os.Getenv("LLGO_REQUIRE_MSVC") == "1" {
			t.Fatalf("Visual Studio cl.exe is required: %v", clErr)
		}
		t.Skip("Visual Studio C++ tools are not installed")
	}
	lib := filepath.Join(filepath.Dir(cl), "lib.exe")
	link := filepath.Join(filepath.Dir(cl), "link.exe")
	_, libErr := os.Stat(lib)
	_, linkErr := os.Stat(link)
	if libErr != nil || linkErr != nil {
		if os.Getenv("LLGO_REQUIRE_MSVC") == "1" {
			t.Fatalf("Visual Studio lib.exe/link.exe are required next to %s: lib=%v, link=%v", cl, libErr, linkErr)
		}
		t.Skip("Visual Studio C++ tools are not installed")
	}

	t.Run("native CRT hello", testWindowsCRTHello)
	t.Run("consume cl static library", func(t *testing.T) {
		testWindowsConsumesCLLibrary(t, cl, lib)
	})
	t.Run("link consumes LLGo archive", func(t *testing.T) {
		testMSVCConsumesWindowsArchive(t, cl, link)
	})
	t.Run("link consumes LLGo DLL", func(t *testing.T) {
		testMSVCConsumesWindowsDLL(t, cl, link)
	})
}

func testWindowsCRTHello(t *testing.T) {
	dir := t.TempDir()
	ctx := newWindowsArtifactContext(t, BuildModeExe)
	object := compileWindowsArtifact(t, ctx, dir, "hello.c", `
int puts(const char *);
int main(void) { return puts("hello from native COFF") < 0; }
`)
	executable := filepath.Join(dir, "hello.exe")
	if err := linkObjFiles(ctx, executable, []string{object}, nil, false); err != nil {
		t.Fatal(err)
	}
	checkPEArtifact(t, executable, false)
	cmd := exec.Command(executable)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run native CRT hello: %v\n%s", err, output)
	}
	if got := strings.TrimSpace(string(output)); got != "hello from native COFF" {
		t.Fatalf("native CRT hello output = %q", got)
	}
}

func testWindowsConsumesCLLibrary(t *testing.T, cl, lib string) {
	dir := t.TempDir()
	source := writeWindowsArtifactSource(t, dir, "msvc-answer.c", `
int answer_from_msvc(void) { return 42; }
`)
	object := filepath.Join(dir, "msvc-answer.obj")
	archive := filepath.Join(dir, "msvc-answer.lib")
	runTool(t, cl, "/nologo", "/TC", "/c", source, "/Fo"+object)
	runTool(t, lib, "/nologo", "/out:"+archive, object)

	ctx := newWindowsArtifactContext(t, BuildModeExe)
	consumer := compileWindowsArtifact(t, ctx, dir, "msvc-consumer.c", `
int answer_from_msvc(void);
int mainCRTStartup(void) { return answer_from_msvc() == 42 ? 0 : 23; }
`)
	executable := filepath.Join(dir, "msvc-consumer.exe")
	linkWindowsExecutable(t, ctx, executable, consumer, archive)
	checkPEArtifact(t, executable, false)
	runTool(t, executable)
}

func testMSVCConsumesWindowsArchive(t *testing.T, cl, link string) {
	dir := t.TempDir()
	ctx := newWindowsArtifactContext(t, BuildModeCArchive)
	object := compileWindowsArtifact(t, ctx, dir, "llgo-static.c", `
int answer_from_llgo_archive(void) { return 42; }
`)
	archive := filepath.Join(dir, "llgo-output.a")
	if err := ctx.createMergedArchiveFile(archive, []string{object}); err != nil {
		t.Fatal(err)
	}

	consumer := compileMSVCArtifact(t, cl, dir, "llgo-archive-consumer.c", `
int answer_from_llgo_archive(void);
int mainCRTStartup(void) { return answer_from_llgo_archive() == 42 ? 0 : 23; }
`)
	executable := filepath.Join(dir, "llgo-archive-consumer.exe")
	linkMSVCExecutable(t, link, executable, consumer, archive)
	checkPEArtifact(t, executable, false)
	runTool(t, executable)
}

func testMSVCConsumesWindowsDLL(t *testing.T, cl, link string) {
	dir := t.TempDir()
	ctx := newWindowsArtifactContext(t, BuildModeCShared)
	object := compileWindowsArtifact(t, ctx, dir, "llgo-shared.c", `
int answer_from_llgo_dll(void) { return 42; }
`)
	dll := filepath.Join(dir, "llgo-output.dll")
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("example.com/msvcfixture", "example.com/msvcfixture")
	lpkg.SetExport("example.com/msvcfixture.answer", "answer_from_llgo_dll")
	linkArgs := append(
		[]string{"-nostdlib", "-Wl,/noentry"},
		cSharedExportArgs(ctx, []*aPackage{{LPkg: lpkg}})...,
	)
	if err := linkObjFiles(ctx, dll, []string{object}, linkArgs, false); err != nil {
		t.Fatal(err)
	}
	imports := strings.TrimSuffix(dll, filepath.Ext(dll)) + ".lib"

	consumer := compileMSVCArtifact(t, cl, dir, "llgo-dll-consumer.c", `
__declspec(dllimport) int answer_from_llgo_dll(void);
int mainCRTStartup(void) { return answer_from_llgo_dll() == 42 ? 0 : 23; }
`)
	executable := filepath.Join(dir, "llgo-dll-consumer.exe")
	linkMSVCExecutable(t, link, executable, consumer, imports)
	checkPEArtifact(t, executable, false)
	runTool(t, executable)
}

func testWindowsExecutableArtifact(t *testing.T) {
	dir := t.TempDir()
	ctx := newWindowsArtifactContext(t, BuildModeExe)
	object := compileWindowsArtifact(t, ctx, dir, "entry.c", `
int mainCRTStartup(void) { return 0; }
`)
	// An explicit -o name is exact even without the conventional suffix.
	executable := filepath.Join(dir, "native-exact-output")
	if err := os.WriteFile(executable, []byte("stale output"), 0o666); err != nil {
		t.Fatal(err)
	}
	linkWindowsExecutable(t, ctx, executable, object)
	if _, err := os.Stat(executable + ".exe"); !os.IsNotExist(err) {
		t.Fatalf("Clang's intermediate .exe was not consumed: %v", err)
	}
	checkPEArtifact(t, executable, false)
	// Windows process lookup intentionally requires an executable suffix even
	// though CreateProcess accepts PE content under an exact extensionless -o
	// name. Build a conventional sibling to exercise execution separately.
	runnable := filepath.Join(dir, "native-runnable.exe")
	linkWindowsExecutable(t, ctx, runnable, object)
	checkPEArtifact(t, runnable, false)
	runTool(t, runnable)
}

func testWindowsCArchiveArtifact(t *testing.T) {
	dir := t.TempDir()
	ctx := newWindowsArtifactContext(t, BuildModeCArchive)
	direct := compileWindowsArtifact(t, ctx, dir, "direct.c", `
int answer_from_archive(void) { return 42; }
`)
	nestedObject := compileWindowsArtifact(t, ctx, dir, "nested.c", `
int nested_archive_value(void) { return 7; }
`)
	nested := filepath.Join(dir, "nested.lib")
	runTool(t, ctx.archiveMergerOrFatal(t), "rcs", nested, nestedObject)

	archive := filepath.Join(dir, "native-output.a")
	if err := ctx.createMergedArchiveFile(archive, []string{direct, nested}); err != nil {
		t.Fatal(err)
	}
	members := archiveMembers(t, ctx.archiveMergerOrFatal(t), archive)
	slices.Sort(members)
	if got, want := strings.Join(members, " "), "direct.o nested.o"; got != want {
		t.Fatalf("flat COFF archive members = %q, want %q", got, want)
	}

	consumerCtx := newWindowsArtifactContext(t, BuildModeExe)
	consumer := compileWindowsArtifact(t, consumerCtx, dir, "archive-consumer.c", `
int answer_from_archive(void);
int mainCRTStartup(void) { return answer_from_archive() == 42 ? 0 : 23; }
`)
	executable := filepath.Join(dir, "archive-consumer.exe")
	linkWindowsExecutable(t, consumerCtx, executable, consumer, archive)
	checkPEArtifact(t, executable, false)
	runTool(t, executable)
}

func testWindowsCSharedArtifact(t *testing.T) {
	dir := t.TempDir()
	ctx := newWindowsArtifactContext(t, BuildModeCShared)
	object := compileWindowsArtifact(t, ctx, dir, "shared.c", `
int answer_from_dll(void) { return 42; }
`)
	dll := filepath.Join(dir, "native-shared.dll")
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("example.com/windowsfixture", "example.com/windowsfixture")
	lpkg.SetExport("example.com/windowsfixture.answer", "answer_from_dll")
	exportArgs := cSharedExportArgs(ctx, []*aPackage{{LPkg: lpkg}})
	linkArgs := append([]string{"-nostdlib", "-Wl,/noentry"}, exportArgs...)
	if err := linkObjFiles(ctx, dll, []string{object}, linkArgs, false); err != nil {
		t.Fatal(err)
	}
	checkPEArtifact(t, dll, true)
	imports := strings.TrimSuffix(dll, filepath.Ext(dll)) + ".lib"
	if _, err := os.Stat(imports); err != nil {
		t.Fatalf("COFF import library was not generated: %v", err)
	}

	consumerCtx := newWindowsArtifactContext(t, BuildModeExe)
	consumer := compileWindowsArtifact(t, consumerCtx, dir, "shared-consumer.c", `
__declspec(dllimport) int answer_from_dll(void);
int mainCRTStartup(void) { return answer_from_dll() == 42 ? 0 : 23; }
`)
	executable := filepath.Join(dir, "shared-consumer.exe")
	linkWindowsExecutable(t, consumerCtx, executable, consumer, imports)
	checkPEArtifact(t, executable, false)
	runTool(t, executable)
}

func newWindowsArtifactContext(t *testing.T, mode BuildMode) *context {
	t.Helper()
	export, err := crosscompile.UseWithGOARM(
		"windows", runtime.GOARCH, "", "", false, false, optlevel.O0, lto.Off, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if export.Toolchain.ObjectFormat != crosscompile.ObjectFormatCOFF {
		t.Fatalf("Windows toolchain object format = %q, want COFF", export.Toolchain.ObjectFormat)
	}
	return &context{
		mode: ModeBuild,
		buildConf: &Config{
			Goos:      "windows",
			Goarch:    runtime.GOARCH,
			Mode:      ModeBuild,
			BuildMode: mode,
			LinkOptions: LinkOptions{
				DWARF: DWARFOmit,
			},
		},
		crossCompile: export,
	}
}

func compileWindowsArtifact(t *testing.T, ctx *context, dir, name, source string) string {
	t.Helper()
	sourcePath := writeWindowsArtifactSource(t, dir, name, source)
	object := filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+".o")
	if err := ctx.compiler().Compile("-x", "c", "-c", "-o", object, sourcePath); err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	return object
}

func writeWindowsArtifactSource(t *testing.T, dir, name, source string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(source)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func compileMSVCArtifact(t *testing.T, cl, dir, name, source string) string {
	t.Helper()
	sourcePath := writeWindowsArtifactSource(t, dir, name, source)
	object := filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+".obj")
	runTool(t, cl, "/nologo", "/TC", "/c", sourcePath, "/Fo"+object)
	return object
}

func linkMSVCExecutable(t *testing.T, link, executable string, inputs ...string) {
	t.Helper()
	args := []string{
		"/nologo",
		"/nodefaultlib",
		"/entry:mainCRTStartup",
		"/subsystem:console",
		"/out:" + executable,
	}
	runTool(t, link, append(args, inputs...)...)
}

func linkWindowsExecutable(t *testing.T, ctx *context, executable string, inputs ...string) {
	t.Helper()
	linkArgs := []string{
		"-nostdlib",
		"-Wl,/entry:mainCRTStartup",
		"-Wl,/subsystem:console",
	}
	if err := linkObjFiles(ctx, executable, inputs, linkArgs, false); err != nil {
		t.Fatal(err)
	}
}

func checkPEArtifact(t *testing.T, path string, wantDLL bool) {
	t.Helper()
	file, err := pe.Open(path)
	if err != nil {
		t.Fatalf("open PE artifact %s: %v", path, err)
	}
	defer file.Close()
	wantMachine := map[string]uint16{
		"386":   pe.IMAGE_FILE_MACHINE_I386,
		"amd64": pe.IMAGE_FILE_MACHINE_AMD64,
		"arm64": pe.IMAGE_FILE_MACHINE_ARM64,
	}[runtime.GOARCH]
	if file.FileHeader.Machine != wantMachine {
		t.Fatalf("PE machine = %#x, want %#x", file.FileHeader.Machine, wantMachine)
	}
	isDLL := file.FileHeader.Characteristics&pe.IMAGE_FILE_DLL != 0
	if isDLL != wantDLL {
		t.Fatalf("PE DLL characteristic = %v, want %v", isDLL, wantDLL)
	}
}

func (c *context) archiveMergerOrFatal(t *testing.T) string {
	t.Helper()
	tool, err := c.archiveMerger()
	if err != nil {
		t.Fatal(err)
	}
	return tool
}

func archiveMembers(t *testing.T, archiver, archive string) []string {
	t.Helper()
	cmd := exec.Command(archiver, "t", archive)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list archive %s: %v\n%s", archive, err, output)
	}
	return strings.Fields(string(output))
}

func runTool(t *testing.T, app string, args ...string) {
	t.Helper()
	cmd := exec.Command(app, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s failed: %v\n%s", formatToolCommand(app, args), err, output)
	}
}

func formatToolCommand(app string, args []string) string {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, fmt.Sprintf("%q", app))
	for _, arg := range args {
		quoted = append(quoted, fmt.Sprintf("%q", arg))
	}
	return strings.Join(quoted, " ")
}
