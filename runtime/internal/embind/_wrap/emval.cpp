#include <string>
#include <stdint.h>
#include <emscripten.h>
#include <emscripten/val.h>
#include <emscripten/bind.h>
#include <emscripten/version.h>

using namespace emscripten;
using namespace emscripten::internal;

// Emscripten 4.0.11 replaced the private method-caller entry points with the
// invoker API. Keep the old path so existing SDKs remain usable while current
// SDKs use the supported interface.
#ifndef __EMSCRIPTEN_MAJOR__
#define __EMSCRIPTEN_MAJOR__ __EMSCRIPTEN_major__
#define __EMSCRIPTEN_MINOR__ __EMSCRIPTEN_minor__
#define __EMSCRIPTEN_TINY__ __EMSCRIPTEN_tiny__
#endif

#define LLGO_EMVAL_INVOKER_API \
    (__EMSCRIPTEN_MAJOR__ > 4 || \
     (__EMSCRIPTEN_MAJOR__ == 4 && (__EMSCRIPTEN_MINOR__ > 0 || __EMSCRIPTEN_TINY__ >= 11)))

template<typename T, typename... Policies>
EM_VAL take_value(T&& value, Policies...) {
#if LLGO_EMVAL_INVOKER_API
    return val(std::forward<T>(value)).release_ownership();
#else
    typename WithPolicies<Policies...>::template ArgTypeList<T> valueType;
    WireTypePack<T> argv(std::forward<T>(value));
    return _emval_take_value(valueType.getTypes()[0], argv);
#endif
}

template<typename T, typename ...Policies>
T as_value(EM_VAL val, Policies...) {
#if LLGO_EMVAL_INVOKER_API
    _emval_incref(val);
    return emscripten::val::take_ownership(val).as<T>();
#else
    typedef BindingType<T> BT;
    typename WithPolicies<Policies...>::template ArgTypeList<T> targetType;
    
    EM_DESTRUCTORS destructors = nullptr;
    EM_GENERIC_WIRE_TYPE result = _emval_as(
        val,
        targetType.getTypes()[0],
        &destructors);
    DestructorsRunner dr(destructors);
    return fromGenericWireType<T>(result);
#endif
}

struct GoString {
    char *data;
    int len;   
};

