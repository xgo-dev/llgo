#include "LLGOLTOPasses.h"

#include "llvm/Config/llvm-config.h"
#include "llvm/Passes/PassBuilder.h"
#include "llvm/Plugins/PassPlugin.h"
#include "llvm/Support/Compiler.h"
#include "llvm/Transforms/IPO/GlobalDCE.h"

using namespace llvm;

PassPluginLibraryInfo getLLGOLTOPluginInfo() {
  return {LLVM_PLUGIN_API_VERSION, "llgo-lto-plugin", LLVM_VERSION_STRING,
          [](PassBuilder &PB) {
            PB.registerPipelineParsingCallback(
                [](StringRef Name, ModulePassManager &MPM,
                   ArrayRef<PassBuilder::PipelineElement>) {
                  if (Name == "llgo-verify-simd-registers") {
                    llgo::addSIMDRegisterVerifier(MPM);
                    return true;
                  }
                  if (Name == llgo::LLGOInterfaceMethodTypeIDPassName) {
                    llgo::addLLGOInterfaceMethodTypeIDPass(MPM);
                    return true;
                  }
                  if (Name == llgo::LLGOPreGlobalDCEPassName) {
                    llgo::addLLGOPreGlobalDCEPipeline(MPM);
                    return true;
                  }
                  return false;
                });

            // LLVM 22 ends its ThinLTO O0 backend immediately after GlobalDCE,
            // bypassing the optimizer-last extension point.
            if (auto *PIC = PB.getPassInstrumentationCallbacks())
              PIC->registerAfterPassCallback([](StringRef Name, Any IR,
                                                const PreservedAnalyses &) {
                if (Name == GlobalDCEPass::name())
                  llgo::verifySIMDRegisterCalls(*any_cast<const Module *>(IR));
              });

            PB.registerOptimizerLastEPCallback([](ModulePassManager &MPM,
                                                  OptimizationLevel,
                                                  ThinOrFullLTOPhase Phase) {
              if (Phase == ThinOrFullLTOPhase::ThinLTOPostLink)
                llgo::addSIMDRegisterVerifier(MPM);
            });
            PB.registerFullLinkTimeOptimizationEarlyEPCallback(
                [](ModulePassManager &MPM, OptimizationLevel) {
                  llgo::addLLGOPreGlobalDCEPipeline(MPM);
                });
            PB.registerFullLinkTimeOptimizationLastEPCallback(
                [](ModulePassManager &MPM, OptimizationLevel) {
                  llgo::addLLGOTypeIDExportCleanupPass(MPM);
                  llgo::addSIMDRegisterVerifier(MPM);
                });
          }};
}

extern "C" LLVM_ATTRIBUTE_WEAK PassPluginLibraryInfo llvmGetPassPluginInfo() {
  return getLLGOLTOPluginInfo();
}
