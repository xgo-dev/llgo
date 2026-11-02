// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package build

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

// RuntimeBoundary records ownership, not a proof of native ABI compatibility.
// Native provider sources still require preprocessing, linking and execution
// in the target's normal test matrix. The declaration's signature is retained
// in Symbols; Go linkname aliases remain unsafe and are not type-equality gated.
type RuntimeBoundary struct {
	Symbol   string   `json:"symbol"`
	Target   string   `json:"target"`
	Kind     string   `json:"kind"`
	Provider string   `json:"provider"`
	Sources  []string `json:"sources,omitempty"`
}

func (a *runtimeAuditor) externalBoundaries(report *RuntimeOwnership) error {
	nativeProviders, err := runtimeNativeProviders(runtimeNativeOwners)
	if err != nil {
		return err
	}
	// Resolving a Go alias can add a provider outside the four inventoried
	// runtime packages, such as internal/coverage/rtcov.AddMeta.
	for _, symbol := range append([]RuntimeSymbol(nil), report.Symbols...) {
		if symbol.Owner != "external-boundary" {
			continue
		}
		boundary := RuntimeBoundary{Symbol: symbol.Name, Target: symbol.Linkname}
		if strings.HasPrefix(symbol.Linkname, "C.") || strings.HasPrefix(symbol.Linkname, "stdcall.") {
			provider, ok := nativeProviders[symbol.Linkname]
			if !ok {
				report.UnresolvedBoundaries = append(report.UnresolvedBoundaries, RuntimeUse{symbol.Linkname, symbol.Source, "native-provider"})
				continue
			}
			boundary.Kind, boundary.Provider = "native", provider.owner
			for _, source := range strings.Fields(provider.sources) {
				if _, err := os.Stat(filepath.Join(filepath.Dir(a.runtimeDir), filepath.FromSlash(source))); err != nil {
					return fmt.Errorf("runtime boundary %s provider %s: %w", symbol.Name, source, err)
				}
				boundary.Sources = append(boundary.Sources, "$LLGO/"+source)
			}
		} else {
			provider, err := a.runtimeGoProvider(report, symbol.Linkname)
			if err != nil {
				return err
			}
			if provider == nil {
				report.UnresolvedBoundaries = append(report.UnresolvedBoundaries, RuntimeUse{symbol.Linkname, symbol.Source, "go-provider"})
				continue
			}
			boundary.Kind, boundary.Provider = "go", provider.Name
			boundary.Sources = []string{provider.Source}
		}
		report.ExternalBoundaries = append(report.ExternalBoundaries, boundary)
	}
	return nil
}

func (a *runtimeAuditor) runtimeGoProvider(report *RuntimeOwnership, target string) (*RuntimeSymbol, error) {
	for i := range report.Symbols {
		s := &report.Symbols[i]
		if (s.Name == target || s.Linkname == target) && (s.Body || s.Kind == "assembly-entry" || s.Owner == "compiler-intrinsic") {
			return s, nil
		}
	}
	dot := strings.LastIndexByte(target, '.')
	if dot <= 0 || runtimeAuditOwnedSymbol(target) {
		return nil, nil
	}
	pkg, name := target[:dot], target[dot+1:]
	files, err := a.standardFiles(pkg)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		for _, decl := range file.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != name || fn.Body == nil {
				continue
			}
			copy := *file.file
			copy.Decls = []ast.Decl{fn}
			if err := a.inventory(report, pkg, "standard-library", []runtimeAuditFile{{file.path, &copy}}); err != nil {
				return nil, err
			}
			return &report.Symbols[len(report.Symbols)-1], nil
		}
	}
	return nil, nil
}

type runtimeNativeOwner struct {
	symbols, owner, sources string
}