extern "C" {

// export from llgo
extern GoString llgo_export_string_from(const char *data, int n);
static EM_VAL llgo_emval_normalize(EM_VAL value) {
    return value == 0 ? EM_VAL(internal::_EMVAL_UNDEFINED) : value;
}

EM_VAL llgo_emval_get_global(const char *name) {
    return _emval_get_global(name);
}

EM_VAL llgo_emval_get_module_property(const char *name) {
    return val::module_property(name).release_ownership();
}

static volatile uint8_t llgo_emval_invoke_pending;

EM_JS_DEPS(llgo_emval_install_invoke_js, "$Emval,$getWasmTableEntry,$Asyncify,$Fibers");
EM_JS(void, llgo_emval_install_invoke_js, (uint8_t *pending_flag, uintptr_t callback, int pointer_bytes), {
    const pending = [];
    const pendingFlag = Number(pending_flag);
    const dispatch = Asyncify.instrumentFunction(getWasmTableEntry(Number(callback)));
    Module['llgoWasmPendingInvokes'] = pending;
    Module['_llgo_invoke'] = function(event) {
        if (Asyncify.exportCallStack.length) {
            const handle = Emval.toHandle(event);
            // Give reentrant Go its own unwind boundary. Replaying the
            // enclosing JS call would repeat arbitrary host side effects.
            const callStack = Asyncify.exportCallStack;
            const trampolineRunning = Fibers.trampolineRunning;
            Asyncify.exportCallStack = [];
            Fibers.trampolineRunning = false;
            try {
                dispatch(pointer_bytes === 8 ? BigInt(handle) : handle);
            } finally {
                Asyncify.exportCallStack = callStack;
                Fibers.trampolineRunning = trampolineRunning;
            }
            return event.result;
        }
        pending.push(event);
        HEAPU8[pendingFlag] = 1;
        const state = Module['llgoWasmHostWait'];
        if (state !== undefined && state.wake !== undefined) {
            const wake = state.wake;
            delete state.wake;
            // No compiled frames are active. Resume Go before returning the
            // callback result to the event's JavaScript caller.
            wake();
        }
        return event.result;
    };
});

void llgo_emval_install_invoke(void (*callback)(EM_VAL)) {
    llgo_emval_invoke_pending = 0;
    llgo_emval_install_invoke_js(const_cast<uint8_t *>(&llgo_emval_invoke_pending), uintptr_t(callback), sizeof(EM_VAL));
}

bool llgo_emval_has_pending_invoke(void) {
    return llgo_emval_invoke_pending != 0;
}

EM_VAL llgo_emval_take_pending_invoke(void) {
    if (llgo_emval_invoke_pending == 0) {
        return 0;
    }
    val pending = val::module_property("llgoWasmPendingInvokes");
    if (pending.isUndefined() || pending["length"].as<unsigned>() == 0) {
        llgo_emval_invoke_pending = 0;
        return 0;
    }
    val event = pending.call<val>("shift");
    if (pending["length"].as<unsigned>() == 0) {
        llgo_emval_invoke_pending = 0;
    }
    return event.release_ownership();
}

EM_VAL llgo_emval_new_double(double v) {
    return take_value(v);
}

// Go strings carry a byte length and may contain embedded NUL characters.
EM_JS_DEPS(llgo_emval_string_bytes, "$Emval,$UTF8ToString");
EM_JS(EM_VAL, llgo_emval_string_bytes,
      (const char *str, size_t length, int pointer_bytes), {
    const handle = Emval.toHandle(UTF8ToString(Number(str), Number(length), true));
    return pointer_bytes === 8 ? BigInt(handle) : handle;
});

EM_VAL llgo_emval_new_string(const char *str, size_t length) {
    return llgo_emval_string_bytes(str, length, sizeof(EM_VAL));
}

EM_VAL llgo_emval_new_object() {
    return _emval_new_object();
}

EM_VAL llgo_emval_new_array() {
    return _emval_new_array();
}

void llgo_emval_incref(EM_VAL value) {
    _emval_incref(llgo_emval_normalize(value));
}

void llgo_emval_decref(EM_VAL value) {
    _emval_decref(llgo_emval_normalize(value));
}

void llgo_emval_set_property(EM_VAL object, EM_VAL key, EM_VAL value) {
    _emval_set_property(llgo_emval_normalize(object), llgo_emval_normalize(key), llgo_emval_normalize(value));
}

EM_VAL llgo_emval_get_property(EM_VAL object, EM_VAL key) {
    return _emval_get_property(llgo_emval_normalize(object), llgo_emval_normalize(key));
}

bool llgo_emval_is_number(EM_VAL object) {
    return _emval_is_number(llgo_emval_normalize(object));
}

bool llgo_emval_is_string(EM_VAL object) {
    return _emval_is_string(llgo_emval_normalize(object));
}

bool llgo_emval_in(EM_VAL item, EM_VAL object) {
    return _emval_in(llgo_emval_normalize(item), llgo_emval_normalize(object));
}

bool llgo_emval_delete(EM_VAL object, EM_VAL property) {
    return _emval_delete(llgo_emval_normalize(object), llgo_emval_normalize(property));
}

EM_VAL llgo_emval_typeof(EM_VAL value) {
    return _emval_typeof(llgo_emval_normalize(value));
}

bool llgo_emval_instanceof(EM_VAL object, EM_VAL constructor) {
    return _emval_instanceof(llgo_emval_normalize(object), llgo_emval_normalize(constructor));
}

EM_JS_DEPS(llgo_emval_length, "$Emval");
EM_JS(double, llgo_emval_length, (EM_VAL object), {
    const length = parseInt(Emval.toValue(Number(object) || 2).length);
    // wasm_exec.js writes non-finite results through setInt64, which produces
    // zero. Normalize here before Go converts the result to int.
    return Number.isFinite(length) ? length : 0;
});

double llgo_emval_as_double(EM_VAL v) {
    return as_value<double>(llgo_emval_normalize(v));
}

GoString llgo_emval_as_string(EM_VAL v) {
    std::string value = as_value<std::string>(llgo_emval_normalize(v));
    return llgo_export_string_from(value.c_str(), int(value.size()));
}

bool llgo_emval_equals(EM_VAL first, EM_VAL second) {
    return _emval_equals(llgo_emval_normalize(first), llgo_emval_normalize(second));
}

// JavaScript throws cannot be caught by C++ catch(val). Match Go's
// syscall/js bridge by returning the original thrown value plus an error bit.
EM_JS_DEPS(llgo_emval_try_call, "$Emval,$UTF8ToString,$ExitStatus");
EM_JS(EM_VAL, llgo_emval_try_call,
      (EM_VAL handle, const char *method, size_t method_length, EM_VAL *args, int nargs, int kind,
       int *error, int pointer_bytes), {
    const argv = [];
    const slots = pointer_bytes === 8 ? HEAPU64 : HEAPU32;
    const first = Number(args) / pointer_bytes;
    for (let i = 0; i < nargs; i++) {
        argv.push(Emval.toValue(Number(slots[first + i]) || 2));
    }
    let result;
    let failed = 0;
    try {
        const value = Emval.toValue(Number(handle) || 2);
        if (method) {
            const name = UTF8ToString(Number(method), Number(method_length), true);
            result = Reflect.apply(Reflect.get(value, name), value, argv);
        } else if (kind === 1) {
            result = Reflect.construct(value, argv);
        } else {
            result = Reflect.apply(value, undefined, argv);
        }
    } catch (thrown) {
        // Emscripten exit/suspension and native Wasm exceptions carry runtime
        // control flow, not syscall/js errors. Keep them with their owner.
        if (thrown === 'unwind' || thrown instanceof ExitStatus ||
            (typeof WebAssembly.Exception === 'function' && thrown instanceof WebAssembly.Exception)) {
            throw thrown;
        }
        failed = 1;
        result = thrown;
    }
    // A callback may grow memory. Use the current heap view after the call.
    HEAP32[Number(error) / 4] = failed;
    const resultHandle = Emval.toHandle(result);
    // EM_VAL is a C pointer even though Emval's JS table uses numeric handles.
    return pointer_bytes === 8 ? BigInt(resultHandle) : resultHandle;
});

EM_VAL llgo_emval_method_call(EM_VAL object, const char *name, size_t name_length, EM_VAL args[], int nargs, int *error) {
    return llgo_emval_try_call(object, name, name_length, args, nargs, 0, error, sizeof(EM_VAL));
}

EM_VAL llgo_emval_call(EM_VAL fn, EM_VAL args[], int nargs, int kind, int *error) {
    return llgo_emval_try_call(fn, nullptr, 0, args, nargs, kind, error, sizeof(EM_VAL));
}

EM_VAL llgo_emval_memory_view_uint8(size_t length, uint8_t *data) {
    val view{ typed_memory_view<uint8_t>(length,data) };
    return view.release_ownership();
}

void llgo_emval_dump(EM_VAL v) {
    v = llgo_emval_normalize(v);
    _emval_incref(v);
    val console = val::global("console");
    console.call<void>("log", val::take_ownership(v));
}

}
