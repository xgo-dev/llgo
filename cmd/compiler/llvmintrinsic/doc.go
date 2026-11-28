// Package llvmintrinsic describes and checks LLVM intrinsic signatures.
//
// Descriptions are optional refinements of LLVM's own signature metadata. They
// operate on logical LLVM value types, before Go storage or native ABI lowering.
// The caller remains responsible for mapping Go types and for verifying the
// completed IR in its actual control-flow, GC and target-feature context.
//
// Structural patterns are reusable across contexts; Exact preserves the
// supplied LLVM type's context and identity. A resolved Binding belongs to the
// context of its signature and must not outlive it. References are checked after
// collecting bindings, so a result can refer to a later argument.
package llvmintrinsic
