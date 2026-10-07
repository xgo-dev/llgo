// Copyright 2026 The XGo Authors. Licensed under the Apache License,
// Version 2.0.
#include "fmv.h"
#include "llvm/ADT/DenseMap.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/ADT/StringExtras.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/DebugInfoMetadata.h"
#include "llvm/IR/IRBuilder.h"
#include "llvm/IR/InlineAsm.h"
#include "llvm/IR/InstIterator.h"
#include "llvm/IR/Module.h"
#include "llvm/Support/xxhash.h"
#include "llvm/TargetParser/Triple.h"
#include "llvm/Transforms/Utils/Cloning.h"
#include "llvm/Transforms/Utils/Local.h"
#include <cstdlib>
#include <cstring>
#include <set>

using namespace llvm;

namespace {
constexpr StringLiteral Entry = "llgo.fmv.avx2-entry";
constexpr StringLiteral Done = "llgo.fmv.processed";
constexpr StringLiteral Suffix = ".__llgo_fmv_avx2";

bool isQuery(const Function *F) {
  if (!F ||
      F->getFnAttribute("llgo.cpu.query").getValueAsString() != "x86.avx2" ||
      !F->getReturnType()->isIntegerTy(1) || F->isVarArg())
    return false;
  // Before ABI lowering the Go method still has its zero-sized receiver.
  return F->arg_empty() ||
         (F->arg_size() == 1 && F->getArg(0)->getType()->isStructTy() &&
          F->getArg(0)->getType()->isEmptyTy());
}

// The first implementation preserves the existing SIMD128 ABI. Wider vectors
// and aggregate ABIs need their own target-independent boundary lowering.
bool supportedType(Type *T) {
  if (auto *V = dyn_cast<FixedVectorType>(T))
    return V->getPrimitiveSizeInBits().getFixedValue() <= 128;
  return T->isVoidTy() || T->isIntegerTy() || T->isFloatingPointTy() ||
         T->isPointerTy();
}

bool supported(const Function &F) {
  if (F.isVarArg() || F.hasFnAttribute(Attribute::Naked) ||
      F.hasFnAttribute(Attribute::ReturnsTwice) ||
      !supportedType(F.getReturnType()))
    return false;
  for (auto &A : F.args())
    if (!supportedType(A.getType()))
      return false;
  return true;
}

std::string hexID(uint64_t ID) {
  std::string S = utohexstr(ID, true);
  return std::string(16 - S.size(), '0') + S;
}

// Runtime tables intentionally use strings, so CloneFunction cannot update
// them. PC-site IDs must also follow a clone if the original is eliminated.
// Keep these layouts in sync with ssa/funcinfo.go:
// funcinfo = {version, symbol, Go name, file, line, column[, flags]}
// pcline   = {version, id, symbol, file, line, column}
void cloneSourceInfo(Module &M, Function &F, Function &V,
                     std::set<uint64_t> &UsedIDs) {
  LLVMContext &C = M.getContext();
  // CloneFunction gives the implementation its own subprogram. Keep the Go
  // display name, but let debuggers resolve the clone's actual linker symbol.
  if (auto *SP = V.getSubprogram())
    SP->replaceLinkageName(MDString::get(C, V.getName()));
  if (auto *Info = M.getNamedMetadata("llgo.funcinfo")) {
    unsigned N = Info->getNumOperands();
    for (unsigned I = 0; I != N; ++I) {
      MDNode *Row = Info->getOperand(I);
      if (Row->getNumOperands() < 6)
        continue;
      auto *Name = dyn_cast<MDString>(Row->getOperand(1));
      if (!Name || Name->getString() != F.getName())
        continue;
      SmallVector<Metadata *> Fields(Row->op_begin(), Row->op_end());
      Fields[1] = MDString::get(C, V.getName());
      Info->addOperand(MDNode::get(C, Fields));
    }
  }
  DenseMap<uint64_t, uint64_t> IDs;
  if (auto *Info = M.getNamedMetadata("llgo.pcline")) {
    unsigned N = Info->getNumOperands();
    for (unsigned I = 0; I != N; ++I) {
      MDNode *Row = Info->getOperand(I);
      if (Row->getNumOperands() != 6)
        continue;
      auto *Name = dyn_cast<MDString>(Row->getOperand(2));
      auto *ID = mdconst::dyn_extract<ConstantInt>(Row->getOperand(1));
      if (!Name || Name->getString() != F.getName() || !ID)
        continue;
      uint64_t Old = ID->getZExtValue();
      uint64_t New = xxHash64((V.getName() + ":" + hexID(Old)).str());
      while (!New || !UsedIDs.insert(New).second)
        ++New;
      IDs[Old] = New;
      SmallVector<Metadata *> Fields(Row->op_begin(), Row->op_end());
      Fields[1] = ConstantAsMetadata::get(ConstantInt::get(ID->getType(), New));
      Fields[2] = MDString::get(C, V.getName());
      Info->addOperand(MDNode::get(C, Fields));
    }
  }
  for (auto &I : instructions(V)) {
    auto *Call = dyn_cast<CallBase>(&I);
    auto *Asm = Call ? dyn_cast<InlineAsm>(Call->getCalledOperand()) : nullptr;
    if (!Asm || !Asm->getAsmString().contains("__llgo_pcsite_"))
      continue;
    std::string Text = Asm->getAsmString().str();
    for (auto ID : IDs) {
      std::string Old = hexID(ID.first), New = hexID(ID.second);
      size_t Pos = 0;
      while ((Pos = Text.find(Old, Pos)) != std::string::npos) {
        Text.replace(Pos, Old.size(), New);
        Pos += New.size();
      }
    }
    Call->setCalledOperand(
        InlineAsm::get(Asm->getFunctionType(), Text, Asm->getConstraintString(),
                       Asm->hasSideEffects(), Asm->isAlignStack(),
                       Asm->getDialect(), Asm->canThrow()));
  }
}

void specialize(Function &F, const DenseMap<Function *, Function *> &Variants) {
  SmallVector<CallInst *> Queries;
  for (auto &I : instructions(F)) {
    auto *Call = dyn_cast<CallBase>(&I);
    if (!Call)
      continue;
    Function *Callee = Call->getCalledFunction();
    if (isQuery(Callee)) {
      if (auto *CI = dyn_cast<CallInst>(Call))
        Queries.push_back(CI);
    } else if (auto It = Variants.find(Callee); It != Variants.end()) {
      Call->setCalledFunction(It->second);
    }
  }
  for (CallInst *Q : Queries) {
    Q->replaceAllUsesWith(ConstantInt::getTrue(F.getContext()));
    Q->eraseFromParent();
  }
  // This is a legality transform, including at O0. Do not rely on an optional
  // optimization pipeline to remove feature-disabled branches.
  bool Changed;
  do {
    Changed = false;
    for (auto &BB : F) {
      Changed |= SimplifyInstructionsInBlock(&BB);
      Changed |= ConstantFoldTerminator(&BB, true);
    }
    Changed |= removeUnreachableBlocks(F);
  } while (Changed);
}

void dispatch(Function &F, Function &V, Function &Query) {
  LLVMContext &C = F.getContext();
  BasicBlock *Baseline = &F.getEntryBlock();
  BasicBlock *EntryBB = BasicBlock::Create(C, "fmv.entry", &F, Baseline);
  BasicBlock *FastBB = BasicBlock::Create(C, "fmv.avx2", &F, Baseline);
  IRBuilder<> B(EntryBB);
  if (F.getSubprogram())
    B.SetCurrentDebugLocation(DILocation::get(C, 0, 0, F.getSubprogram()));
  SmallVector<Value *> QueryArgs;
  for (auto &A : Query.args())
    QueryArgs.push_back(Constant::getNullValue(A.getType()));
  auto *Test = B.CreateCall(&Query, QueryArgs);
  Test->setCallingConv(Query.getCallingConv());
  B.CreateCondBr(Test, FastBB, Baseline);
  B.SetInsertPoint(FastBB);
  SmallVector<Value *> Args;
  for (auto &A : F.args())
    Args.push_back(&A);
  auto *Call = B.CreateCall(&V, Args);
  Call->setCallingConv(F.getCallingConv());
  // Forward the parameter/return ABI, not function-only optimization policy.
  Call->setAttributes(F.getAttributes().removeFnAttributes(C));
  Call->setTailCallKind(CallInst::TCK_MustTail);
  if (F.getReturnType()->isVoidTy())
    B.CreateRetVoid();
  else
    B.CreateRet(Call);
}

std::string run(Module &M) {
  if (M.getTargetTriple().getArch() != Triple::x86_64)
    return "";
  SmallVector<Function *> Originals;
  DenseMap<Function *, Function *> Roots, Variants;
  // Discover roots from direct uses of trusted queries. Ordinary packages
  // without SIMD must not pay for a scan of every instruction in the module.
  for (auto &F : M) {
    if (!isQuery(&F))
      continue;
    for (User *U : F.users())
      if (auto *Call = dyn_cast<CallInst>(U))
        if (Call->getCalledFunction() == &F)
          Roots[Call->getFunction()] = &F;
  }
  for (auto &F : M) {
    if (F.hasFnAttribute(Done) || !supported(F))
      continue;
    Function *Query = Roots.lookup(&F);
    if (!F.hasFnAttribute(Entry) && !Query)
      continue;
    if (M.getNamedValue((F.getName() + Suffix).str()))
      return "SIMD FMV symbol collision: " + (F.getName() + Suffix).str();
    Originals.push_back(&F);
  }
  if (Originals.empty())
    return "";
  std::set<uint64_t> UsedIDs;
  if (auto *Info = M.getNamedMetadata("llgo.pcline"))
    for (auto *Row : Info->operands())
      if (Row->getNumOperands() == 6)
        if (auto *ID = mdconst::dyn_extract<ConstantInt>(Row->getOperand(1)))
          UsedIDs.insert(ID->getZExtValue());
  for (Function *F : Originals) {
    Function *V;
    if (F->isDeclaration()) {
      V = Function::Create(F->getFunctionType(), F->getLinkage(),
                           F->getAddressSpace(), F->getName() + Suffix, &M);
      V->copyAttributesFrom(F);
    } else {
      ValueToValueMapTy Map;
      V = CloneFunction(F, Map);
      V->setName(F->getName() + Suffix);
      cloneSourceInfo(M, *F, *V, UsedIDs);
    }
    std::string Features =
        F->getFnAttribute("target-features").getValueAsString().str();
    if (!Features.empty())
      Features += ",";
    V->addFnAttr("target-features", Features + "+avx,+avx2");
    V->removeFnAttr(Entry);
    V->addFnAttr(Done);
    F->addFnAttr(Done);
    Variants[F] = V;
  }
  for (Function *F : Originals) {
    if (F->isDeclaration())
      continue;
    Function *V = Variants.lookup(F);
    specialize(*V, Variants);
    if (Function *Query = Roots.lookup(F))
      dispatch(*F, *V, *Query);
  }
  return "";
}
} // namespace

extern "C" char *llgoRunSIMDFMV(void *ModulePtr) {
  std::string Error = run(*static_cast<Module *>(ModulePtr));
  if (Error.empty())
    return nullptr;
  char *Result = static_cast<char *>(std::malloc(Error.size() + 1));
  if (!Result)
    std::abort();
  std::memcpy(Result, Error.c_str(), Error.size() + 1);
  return Result;
}