// Explicit symbol sets make newly introduced native boundaries request review.
// Sources include all platform implementations where selection happens in C
// preprocessing or LLGoFiles. SDK entries are supplied by the target sysroot.
var runtimeNativeOwners = []runtimeNativeOwner{
	{
		`C.getenv C.getpagesize C.gettimeofday C.poll C.rand C.srand C.syscall C.write C.getentropy
C.clock_gettime_nsec_np`,
		"target libc / platform SDK",
		``,
	},
	{"stdcall.GetSystemDirectoryA", "Windows kernel32; stdcall on 386, native Windows ABI on amd64/arm64", ""},
	{"C.__sigsetjmp C.sigsetjmp C.siglongjmp", "target libc; returns-twice and target SigjmpBuf layout", "ssa/eh.go"},
	{
		`C.setjmp C.longjmp`,
		"compiler-selected Windows CRT/context entry; these logical names are not direct CRT calls",
		`ssa/eh.go runtime/internal/runtime/_wrap/setjmp_windows_amd64.c
runtime/internal/runtime/_wrap/setjmp_windows_arm64.c`,
	},
	{"C.__llgo_wasm_main", "compiler-generated Wasm scheduler entry", "internal/build/main_module.go"},
	{
		`C.llgo_libc_call C.llgo_cgo_argset_call C.llgo_cgo_setuid C.llgo_cgo_seteuid C.llgo_cgo_setgid
C.llgo_cgo_setegid C.llgo_cgo_setreuid C.llgo_cgo_setregid C.llgo_cgo_setresuid
C.llgo_cgo_setresgid C.llgo_cgo_setgroups`,
		"LLGo packed C argument and libc credential bridges",
		`runtime/internal/lib/runtime/_wrap/libcall.c`,
	},
	{
		`C.llgo_cpu_profile_drain C.llgo_cpu_profile_refresh_signal C.llgo_cpu_profile_start
C.llgo_cpu_profile_stop C.llgo_cpu_profile_update_signal`,
		"LLGo platform profiler",
		`runtime/internal/lib/runtime/_wrap/profile.c
runtime/internal/lib/runtime/_wrap/profile_windows.c`,
	},
	{"C.llgo_debugtrap", "LLGo platform debug trap", "runtime/internal/lib/runtime/_wrap/debugtrap.c"},
	{
		`C.llgo_dynunwind_endfp C.llgo_dynunwind_name C.llgo_dynunwind_pccount`,
		"LLGo native unwinder",
		`runtime/internal/lib/runtime/_wrap/dynunwind.c`,
	},
	{"C.llgo_fault_capture_done C.llgo_install_fault_handler", "LLGo POSIX fault bridge", "runtime/internal/lib/runtime/_wrap/fault.c"},
	{"C.llgo_framepointer", "LLGo frame-pointer intrinsic bridge", "runtime/internal/runtime/_wrap/fp.c"},
	{"C.llgo_gc_stack_top", "LLGo linear-memory GC", "runtime/internal/runtime/tinygogc/_wrap/gc_wasm.c"},
	{
		`C.llgo_getpagesize C.llgo_iocp_associate C.llgo_iocp_create C.llgo_iocp_get C.llgo_nanotime
C.llgo_nanotime_init C.llgo_query_performance_counter C.llgo_query_performance_frequency
C.llgo_walltime C.llgo_windows_capture_context C.llgo_windows_lookup_function_entry
C.llgo_windows_signal_init C.llgo_windows_virtual_unwind`,
		"LLGo Windows SDK bridge",
		`runtime/internal/lib/runtime/_wrap/runtime_windows.c`,
	},
	{
		`C.llgo_install_windows_fault_handler C.llgo_windows_fault_capture_done
C.llgo_windows_fault_pcbuf`,
		"LLGo Windows fault bridge",
		`runtime/internal/runtime/_wrap/fault_windows.c`,
	},
	{
		`C.llgo_maxprocs`,
		"LLGo platform CPU query",
		`runtime/internal/lib/runtime/_wrap/runtime.c
runtime/internal/lib/runtime/_wrap/runtime_windows.c`,
	},
	{
		`C.llgo_mem_readable`,
		"LLGo platform memory probe",
		`runtime/internal/lib/runtime/_wrap/fault.c runtime/internal/lib/runtime/_wrap/runtime_windows.c`,
	},
	{
		`C.llgo_print_byte C.llgo_print_stderr_is_console C.llgo_print_write C.llgo_print_write_console`,
		"LLGo Windows console bridge",
		`runtime/internal/runtime/_wrap/print_windows.c`,
	},
	{
		`C.llgo_signal_barrier C.llgo_signal_die_pipe C.llgo_signal_disable C.llgo_signal_enable
C.llgo_signal_ignore C.llgo_signal_ignore_pipe C.llgo_signal_init C.llgo_signal_raise_pipe
C.llgo_signal_recv`,
		"LLGo POSIX signal bridge",
		`runtime/internal/lib/runtime/_wrap/signal.c`,
	},
	{"C.llgo_synctest_can_associate", "LLGo synctest heap classification", "runtime/internal/runtime/_wrap/synctest_heap.c"},
	{
		`C.llgo_timer_cond_init C.llgo_timer_cond_timedwait C.llgo_wasi_monotonic_time`,
		"LLGo pthread/WASI clock and timed-wait adapter",
		`runtime/internal/lib/runtime/_wrap/timer_wait.c`,
	},
	{
		`C.llgo_traceback_all_threads_syscall C.llgo_traceback_mark_owned`,
		"LLGo owned-thread registry and guarded syscall protocol",
		`runtime/internal/stacktrace/_wrap/allthreads_linux.h`,
	},
	{"C.llgo_traceback_config C.llgo_traceback_crash", "LLGo fatal-output bridge", "runtime/internal/runtime/_wrap/traceback_crash.c"},
	{
		`C.llgo_wasi_gc_allocator_finish C.llgo_wasi_gc_allocator_lock C.llgo_wasi_gc_cond_timedwait
C.llgo_wasi_gc_enter_begin C.llgo_wasi_gc_enter_end C.llgo_wasi_gc_leave_begin
C.llgo_wasi_gc_leave_end C.llgo_wasi_gc_mutex_lock C.llgo_wasi_gc_park C.llgo_wasi_gc_pending
C.llgo_wasi_gc_resume C.llgo_wasi_gc_stop C.llgo_wasi_gc_thread_start
C.llgo_wasi_gc_wait_uint32`,
		"LLGo WASI thread/GC synchronization",
		`runtime/internal/runtime/_wrap/wasi_gc_world.c`,
	},
	{
		`C.llgo_wasm_call_interface_sret C.llgo_wasm_call_sret`,
		"LLGo Wasm finalizer ABI bridge",
		`runtime/internal/lib/runtime/_wrap/finalizer_wasm.c`,
	},
	{"C.llgo_windows_random", "LLGo Windows random source", "runtime/internal/runtime/_wrap/rand_windows.c"},
	{"C.llgo_windows_syscall", "LLGo Windows variable-arity syscall bridge", "runtime/internal/lib/runtime/_wrap/syscall_windows.S"},
}

func runtimeNativeProviders(owners []runtimeNativeOwner) (map[string]runtimeNativeOwner, error) {
	providers := make(map[string]runtimeNativeOwner)
	for _, owner := range owners {
		for _, name := range strings.Fields(owner.symbols) {
			if previous, ok := providers[name]; ok {
				return nil, fmt.Errorf("runtime native provider %s has multiple entries: %q and %q", name, previous.owner, owner.owner)
			}
			providers[name] = owner
		}
	}
	return providers, nil
}
