.global __llgo_wasm_context_rewinding_state
.hidden __llgo_wasm_context_rewinding_state
.type __llgo_wasm_context_rewinding_state,@function
__llgo_wasm_context_rewinding_state:
    .functype __llgo_wasm_context_rewinding_state () -> (i32)
    i32.const 0
    i32.load8_u __llgo_wasm_context_rewinding
    return
    end_function
