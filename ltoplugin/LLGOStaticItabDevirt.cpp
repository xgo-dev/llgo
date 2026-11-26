#include "LLGOStaticItabDevirt.h"
#include "LLGOReflectStringAnalysis.h"

#include "llvm/ADT/DenseMap.h"
#include "llvm/ADT/SmallPtrSet.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/Analysis/ValueTracking.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/IRBuilder.h"
#include "llvm/IR/Instructions.h"
#include "llvm/IR/Intrinsics.h"
#include "llvm/IR/Metadata.h"
#include "llvm/Support/raw_ostream.h"
#include "llvm/Transforms/Utils/ModuleUtils.h"

#include <cstdlib>
#include <optional>

using namespace llvm;

namespace {

static constexpr char ReflectValueMethodTypeID[] = "go.method.value.reflect";
static constexpr char ReflectTypeMethodTypeID[] = "go.method.type.reflect";
static constexpr char ReflectValueMethodTypeIDPrefix[] =
    "go.method.value.reflect.";
static constexpr char ReflectTypeMethodTypeIDPrefix[] =
    "go.method.type.reflect.";
static constexpr char StaticItabSlotMetadata[] = "llgo.static.itab.slot";

bool isTypeCheckedLoad(CallBase *CB) {
  if (!CB)
    return false;
  auto *Callee = CB->getCalledFunction();
  return Callee && Callee->getIntrinsicID() == Intrinsic::type_checked_load;
}

std::optional<StringRef> checkedLoadTypeID(CallBase *CheckedLoad) {
  if (!isTypeCheckedLoad(CheckedLoad) || CheckedLoad->arg_size() < 3)
    return std::nullopt;
  auto *MDValue = dyn_cast<MetadataAsValue>(CheckedLoad->getArgOperand(2));
  if (!MDValue)
    return std::nullopt;
  auto *TypeID = dyn_cast<MDString>(MDValue->getMetadata());
  if (!TypeID)
    return std::nullopt;
  return TypeID->getString();
}

std::optional<StringRef> typeTestTypeID(CallBase *TypeTest) {
  if (!TypeTest || TypeTest->arg_size() < 2)
    return std::nullopt;
  auto *Callee = TypeTest->getCalledFunction();
  if (!Callee || Callee->getIntrinsicID() != Intrinsic::type_test)
    return std::nullopt;
  auto *MDValue = dyn_cast<MetadataAsValue>(TypeTest->getArgOperand(1));
  auto *TypeID = MDValue ? dyn_cast<MDString>(MDValue->getMetadata()) : nullptr;
  return TypeID ? std::optional<StringRef>(TypeID->getString()) : std::nullopt;
}

std::optional<uint64_t> checkedLoadOffset(CallBase *CheckedLoad) {
  if (!CheckedLoad || CheckedLoad->arg_size() < 2)
    return std::nullopt;
  auto *Offset = dyn_cast<ConstantInt>(CheckedLoad->getArgOperand(1));
  if (!Offset || Offset->isNegative())
    return std::nullopt;
  return Offset->getZExtValue();
}

struct StaticItabSlot {
  Constant *Target;
};

std::optional<StaticItabSlot> resolveStaticItabSlot(Value *Pointer,
                                                    uint64_t ExtraOffset,
                                                    StringRef TypeID,
                                                    const DataLayout &DL) {
  int64_t PointerOffset = 0;
  Value *Base = GetPointerBaseWithConstantOffset(Pointer, PointerOffset, DL);
  if (!Base || PointerOffset < 0)
    return std::nullopt;
  auto *GV = dyn_cast<GlobalVariable>(Base->stripPointerCasts());
  if (!GV || !GV->getName().starts_with("_llgo_itab$") ||
      !llgo::canReadGlobalInitializer(GV))
    return std::nullopt;
  uint64_t Offset = static_cast<uint64_t>(PointerOffset) + ExtraOffset;

  bool HasMatchingSlot = false;
  unsigned SlotKind = GV->getContext().getMDKindID(StaticItabSlotMetadata);
  SmallVector<std::pair<unsigned, MDNode *>, 16> Metadata;
  GV->getAllMetadata(Metadata);
  for (auto [Kind, Node] : Metadata) {
    if (Kind != SlotKind || Node->getNumOperands() < 2)
      continue;
    auto *OffsetMD = dyn_cast<ConstantAsMetadata>(Node->getOperand(0));
    auto *ID = dyn_cast<MDString>(Node->getOperand(1));
    auto *OffsetC =
        OffsetMD ? dyn_cast<ConstantInt>(OffsetMD->getValue()) : nullptr;
    if (OffsetC && ID && OffsetC->getZExtValue() == Offset &&
        ID->getString() == TypeID) {
      HasMatchingSlot = true;
      break;
    }
  }
  if (!HasMatchingSlot)
    return std::nullopt;

  Constant *Target = llgo::constantAtByteOffset(GV->getInitializer(), Offset,
                                                Pointer->getType(), DL);
  if (!Target || Target->isNullValue())
    return std::nullopt;
  return StaticItabSlot{Target};
}

bool isNewItabCall(CallBase *CB) {
  if (!CB)
    return false;
  auto *Callee =
      dyn_cast<Function>(CB->getCalledOperand()->stripPointerCasts());
  return Callee && Callee->getName().ends_with("runtime.NewItab");
}

GlobalVariable *findStaticItabTemplate(Module &M, CallBase *NewItab) {
  if (!isNewItabCall(NewItab) || NewItab->arg_size() < 2)
    return nullptr;
  Value *Interface = NewItab->getArgOperand(0)->stripPointerCasts();
  Value *Concrete = NewItab->getArgOperand(1)->stripPointerCasts();
  for (GlobalVariable &GV : M.globals()) {
    if (!GV.getName().starts_with("_llgo_itab$") ||
        !llgo::canReadGlobalInitializer(&GV))
      continue;
    Constant *Init = GV.getInitializer();
    Constant *TemplateInterface = Init->getAggregateElement(0u);
    Constant *TemplateConcrete = Init->getAggregateElement(1u);
    if (!TemplateInterface || !TemplateConcrete)
      continue;
    if (TemplateInterface->stripPointerCasts() == Interface &&
        TemplateConcrete->stripPointerCasts() == Concrete)
      return &GV;
  }
  return nullptr;
}

std::optional<StaticItabSlot> resolveNewItabSlot(Module &M, CallBase *NewItab,
                                                 uint64_t Offset,
                                                 StringRef TypeID,
                                                 const DataLayout &DL) {
  GlobalVariable *Template = findStaticItabTemplate(M, NewItab);
  if (!Template)
    return std::nullopt;
  return resolveStaticItabSlot(Template, Offset, TypeID, DL);
}

Constant *collectStaticItabArgumentTarget(Argument *Arg,
                                          ArrayRef<unsigned> Indices,
                                          uint64_t PointerOffset,
                                          uint64_t LoadOffset, StringRef TypeID,
                                          const DataLayout &DL) {
  Function *F = Arg->getParent();
  if (!F->hasLocalLinkage())
    return nullptr;
  Constant *Target = nullptr;
  bool SawCall = false;
  for (User *U : F->users()) {
    auto *CB = dyn_cast<CallBase>(U);
    if (!CB || CB->getCalledOperand()->stripPointerCasts() != F ||
        Arg->getArgNo() >= CB->arg_size())
      return nullptr;
    Value *Actual = CB->getArgOperand(Arg->getArgNo());
    if (!Indices.empty()) {
      // Aggregate ABIs keep the interface's itab in a struct argument.
      // Recover only a proven field value; an unknown caller must retain
      // the checked load and its broad method root.
      Actual = FindInsertedValue(Actual, Indices);
      if (!Actual)
        return nullptr;
    }
    auto Slot =
        resolveStaticItabSlot(Actual, PointerOffset + LoadOffset, TypeID, DL);
    CallBase *NewItab = nullptr;
    if (!Slot) {
      NewItab = dyn_cast<CallBase>(Actual->stripPointerCasts());
      Slot = resolveNewItabSlot(*F->getParent(), NewItab,
                                PointerOffset + LoadOffset, TypeID, DL);
    }
    if (!Slot)
      return nullptr;
    Constant *Candidate = dyn_cast<Constant>(Slot->Target->stripPointerCasts());
    if (!Candidate)
      return nullptr;
    if (Target && Candidate != Target)
      return nullptr;
    Target = Candidate;
    SawCall = true;
  }
  return SawCall ? Target : nullptr;
}

} // namespace

