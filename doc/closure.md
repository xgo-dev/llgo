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
  attributes. WebAssembly and architectures without a validated LLVM
  hidden-register mapping use an explicit physical env parameter, but only on
  env-bearing entries.
- Direct interface invocation remains a transient `(method entry, receiver)`
  operation. Its receiver is an ordinary ABI argument; first-class interface
  method values are lowered through the normal bound-wrapper closure path.
- Function values point directly at their physical entries; closure calls do
  not add a generated adapter layer. C function values point directly at the C
  entry.
- PCLN metadata remains function-centric. A compiler-generated wrapper or
  adapter, if one is needed for another purpose, is an ordinary function with
  its own function record; closure environment transport is not part of PCLN.

Native dynamic calls always use one hidden-env call edge, including when env is
nil. An optimizer barrier keeps the indirect code pointer opaque: LLVM IR
considers `R(ptr nest, args...)` and `R(args...)` different prototypes and must
not devirtualize a plain entry into the hidden-env call edge. The barrier emits
no machine instruction.

The backend selects this ABI from the resolved LLVM target triple, not from
`GOARCH`. `GOOS/GOARCH` select Go source files and type sizes, but named embedded
targets such as Xtensa, AVR, and some RISC-V targets may intentionally reuse a
compatible Go architecture while emitting a different physical architecture.
The triple is what LLVM uses to assign `nest`/`swiftself` registers.

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

Windows follows the architecture-selected ABI even though LLGo does not yet
support the OS: x86 uses `nest`, while ARM/AArch64 use `swiftself`. The x86
libffi Go ABI already matches LLVM `nest`; the Windows ARM/AArch64 public-FFI
final hop remains a TODO. This keeps compiled closure entries independent of
when runtime support is added.

WebAssembly remains explicit because it has no compatible hidden-register
transport. Adding one in the future is a deliberate target ABI upgrade.

LLVM parameter attributes are preserved when LLGo rewrites large aggregate
returns or lowers its C ABI.

## FFI and reflection

`reflect.Value.Call` starts from the semantic libffi signature:

- explicit env target: add the env type/value only when `env != nil`; otherwise
  use the semantic signature. This is the only `env != nil` decision in the
  reflection/FFI path;
- native hidden env target: x86, RISC-V, and AArch64 targets where X18 is
  available use libffi's `ffi_call_go` directly when libffi exposes that API,
  because its static-chain register is LLGo's `nest` register. ARM also uses
  `ffi_call_go`; a short final-hop bridge moves libffi's IP/R12 context to
  `swiftself`/R10 without saving argument registers or using TLS. LLVM lowers
  ARM32 `nest` as an ordinary leading argument rather than a hidden static
  chain, so it cannot replace this `swiftself` bridge.
- x86 libffi builds without `FFI_GO_CLOSURES`, including Apple SDK libffi, use
  stock `ffi_call` plus the TLS final-hop trampoline, which installs LLVM's
  `nest` register before entering the real target.
- Apple/Android AArch64 use stock `ffi_call` plus the TLS final-hop trampoline:
  libffi's Go ABI uses X18 while LLGo's entry ABI uses `swiftself`/X20, so the
  public `ffi_call` path needs TLS to carry `{fn, env}` to its final target.

The build obtains both headers and linker flags from `pkg-config libffi`; no
Homebrew-specific libffi path is required. Apple AArch64 remains on the X20 TLS
trampoline because libffi's Go ABI uses X18 while LLGo's selected entry ABI
uses `swiftself`/X20 there.

Every architecture that selects a hidden env must select exactly one native
FFI final hop: direct `ffi_call_go`, `ffi_call_go` plus a register bridge, or
public `ffi_call` plus a TLS trampoline. The wrapper rejects missing or
ambiguous selections at compile time.

Bridge calls are balanced: their targets must return normally through the
final hop so saved registers and, where used, the prior TLS context are
restored.

Although libffi's C implementation of `ffi_call_go` is a thin wrapper, it calls
an architecture-private `ffi_call_int(..., closure)` and matching assembly; it
cannot be reproduced outside libffi by wrapping public `ffi_call` alone. The
compile-time direct path and the public-API fallback require neither a patched
libffi nor rebuilding it. AArch64 libffi's Go ABI writes X18, so Apple/Android
`swiftself`/X20 deliberately uses the public-call fallback. `reflect.MakeFunc`
remains a normal libffi C closure: its funcval has `env == nil`, while libffi
userdata owns the callback state separately.

## Scope

This phase includes closure creation/calls, method values, C function values,
ABI rewriting, reflection, FFI, and direct-entry function values.

It deliberately excludes:

- a WebAssembly TLS/mutable-global optimization for `g.ctxt`;
- flags or a future one-pointer funcval representation.
