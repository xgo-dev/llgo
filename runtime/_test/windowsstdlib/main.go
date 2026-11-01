package main

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

const LLGoFiles = "_wrap/syscall.c; _wrap/syscall_dynamic.c; _wrap/syscall_registers.S"

//go:linkname cSum12 C.llgo_windows_sum12
func cSum12(a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12 uintptr) uintptr

//go:linkname cAddDouble C.llgo_windows_add_double
func cAddDouble(a, b float64) float64

func testWindowsSyscalls() {
	sum12 := uintptr(c.Func(cSum12))
	wantSum := uintptr(78)
	r1, _, errNo := syscall.SyscallN(sum12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	if r1 != wantSum || errNo != 0 {
		panic("SyscallN did not pass 12 C ABI arguments")
	}
	var maxArgs [42]uintptr
	for i := 0; i < 12; i++ {
		maxArgs[i] = uintptr(i + 1)
	}
	r1, _, errNo = syscall.SyscallN(sum12, maxArgs[:]...)
	if r1 != wantSum || errNo != 0 {
		panic("SyscallN did not preserve its 42-argument stack frame")
	}
	r1, _, errNo = syscall.Syscall12(sum12, 12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	if r1 != wantSum || errNo != 0 {
		panic("Syscall12 did not pass 12 C ABI arguments")
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	pid, _, _ := kernel32.NewProc("GetCurrentProcessId").Call()
	if pid == 0 {
		panic("zero-argument SyscallN returned an invalid process id")
	}
	if err := kernel32.NewProc("llgo_missing_windows_procedure").Find(); err == nil {
		panic("GetProcAddress failure lost its error")
	}
	if _, err := syscall.LoadDLL("llgo_missing_windows_library.dll"); err == nil {
		panic("LoadLibraryExW failure lost its error")
	}
	setLastError := kernel32.NewProc("SetLastError")
	const wantErr = syscall.Errno(0x4d2)
	_, _, err := setLastError.Call(uintptr(wantErr))
	if err != wantErr {
		panic("SyscallN did not preserve GetLastError")
	}
	if runtime.GOARCH == "amd64" {
		addDouble := uintptr(c.Func(cAddDouble))
		_, bits, _ := syscall.SyscallN(
			addDouble,
			uintptr(math.Float64bits(1.25)),
			uintptr(math.Float64bits(2.5)),
		)
		if got := math.Float64frombits(uint64(bits)); got != 3.75 {
			panic("SyscallN did not preserve amd64 floating-point registers")
		}
	}

	var wsa syscall.WSAData
	if err := syscall.WSAStartup(0x202, &wsa); err != nil {
		panic("WSAStartup failed")
	}
	defer syscall.WSACleanup()
	var received uint32
	var flags uint32
	var fromLen int32
	if err := syscall.WSARecvFrom(
		syscall.InvalidHandle, nil, 0, &received, &flags,
		nil, &fromLen, nil, nil,
	); err == nil {
		panic("nine-argument WSARecvFrom unexpectedly succeeded")
	}

	const envKey = "LLGO_WINDOWS_STDLIB_SMOKE"
	if err := syscall.Setenv(envKey, "ok"); err != nil {
		panic("Setenv failed")
	}
	if value, ok := syscall.Getenv(envKey); !ok || value != "ok" {
		panic("Getenv did not observe Setenv")
	}
	if err := syscall.Unsetenv(envKey); err != nil {
		panic("Unsetenv failed")
	}
}

func testWindowsOS() {
	// Windows initializes os.Args from GetCommandLine and commandLineToArgv.
	if len(os.Args) == 0 {
		panic("os.Args is empty")
	}
	if os.Args[0] == "" {
		panic("os.Args[0] is empty")
	}

	if pid := os.Getpid(); pid <= 0 {
		panic("os.Getpid returned invalid pid")
	}

	exePath, err := os.Executable()
	if err != nil {
		panic("os.Executable failed: " + err.Error())
	}
	if exePath == "" {
		panic("os.Executable returned empty path")
	}

	origWd, err := os.Getwd()
	if err != nil {
		panic("os.Getwd failed: " + err.Error())
	}
	if origWd == "" {
		panic("os.Getwd returned empty path")
	}

	fi, err := os.Stat(exePath)
	if err != nil {
		panic("os.Stat(executable) failed: " + err.Error())
	}
	if fi.IsDir() {
		panic("os.Stat(executable) reported a directory")
	}
	if fi.Size() == 0 {
		panic("os.Stat(executable) reported zero size")
	}

	testDir, err := os.MkdirTemp("", "llgo-windows-stdlib-")
	if err != nil {
		panic("os.MkdirTemp failed: " + err.Error())
	}
	defer os.RemoveAll(testDir)

	nestedDir := filepath.Join(testDir, "sub", "dir")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		panic("os.MkdirAll failed: " + err.Error())
	}

	di, err := os.Stat(nestedDir)
	if err != nil {
		panic("os.Stat(nestedDir) failed: " + err.Error())
	}
	if !di.IsDir() {
		panic("os.Stat(nestedDir) is not a directory")
	}

	testFile := filepath.Join(testDir, "hello.txt")
	testContent := []byte("Hello from LLGo Windows stdlib test!")
	if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
		panic("os.WriteFile failed: " + err.Error())
	}
	got, err := os.ReadFile(testFile)
	if err != nil {
		panic("os.ReadFile failed: " + err.Error())
	}
	if string(got) != string(testContent) {
		panic("os.ReadFile content mismatch")
	}

	// Until IOCP lands, an overlapped file must use internal/poll's event
	// fallback after runtime_pollOpen reports that it is unsupported.
	const windowsFileFlagOverlapped = 0x40000000
	overlappedPath := filepath.Join(testDir, "overlapped.txt")
	overlapped, err := os.OpenFile(overlappedPath,
		os.O_CREATE|os.O_RDWR|windowsFileFlagOverlapped, 0o644)
	if err != nil {
		panic("os.OpenFile(overlapped) failed: " + err.Error())
	}
	if _, err := overlapped.Write(testContent); err != nil {
		panic("overlapped Write failed: " + err.Error())
	}
	if _, err := overlapped.Seek(0, 0); err != nil {
		panic("overlapped Seek failed: " + err.Error())
	}
	overlappedContent := make([]byte, len(testContent))
	if _, err := io.ReadFull(overlapped, overlappedContent); err != nil {
		panic("overlapped Read failed: " + err.Error())
	}
	if err := overlapped.Close(); err != nil {
		panic("overlapped Close failed: " + err.Error())
	}
	if string(overlappedContent) != string(testContent) {
		panic("overlapped file content mismatch")
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		panic("os.ReadDir failed: " + err.Error())
	}
	if len(entries) == 0 {
		panic("os.ReadDir returned zero entries")
	}

	if err := os.Chdir(testDir); err != nil {
		panic("os.Chdir failed: " + err.Error())
	}
	newWd, err := os.Getwd()
	if err != nil {
		panic("os.Getwd after Chdir failed: " + err.Error())
	}
	if newWd == "" {
		panic("os.Getwd after Chdir returned empty")
	}
	if err := os.Chdir(origWd); err != nil {
		panic("os.Chdir(origWd) failed: " + err.Error())
	}

	env := os.Environ()
	if len(env) == 0 {
		panic("os.Environ returned empty")
	}

	const envKey = "LLGO_STDLIB_OS_TEST"
	if err := os.Setenv(envKey, "ok"); err != nil {
		panic("os.Setenv failed: " + err.Error())
	}
	if v := os.Getenv(envKey); v != "ok" {
		panic("os.Getenv did not observe Setenv")
	}
	if err := os.Unsetenv(envKey); err != nil {
		panic("os.Unsetenv failed: " + err.Error())
	}

	abs, err := filepath.Abs(".")
	if err != nil {
		panic("filepath.Abs failed: " + err.Error())
	}
	if abs == "" || abs == "." {
		panic("filepath.Abs returned invalid result")
	}

	pagesize := syscall.Getpagesize()
	if pagesize < 4096 || pagesize&(pagesize-1) != 0 {
		panic("syscall.Getpagesize returned invalid value")
	}

	utf16Slice := []uint16{'H', 'e', 'l', 'l', 'o', ' ', 'W', 'i', 'n', 'd', 'o', 'w', 's', 0}
	str := syscall.UTF16ToString(utf16Slice)
	if str != "Hello Windows" {
		panic("syscall.UTF16ToString failed: " + str)
	}

	sPtr, err := syscall.UTF16PtrFromString("TestPath\\SubDir")
	if err != nil || sPtr == nil {
		panic("syscall.UTF16PtrFromString failed")
	}

	// Verify that the filesystem path bridge preserves UTF-16 names.
	uniDir := filepath.Join(testDir, "日本語テスト")
	if err := os.Mkdir(uniDir, 0o755); err != nil {
		panic("os.Mkdir(unicode) failed: " + err.Error())
	}
	ufi, err := os.Stat(uniDir)
	if err != nil {
		panic("os.Stat(unicode dir) failed: " + err.Error())
	}
	if !ufi.IsDir() {
		panic("os.Stat(unicode dir) is not a directory")
	}
}

func main() {
	if os.Getenv("LLGO_TEST_OS_EXIT") == "1" {
		os.Exit(23)
	}

	var once sync.Once
	done := make(chan struct{}, 4)
	value := 0
	for i := 0; i < 4; i++ {
		go func() {
			once.Do(func() { value = 42 })
			done <- struct{}{}
		}()
	}
	for i := 0; i < 4; i++ {
		<-done
	}
	if value != 42 {
		panic("sync.Once ran incorrectly")
	}
	testWindowsSyscalls()
	testDynamicWindowsSyscalls()
	testWindowsOS()

	start := time.Now()
	time.Sleep(time.Millisecond)
	if time.Since(start) < 0 {
		panic("monotonic clock moved backwards")
	}
	println("windows stdlib smoke: ok")
}
