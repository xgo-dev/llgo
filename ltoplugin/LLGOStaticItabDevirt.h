#ifndef LLGO_STATIC_ITAB_DEVIRT_H
#define LLGO_STATIC_ITAB_DEVIRT_H

#include "llvm/IR/DataLayout.h"
#include "llvm/IR/Module.h"

namespace llgo {

// Devirtualize checked loads whose receiver itab resolves to a compile-time
// static itab template (_llgo_itab$*), either directly, via a local-linkage
// helper parameter, or through runtime.NewItab(inter, concrete).
bool devirtualizeStaticItabCalls(llvm::Module &M, const llvm::DataLayout &DL);

// Remove _llgo_itab$* templates from llvm.compiler.used and erase dormant
// templates before GlobalDCE builds its type-id candidate sets.
bool eraseStaticItabTemplatePreservation(llvm::Module &M);

} // namespace llgo

#endif // LLGO_STATIC_ITAB_DEVIRT_H
