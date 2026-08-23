#include "LLGOLTOPasses.h"

#include "llvm/Config/llvm-config.h"
#include "llvm/Passes/PassBuilder.h"
#include "llvm/Passes/PassPlugin.h"
#include "llvm/Support/Compiler.h"

using namespace llvm;

PassPluginLibraryInfo getLLGOLTOPluginInfo() {
  return {LLVM_PLUGIN_API_VERSION, "llgo-lto-plugin", LLVM_VERSION_STRING,
          [](PassBuilder &PB) {
            PB.registerPipelineParsingCallback(
                [](StringRef Name, ModulePassManager &MPM,
                   ArrayRef<PassBuilder::PipelineElement>) {
                  if (Name != llgo::LLGOPreGlobalDCEPassName)
                    return false;
                  llgo::addLLGOPreGlobalDCEPipeline(MPM);
                  return true;
                });

            PB.registerFullLinkTimeOptimizationEarlyEPCallback(
                [](ModulePassManager &MPM, OptimizationLevel) {
                  llgo::addLLGOPreGlobalDCEPipeline(MPM);
                });

            // ThinLTO optimizes each backend module independently through the
            // regular optimizer pipeline. Run after its scalar/IPO pipeline so
            // global slice loads and bounded loops have been simplified as far
            // as possible, then leave the recovered names on the call site for
            // LLGo's feedback planner to consume from .4.opt.bc.
            PB.registerOptimizerLastEPCallback(
                [](ModulePassManager &MPM, OptimizationLevel) {
                  llgo::addLLGOReflectMethodByNamePass(MPM);
                });
          }};
}

extern "C" LLVM_ATTRIBUTE_WEAK PassPluginLibraryInfo llvmGetPassPluginInfo() {
  return getLLGOLTOPluginInfo();
}
