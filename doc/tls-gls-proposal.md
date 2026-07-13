Proposal draft, 2026-07-13

# Proposal: LLGo TLS and GLS Variable Directives

## Summary

Add two package-variable directives:

```go
//llgo:tls
var threadState T

//llgo:gls
var goroutineState U
```

`//llgo:tls` gives each operating-system thread an independent instance of the
variable. `//llgo:gls` gives each Go goroutine an independent instance.

LLGo currently implements one Go goroutine with one pthread. The initial
implementation can therefore lower both directives to LLVM thread-local
storage. Their source semantics and compiler metadata remain distinct so that
`gls` can move into a goroutine object when LLGo gains an M:N scheduler, while
`tls` remains tied to the operating-system thread.

## Goals

* Provide low-cost access to LLGo runtime state owned by a thread or goroutine.
* Define initialization, recursion, panic, GC, and thread-exit behavior.
* Preserve the TLS/GLS distinction across package loading and build caches.
* Support references to local variables from other packages.
* Avoid a pthread-specific API in ordinary LLGo runtime code.

## Non-Goals

* Implement an M:N scheduler as part of this proposal.
* Define TLS semantics for bare-metal targets that have no thread lifecycle.
* Provide source compatibility with the complete platform-specific TLS APIs.
* Make these directives part of the Go language or portable to other Go
  compilers.

## Terminology and Naming

TLS means thread-local storage. GLS means goroutine-local storage. The short
names describe the ownership model directly and form a clear pair. Longer
spellings such as `threadlocal` and `goroutinelocal` are not proposed as aliases:
aliases become permanent syntax without adding semantics.

The spelling follows existing LLGo directives: no space is required between
`//` and `llgo:`. A parser may continue to accept the spaced form consistently
with other LLGo directives, but documentation and generated code use the
canonical form.

## Semantics

| Directive | Owner | Initial lowering | Future M:N lowering |
| --- | --- | --- | --- |
| `//llgo:tls` | OS thread | LLVM TLS | LLVM TLS |
| `//llgo:gls` | Go goroutine | LLVM TLS | Field or side storage owned by `g` |

Taking a variable's address returns the address of the current owner's
instance. Access from another package has the same behavior as access from the
declaring package.

The LLGo runtime's pointer to the current goroutine is TLS, not GLS. It is the
bootstrap used to find GLS after an M:N scheduler is introduced.

## Declaration Rules

The directives apply only to package-level `var` declarations. They do not
apply to constants, types, functions, parameters, or local variables.

A declaration cannot carry both directives. All non-blank variables in one
multi-value initialization must use the same ownership class because Go
evaluates its right-hand side as one initialization unit. A blank identifier
cannot be local storage. Invalid combinations are compile-time errors.

The initial implementation does not combine these directives with `go:embed`.
Such a declaration is rejected until its initialization and generated data
ownership are specified.

## Initialization

Each owner starts with a zero-valued instance. A variable with an explicit
initializer evaluates that initializer exactly once for each owner before the
owner's first use of any variable in the same initialization unit.

Package initialization on the initial thread evaluates local initializers in
the same order as ordinary Go package variables. It must not duplicate existing
initializer side effects. A newly created thread or goroutine initializes its
instance lazily on first access.

Initialization has three states: uninitialized, initializing, and initialized.
Recursive access while the state is initializing observes the partially
initialized instance and does not re-enter the initializer. If initialization
panics, the state remains initializing and the initializer is not retried. This
matches the process-wide rule that package initialization side effects are not
silently repeated.

Zero-valued, pointer-free local variables need no initialization guard. A
zero-valued variable that can contain Go pointers still performs the root
registration described below.

## GC Roots and Lifetime

Native TLS is outside the ordinary Go heap and global-root scan. Before a
pointer-containing TLS allocation can be used, the runtime registers its byte
range as a conservative GC root for the current thread. Registration is once
per range per thread.

The runtime keeps all registered ranges in one thread-owned list. One pthread
destructor removes those roots and releases the list when the thread exits.
The key is created lazily so runtime-owned TLS can register safely during
runtime package initialization. Steady-state variable access uses a TLS guard
and does not require a pthread key lookup. In `nogc` builds, root registration
is a no-op.

When GLS is moved into a GC-visible goroutine object, its native TLS root
registration is removed. TLS root handling remains unchanged.

## Compiler Representation

Directive scanning is part of the existing package declaration scan, together
with `go:linkname`, LLGo type directives, and exports. It must not add separate
whole-package AST passes. The program records each successfully scanned package
object so the direct compiler entry point can provide a fallback scan without
repeating the preload scan used by the normal build path.

The program stores one declaration record for each relevant object. The record
contains link name, type background, locality (`none`, `tls`, or `gls`), and
generated initializer/ensure symbols. Lowering consumes this record instead of
rescanning comments.

TLS and GLS initially emit LLVM `thread_local` globals. Pointer-free zero-value
variables have no access helper. Other local variables use a generated ensure
function for per-owner initialization and/or root registration. Imported
references declare the same thread-local symbol and call the declaring
package's ensure function when required.

## Build Cache

Declaration records are serialized into each package cache manifest in a
stable order. On a cache hit, the loader validates and atomically merges the
records before compiling dependent packages. Conflicting records or duplicate
ownership result in a cache miss rather than silently selecting one value.

This makes cache hits and source builds provide the same cross-package TLS/GLS
metadata. Aggregation happens during the normal package preload phase, before
SSA construction, so no later global scan is needed.

## Target Support

The first implementation supports LLGo pthread targets with LLVM TLS. Darwin
and Linux are required test platforms. Unsupported targets, including
bare-metal targets without a defined thread lifecycle, should report a clear
compile-time diagnostic instead of silently using process-global storage.

## Compatibility

These directives are new and LLGo-specific. No compatibility alias is required.
The implementation should reject the earlier experimental spellings
`//llgo:threadlocal` and `//llgo:goroutinelocal` so source code does not depend
on syntax that was never released.

## Testing

Tests must cover:

* directive parsing, invalid declarations, and conflicting directives;
* LLVM TLS emission for both ownership classes;
* zero, constant, dynamic, recursive, and panicking initializers;
* independent values on multiple threads and goroutines;
* pointer survival across GC and cleanup at thread exit;
* cross-package access with source builds and cache hits;
* `gc` and `nogc` builds; and
* Darwin and Linux CI targets supported by LLGo.

The runtime migration of the current goroutine pointer must demonstrate that
the steady-state `getg` path performs native TLS address resolution without a
pthread key lookup or heap-backed TLS slot.

## Alternatives Considered

Using only `tls` would describe the current lowering but lose the semantic
boundary required by a future M:N scheduler. A generic `local` directive would
leave ownership ambiguous. The longer `threadlocal` and `goroutinelocal`
spellings are explicit but add verbosity without improving the model defined by
TLS and GLS.

## Open Questions

* Whether bare-metal builds should reject both directives or define GLS for a
  single execution context in a later proposal.
* Whether a future M:N implementation needs a restricted static-initializer
  mode for GLS before it supports arbitrary per-goroutine initializer calls.
