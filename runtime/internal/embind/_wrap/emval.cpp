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

template<typename T>
TYPEID take_typeid() {
    typename WithPolicies<>::template ArgTypeList<T> targetType;
    return targetType.getTypes()[0];    
}

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

static TYPEID typeid_val = take_typeid<val>();

#if LLGO_EMVAL_INVOKER_API
EM_INVOKER take_invoker(int nargs, EM_INVOKER_KIND kind, const TYPEID *types) {
    static thread_local std::vector<EM_INVOKER> invokers[3];
    std::vector<EM_INVOKER>& byArity = invokers[static_cast<int>(kind)];
    if (byArity.size() <= static_cast<size_t>(nargs)) {
        byArity.resize(nargs + 1, nullptr);
    }
    EM_INVOKER& invoker = byArity[nargs];
    if (invoker == nullptr) {
        invoker = _emval_create_invoker(nargs + 1, types, kind);
    }
    return invoker;
}

EM_VAL take_val_result(EM_GENERIC_WIRE_TYPE result) {
    using WireType = BindingType<val>::WireType;
    WireType wire = GenericWireTypeConverter<WireType>::from(result);
    return BindingType<val>::fromWireType(wire).release_ownership();
}

// _emval_invoke throws a native JavaScript exception. Wasm C++ catch clauses
// do not see that object, so convert it here before returning to Go.
EM_JS(EM_GENERIC_WIRE_TYPE, llgo_emval_invoke, (EM_INVOKER caller, EM_VAL handle, const char* methodName, EM_DESTRUCTORS* destructors, void* args, int *error, EM_VAL *thrown), {
    try {
        HEAP32[error >> 2] = 0;
        HEAPU32[thrown >> 2] = 0;
        return emval_methodCallers[caller](handle, methodName, destructors, args);
    } catch (jsErr) {
        HEAP32[error >> 2] = 1;
        HEAPU32[thrown >> 2] = Emval.toHandle(jsErr);
        return 0;
    }
});
#endif

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

EM_VAL llgo_emval_method_call(EM_VAL object, const char* name, EM_VAL args[], int nargs, int *error) {
    std::vector<TYPEID> arr;
    arr.resize(nargs+1);
    std::vector<GenericWireType> elements;
    elements.resize(nargs);
    GenericWireType *cursor = elements.data();
    arr[0] = typeid_val;
    for (int i = 0; i < nargs; i++) {
        arr[i+1] = typeid_val;
        EM_VAL arg = llgo_emval_normalize(args[i]);
        _emval_incref(arg);
        writeGenericWireTypes(cursor, arg);
    }
#if LLGO_EMVAL_INVOKER_API
    EM_INVOKER caller = take_invoker(nargs, EM_INVOKER_KIND::METHOD, arr.data());
#else
    EM_METHOD_CALLER caller = _emval_get_method_caller(nargs+1,&arr[0],EM_METHOD_CALLER_KIND::FUNCTION);
#endif
    EM_DESTRUCTORS destructors = nullptr;
    *error = 0;
#if LLGO_EMVAL_INVOKER_API
    EM_VAL thrown = nullptr;
    EM_GENERIC_WIRE_TYPE ret = llgo_emval_invoke(caller, llgo_emval_normalize(object), name, &destructors, elements.data(), error, &thrown);
    if (*error) {
        return thrown;
    }
    DestructorsRunner dr(destructors);
    return take_val_result(ret);
#else
    EM_GENERIC_WIRE_TYPE ret;
    try {
        ret = _emval_call_method(caller, llgo_emval_normalize(object), name, &destructors, elements.data());
    } catch(const emscripten::val& jsErr) {
        *error = 1;
        emscripten::val errorValue = jsErr;
        return errorValue.release_ownership();
    }
    return fromGenericWireType<val>(ret).release_ownership();
#endif
}

/*
kind:
FUNCTION = 0,
CONSTRUCTOR = 1,
*/
EM_VAL llgo_emval_call(EM_VAL fn, EM_VAL args[], int nargs, int kind, int *error) {
   std::vector<TYPEID> arr;
   arr.resize(nargs+1);
   std::vector<GenericWireType> elements;
   elements.resize(nargs);
   GenericWireType *cursor = elements.data();
   arr[0] = typeid_val;
   for (int i = 0; i < nargs; i++) {
       arr[i+1] = typeid_val;
       EM_VAL arg = llgo_emval_normalize(args[i]);
       _emval_incref(arg);
       writeGenericWireTypes(cursor, arg);
   }
#if LLGO_EMVAL_INVOKER_API
   EM_INVOKER_KIND invokerKind = kind == 0
       ? EM_INVOKER_KIND::FUNCTION
       : EM_INVOKER_KIND::CONSTRUCTOR;
   EM_INVOKER caller = take_invoker(nargs, invokerKind, arr.data());
#else
   EM_METHOD_CALLER caller = _emval_get_method_caller(nargs+1,&arr[0],EM_METHOD_CALLER_KIND(kind));
#endif
   EM_DESTRUCTORS destructors = nullptr;
   *error = 0;
#if LLGO_EMVAL_INVOKER_API
   EM_VAL thrown = nullptr;
   EM_GENERIC_WIRE_TYPE ret = llgo_emval_invoke(caller, llgo_emval_normalize(fn), nullptr, &destructors, elements.data(), error, &thrown);
   if (*error) {
       return thrown;
   }
   DestructorsRunner dr(destructors);
   return take_val_result(ret);
#else
   EM_GENERIC_WIRE_TYPE ret;
   try {
       ret = _emval_call(caller, llgo_emval_normalize(fn), &destructors, elements.data());
   } catch(const emscripten::val& jsErr) {
       *error = 1;
       emscripten::val errorValue = jsErr;
       return errorValue.release_ownership();
   }
   return fromGenericWireType<val>(ret).release_ownership();
#endif
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
