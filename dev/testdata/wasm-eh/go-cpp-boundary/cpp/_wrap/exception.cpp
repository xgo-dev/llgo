#include <emscripten.h>

// Read only the harness argument, without depending on Go's os.Args.
EM_JS(int, llgo_eh_test_mode, (), {
    var args = Module['arguments'] || [];
    // Go startup can prepend argv[0] to this shared array.
    var mode = args[args.length - 1];
    return mode === 'direct' ? 1 : mode === 'indirect' ? 2 : 0;
});

// Keep the catch in a separate function even under LTO. A following Go sleep
// must not be folded into the Wasm catch, where Asyncify cannot suspend.
static __attribute__((noinline)) int catch_status() {
    try {
        throw 7;
    } catch (int value) {
        return value;
    }
}

// The Wasm optimizer does not preserve LLVM's noinline attribute. A volatile
// function pointer also keeps the catch boundary through post-link inlining.
static int (*volatile catch_boundary)() = catch_status;
extern "C" int llgo_eh_cpp_catch() { return catch_boundary(); }

// This callback only sleeps; it cannot throw a C++ exception or panic.
extern "C" void llgo_eh_suspend_callback() noexcept;

// Deliberately unsupported: a Go callback suspends before the foreign catch
// exits. Check both direct and indirect calls without relying on inlining.
extern "C" __attribute__((noinline)) void
llgo_eh_cpp_catch_and_suspend(int indirect) {
    try {
        throw 7;
    } catch (int value) {
        if (indirect) {
            void (*volatile callback)() noexcept = llgo_eh_suspend_callback;
            callback();
        } else {
            llgo_eh_suspend_callback();
        }
    }
}