namespace llgo {

bool devirtualizeStaticItabCalls(Module &M, const DataLayout &DL) {
  struct Replacement {
    CallBase *CheckedLoad;
    Constant *Target;
  };
  SmallVector<Replacement, 32> Candidates;
  DenseMap<StringRef, unsigned> CheckedLoadsByTypeID;
  DenseMap<StringRef, unsigned> TypeTestsByTypeID;
  DenseMap<StringRef, unsigned> CandidatesByTypeID;
  DenseMap<StringRef, SmallVector<CallBase *, 4>> CheckedLoadSitesByTypeID;
  SmallPtrSet<CallBase *, 32> CandidateLoads;
  for (Function &F : M) {
    for (BasicBlock &BB : F) {
      for (Instruction &I : BB) {
        auto *CheckedLoad = dyn_cast<CallBase>(&I);
        if (auto TypeID = typeTestTypeID(CheckedLoad)) {
          if (TypeID->starts_with("go.method.") &&
              *TypeID != ReflectValueMethodTypeID &&
              !TypeID->starts_with(ReflectValueMethodTypeIDPrefix) &&
              *TypeID != ReflectTypeMethodTypeID &&
              !TypeID->starts_with(ReflectTypeMethodTypeIDPrefix))
            ++TypeTestsByTypeID[*TypeID];
          continue;
        }
        auto TypeID = checkedLoadTypeID(CheckedLoad);
        auto LoadOffset = checkedLoadOffset(CheckedLoad);
        bool IsReflect =
            TypeID && (*TypeID == ReflectValueMethodTypeID ||
                       TypeID->starts_with(ReflectValueMethodTypeIDPrefix) ||
                       *TypeID == ReflectTypeMethodTypeID ||
                       TypeID->starts_with(ReflectTypeMethodTypeIDPrefix));
        if (!TypeID || !TypeID->starts_with("go.method.") || IsReflect)
          continue;
        ++CheckedLoadsByTypeID[*TypeID];
        CheckedLoadSitesByTypeID[*TypeID].push_back(CheckedLoad);
        if (!LoadOffset || CheckedLoad->arg_empty())
          continue;
        auto *ResultTy = dyn_cast<StructType>(CheckedLoad->getType());
        if (!ResultTy || ResultTy->getNumElements() != 2)
          continue;

        Constant *Target = nullptr;
        if (auto Slot = resolveStaticItabSlot(CheckedLoad->getArgOperand(0),
                                              *LoadOffset, *TypeID, DL)) {
          Target = Slot->Target;
        } else {
          int64_t PointerOffset = 0;
          Value *Base = GetPointerBaseWithConstantOffset(
              CheckedLoad->getArgOperand(0), PointerOffset, DL);
          auto *Arg = Base && PointerOffset >= 0
                          ? dyn_cast<Argument>(Base->stripPointerCasts())
                          : nullptr;
          ArrayRef<unsigned> Indices;
          if (!Arg && Base && PointerOffset >= 0) {
            if (auto *Extract = dyn_cast<ExtractValueInst>(Base)) {
              Arg = dyn_cast<Argument>(Extract->getAggregateOperand());
              Indices = Extract->getIndices();
            }
          }
          if (Arg) {
            Target = collectStaticItabArgumentTarget(
                Arg, Indices, static_cast<uint64_t>(PointerOffset), *LoadOffset,
                *TypeID, DL);
          } else if (auto *NewItab =
                         Base ? dyn_cast<CallBase>(Base->stripPointerCasts())
                              : nullptr) {
            auto Slot =
                PointerOffset >= 0
                    ? resolveNewItabSlot(M, NewItab,
                                         static_cast<uint64_t>(PointerOffset) +
                                             *LoadOffset,
                                         *TypeID, DL)
                    : std::nullopt;
            if (Slot) {
              Target = Slot->Target;
            }
          }
          if (Target) {
            Candidates.push_back({CheckedLoad, Target});
            ++CandidatesByTypeID[*TypeID];
            CandidateLoads.insert(CheckedLoad);
          }
          continue;
        }
        if (Target) {
          Candidates.push_back({CheckedLoad, Target});
          ++CandidatesByTypeID[*TypeID];
          CandidateLoads.insert(CheckedLoad);
        }
      }
    }
  }

  // Devirtualize each proven NewItab context independently. A remaining
  // checked load or type test with the same signature-wide type ID still keeps
  // the broad method root alive, but that must not prevent a known call site
  // from becoming a direct call. Once the final dynamic use disappears, the
  // type ID loses all IR users and the broad root goes away naturally.
  if (std::getenv("LLGO_LTO_PLUGIN_VERBOSE")) {
    for (auto [TypeID, Count] : CandidatesByTypeID) {
      unsigned Total = CheckedLoadsByTypeID.lookup(TypeID);
      unsigned TypeTests = TypeTestsByTypeID.lookup(TypeID);
      if (Count == Total && TypeTests == 0)
        errs() << "llgo-lto-plugin: removed method root " << TypeID << " ("
               << Count << " checked loads)\n";
      else
        errs() << "llgo-lto-plugin: kept method root " << TypeID
               << " (resolved " << Count << " of " << Total
               << " checked loads, " << TypeTests << " type tests)\n";
      if ((Count != Total || TypeTests != 0) &&
          (Total <= 16 || Count + 2 >= Total)) {
        for (CallBase *CheckedLoad : CheckedLoadSitesByTypeID[TypeID]) {
          if (CandidateLoads.contains(CheckedLoad))
            continue;
          errs() << "  unresolved in " << CheckedLoad->getFunction()->getName()
                 << ": base=" << *CheckedLoad->getArgOperand(0) << '\n';
        }
      }
    }
  }

  for (Replacement &R : Candidates) {
    CallBase *CheckedLoad = R.CheckedLoad;
    Constant *Target = R.Target;
    auto *ResultTy = cast<StructType>(CheckedLoad->getType());
    IRBuilder<> B(CheckedLoad);
    Value *Result = PoisonValue::get(ResultTy);
    Result = B.CreateInsertValue(Result, Target, {0});
    Result = B.CreateInsertValue(Result, B.getTrue(), {1});
    CheckedLoad->replaceAllUsesWith(Result);
    CheckedLoad->eraseFromParent();
  }
  if (!Candidates.empty() && std::getenv("LLGO_LTO_PLUGIN_VERBOSE"))
    errs() << "llgo-lto-plugin: devirtualized " << Candidates.size()
           << " static itab calls\n";
  return !Candidates.empty();
}

bool eraseStaticItabTemplatePreservation(Module &M) {
  bool Changed = false;
  removeFromUsedLists(M, [&](Constant *C) {
    auto *GV = dyn_cast<GlobalVariable>(C->stripPointerCasts());
    bool Remove = GV && GV->getName().starts_with("_llgo_itab$");
    Changed |= Remove;
    return Remove;
  });
  // Type metadata is intentionally not an IR use.  Once compiler.used has
  // been removed, erase dormant templates before GlobalDCE builds its type-id
  // candidate sets; otherwise an unreachable template can still make its
  // method look like a possible virtual-call target.
  for (GlobalVariable &GV : llvm::make_early_inc_range(M.globals())) {
    GV.removeDeadConstantUsers();
    if (GV.getName().starts_with("_llgo_itab$") && GV.use_empty()) {
      GV.eraseFromParent();
      Changed = true;
    }
  }
  return Changed;
}

} // namespace llgo
