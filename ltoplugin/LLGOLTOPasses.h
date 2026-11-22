#ifndef LLGO_LTO_PASSES_H
#define LLGO_LTO_PASSES_H

#include "llvm/IR/PassManager.h"

namespace llgo {

inline constexpr char LLGOPreGlobalDCEPassName[] = "llgo-lto-pre-globaldce";
inline constexpr char LLGOInterfaceMethodTypeIDPassName[] =
    "llgo-interface-method-typeids";

void addLLGOInterfaceMethodTypeIDPass(llvm::ModulePassManager &MPM);
void addLLGOReflectMethodByNamePass(llvm::ModulePassManager &MPM);
void addLLGOTypeIDExportCleanupPass(llvm::ModulePassManager &MPM);

void addSIMDRegisterVerifier(llvm::ModulePassManager &MPM);

inline void addLLGOPreGlobalDCEPipeline(llvm::ModulePassManager &MPM) {
  addLLGOInterfaceMethodTypeIDPass(MPM);
  addLLGOReflectMethodByNamePass(MPM);
}

} // namespace llgo

#endif // LLGO_LTO_PASSES_H
