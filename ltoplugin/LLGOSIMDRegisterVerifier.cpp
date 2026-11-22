#include "LLGOLTOPasses.h"
#include "llvm/ADT/SmallPtrSet.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/Instructions.h"
#include "llvm/IR/Module.h"

using namespace llvm;

namespace {
unsigned registerWidth(const Function &F) {
  Attribute Base = F.getFnAttribute("llgo.fmv.register-baseline");
  StringRef Floor = Base.isValid() ? Base.getValueAsString() : StringRef();
  unsigned Width = Floor == "v4" ? 512 : Floor == "v3" ? 256 : 0;
  SmallVector<StringRef> Features;
  if (Attribute A = F.getFnAttribute("target-features"); A.isValid())
    A.getValueAsString().split(Features, ',');
  for (StringRef Feature : Features) {
    if (Feature == "+avx512f")
      Width = 512;
    else if (Feature == "+avx2")
      Width = std::max(Width, 256u);
    else if (Feature == "-avx512f")
      Width = std::min(Width, 256u);
    else if (Feature == "-avx" || Feature == "-avx2")
      Width = 0;
  }
  return Width;
}

} // namespace

void llgo::verifySIMDRegisterCalls(const Module &M) {
  for (const Function &Target : M) {
    Attribute Tag = Target.getFnAttribute("llgo.fmv.register-width");
    if (!Tag.isValid())
      continue;
    unsigned Required = 0;
    if (Tag.getValueAsString().getAsInteger(10, Required) ||
        (Required != 256 && Required != 512) ||
        registerWidth(Target) < Required) {
      M.getContext().emitError("invalid SIMD register entry " +
                               Target.getName());
      continue;
    }
    SmallVector<const Value *> Work{&Target};
    SmallPtrSet<const Value *, 8> Seen;
    while (!Work.empty()) {
      const Value *V = Work.pop_back_val();
      if (!Seen.insert(V).second)
        continue;
      for (const User *U : V->users()) {
        if (isa<GlobalAlias>(U) ||
            (isa<ConstantExpr>(U) &&
             cast<ConstantExpr>(U)->getOpcode() == Instruction::BitCast)) {
          Work.push_back(U);
          continue;
        }
        const auto *CB = dyn_cast<CallBase>(U);
        if (!CB || CB->getCalledOperand() != V)
          continue;
        const Function *Caller = CB->getFunction();
        if (registerWidth(*Caller) < Required)
          M.getContext().emitError("SIMD register call " + Caller->getName() +
                                   " -> " + Target.getName() + " requires " +
                                   Twine(Required) + "-bit features");
      }
    }
  }
}

namespace {
class SIMDRegisterVerifier : public PassInfoMixin<SIMDRegisterVerifier> {
public:
  static bool isRequired() { return true; }
  PreservedAnalyses run(Module &M, ModuleAnalysisManager &) {
    llgo::verifySIMDRegisterCalls(M);
    return PreservedAnalyses::all();
  }
};
} // namespace

void llgo::addSIMDRegisterVerifier(ModulePassManager &MPM) {
  MPM.addPass(SIMDRegisterVerifier());
}
