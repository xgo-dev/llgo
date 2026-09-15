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

EM_JS(void, llgo_emval_install_invoke_js, (uint8_t *pending_flag), {
    // Reinstalling would drop queued host events. Keep the first bridge for
    // the life of the module.
    if (Module["_llgo_invoke"]) {
        return;
    }
    const pending = [];
    const pendingFlag = Number(pending_flag);
    Module["llgoWasmPendingInvokes"] = pending;
    Module["_llgo_invoke"] = function(event) {
        pending.push(event);
        HEAPU8[pendingFlag] = 1;
        const state = Module["llgoWasmHostWait"];
        if (state !== undefined && state.wake !== undefined) {
            const wake = state.wake;
            delete state.wake;
            setTimeout(wake, 0);
        }
    };
});

static bool llgo_emval_invoke_installed;

void llgo_emval_install_invoke(void) {
    if (llgo_emval_invoke_installed) {
        return;
    }
    llgo_emval_invoke_installed = true;
    llgo_emval_invoke_pending = 0;
    llgo_emval_install_invoke_js(const_cast<uint8_t *>(&llgo_emval_invoke_pending));
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

EM_VAL llgo_emval_new_string(const char *str) {
    return _emval_new_u8string(str);
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

// Catch JS exceptions in JavaScript. C++ catch(emscripten::val) does not see
// throws from wasm imports such as Node's fs.statSync, so recoverErr would
// never observe js.Error and the process would exit on the uncaught exception.
EM_JS(void, llgo_emval_install_try_call_js, (), {
    if (Module["_llgo_emval_try_call"]) {
        return;
    }
    Module["_llgo_emval_try_call"] = function(target, method, args, kind) {
        try {
            var result;
            if (kind === 1) {
                result = Reflect.construct(target, args);
            } else if (kind === 2) {
                result = Reflect.apply(target[method], target, args);
            } else {
                result = Reflect.apply(target, undefined, args);
            }
            return [0, result];
        } catch (e) {
            return [1, e];
        }
    };
});

static EM_VAL llgo_emval_try_call(EM_VAL target, const char* method, EM_VAL args[], int nargs, int kind, int *error) {
    llgo_emval_install_try_call_js();
    val jsArgs = val::array();
    for (int i = 0; i < nargs; i++) {
        EM_VAL arg = llgo_emval_normalize(args[i]);
        _emval_incref(arg);
        jsArgs.call<void>("push", val::take_ownership(arg));
    }
    EM_VAL targetHandle = llgo_emval_normalize(target);
    _emval_incref(targetHandle);
    val targetVal = val::take_ownership(targetHandle);
    val methodVal = method ? val(method) : val::undefined();
    val helper = val::module_property("_llgo_emval_try_call");
    val outcome = helper(targetVal, methodVal, jsArgs, kind);
    *error = outcome[0].as<int>();
    return outcome[1].release_ownership();
}

EM_VAL llgo_emval_method_call(EM_VAL object, const char* name, EM_VAL args[], int nargs, int *error) {
    return llgo_emval_try_call(object, name, args, nargs, 2, error);
}

/*
kind:
FUNCTION = 0,
CONSTRUCTOR = 1,
*/
EM_VAL llgo_emval_call(EM_VAL fn, EM_VAL args[], int nargs, int kind, int *error) {
    return llgo_emval_try_call(fn, nullptr, args, nargs, kind, error);
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
