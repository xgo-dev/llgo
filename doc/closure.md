# Closure ABI

This document records the phase-one design from
[proposal #2170](https://github.com/xgo-dev/llgo/issues/2170).

## Decisions

- Keep the existing two-word function value: `{fn, env}`. This change does not
  add flags or switch to Go's one-pointer funcval layout.
- A Go/go-types function signature never contains an environment parameter.
  `llssa.Function` records independently whether its physical entry needs env.
- Plain Go functions and C functions use `{real entry, nil}`. Closures and
  bound method wrappers use `{real entry, non-nil env}`.
- `env == nil` means that the physical entry has no env parameter. A source
  closure whose captures are all zero-sized is reclassified as a no-env entry;
  it recreates their permitted shared non-nil address from the module sentinel.
  Other required environments never use nil. A nil pointer receiver method
  value is represented by an allocated environment cell, and an interface
  method value captures the complete interface state.
- A statically known call uses the entry's `NeedsEnv` property. An
  explicit-context dynamic funcval call (including WebAssembly) branches once
  on `env != nil` and emits two exact LLVM call edges: `fn(args...)` and
  `fn(env, args...)`.
- Native hidden env parameters use LLVM `nest` or `swiftself` parameter
  attributes. WebAssembly and targets without a validated hidden-register FFI
  bridge use an explicit physical env parameter, but only on env-bearing
  entries.
- Direct interface invocation remains a transient `(method entry, receiver)`
  operation. Its receiver is an ordinary ABI argument; first-class interface
  method values are lowered through the normal bound-wrapper closure path.
- Function values point directly at their physical entries; closure calls do
  not add a generated adapter layer. C function values point directly at the C
  entry.

Native dynamic calls always use one hidden-env call edge, including when env is
nil. An optimizer barrier keeps the indirect code pointer opaque: LLVM IR
considers `R(ptr nest, args...)` and `R(args...)` different prototypes and must
not devirtualize a plain entry into the hidden-env call edge. The barrier emits
no machine instruction.

The backend selects this ABI from the resolved LLVM target triple, not from
`GOARCH`. `GOOS/GOARCH` select Go source files and type sizes, but named targets
may intentionally reuse a compatible Go architecture: for example,
`wasm-unknown`, `wasip2`, Xtensa, AVR, and some RISC-V targets use `GOARCH=arm`
while emitting a different physical architecture. The triple is what LLVM uses
to assign `nest`/`swiftself` registers.

## Physical entry ABI

An env-bearing entry is created from the semantic signature plus one backend
parameter:

```text
semantic:  R func(A, B)
physical:  R entry(env, A, B)
```

The physical env parameter is:

- `nest` on validated x86, RISC-V, and AArch64 platforms where X18 is
  available;
- `swiftself` on ARM and platforms where AArch64 X18 is reserved;
- an ordinary leading parameter on the explicit fallback.

Windows currently uses the explicit fallback because this phase does not add a
Windows final-hop FFI bridge. This is not an LLVM `nest` limitation: for
example, x86-64 Win64 can use its static-chain register once the corresponding
bridge is available and tested.

LLVM parameter attributes are preserved when LLGo rewrites large aggregate
returns or lowers its C ABI.

## FFI and reflection

`reflect.Value.Call` starts from the semantic libffi signature:

- explicit env target: add the env type/value only when `env != nil`; otherwise
  use the semantic signature. This is the only `env != nil` decision in the
  reflection/FFI path;
- native hidden env target: `ffi.CallWithEnv` first uses `ffi_call_go` when the
  linked libffi exports it and its Go static-chain register matches LLGo's
  `nest` ABI. Otherwise stock `ffi_call` marshals the semantic arguments into a
  small runtime final-hop trampoline. The trampoline preserves those arguments,
  obtains `{fn, env}` from thread-local call state, installs the target's
  `nest` or `swiftself` register (including nil), and enters fn with the original
  stack pointer. Transports in caller-saved registers tail-jump; ARM `swiftself`
  returns through a continuation that restores its callee-saved self register.

Although libffi's C implementation of `ffi_call_go` is a thin wrapper, it calls
an architecture-private `ffi_call_int(..., closure)` and matching assembly; it
cannot be reproduced outside libffi by wrapping public `ffi_call` alone. The
optional direct path plus public-API fallback requires neither a patched libffi
nor rebuilding it. AArch64 libffi's Go ABI writes X18, so Apple/Android
`swiftself`/X20 deliberately uses the fallback. `reflect.MakeFunc` remains a
normal libffi C closure: its funcval has `env == nil`, while libffi userdata owns
the callback state separately.

## Scope

This phase includes closure creation/calls, method values, C function values,
ABI rewriting, reflection, FFI, and direct-entry function values.

It deliberately excludes:

- a WebAssembly TLS/mutable-global optimization for `g.ctxt`;
- flags or a future one-pointer funcval representation.
