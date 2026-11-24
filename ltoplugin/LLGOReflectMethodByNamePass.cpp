#include "LLGOLTOPasses.h"
#include "LLGOReflectStringAnalysis.h"
#include "LLGOStaticItabDevirt.h"

#include "llvm/ADT/DenseMap.h"
#include "llvm/ADT/SmallPtrSet.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/Analysis/ValueTracking.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/DataLayout.h"
#include "llvm/IR/IRBuilder.h"
#include "llvm/IR/InstrTypes.h"
#include "llvm/IR/Instructions.h"
#include "llvm/IR/IntrinsicInst.h"
#include "llvm/IR/Intrinsics.h"
#include "llvm/IR/Metadata.h"
#include "llvm/IR/Module.h"
#include "llvm/IR/PassManager.h"
#include "llvm/Support/raw_ostream.h"

#include <algorithm>
#include <cstdlib>
#include <optional>
#include <string>

using namespace llvm;

namespace {

static constexpr char ReflectMethodByNameCallAttr[] =
    "llgo.reflect.methodbyname";
static constexpr char ReflectMethodByNameValueKind[] = "value";
static constexpr char ReflectMethodByNameTypeKind[] = "type";
static constexpr char ReflectValueMethodTypeID[] = "go.method.value.reflect";
static constexpr char ReflectTypeMethodTypeID[] = "go.method.type.reflect";
static constexpr char ReflectValueMethodTypeIDPrefix[] =
    "go.method.value.reflect.";
static constexpr char ReflectTypeMethodTypeIDPrefix[] =
    "go.method.type.reflect.";
static constexpr unsigned MaxReflectFuncAnalysisDepth = 48;

bool isTypeCheckedLoad(CallBase *CB) {
  if (!CB)
    return false;
  auto *Callee = CB->getCalledFunction();
  return Callee && Callee->getIntrinsicID() == Intrinsic::type_checked_load;
}

bool isAssume(CallBase *CB) {
  if (!CB)
    return false;
  auto *Callee = CB->getCalledFunction();
  return Callee && Callee->getIntrinsicID() == Intrinsic::assume;
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

// Remove the original broad marker:
//
//   %r = call @llvm.type.checked.load(..., !"go.method.*.reflect")
//   %ok = extractvalue %r, 1
//   call @llvm.assume(%ok)
//
// The pointer result is intentionally unused in LLGo's GlobalDCE markers, so
// the pass only erases this pattern when all uses match the marker shape. If a
// future IR change gives the checked-load result a real data dependency, the
// pass fails loudly instead of silently changing semantics.
bool eraseGenericCheckedLoad(CallBase *CheckedLoad) {
  SmallVector<Instruction *, 4> ToErase;
  for (User *U : make_early_inc_range(CheckedLoad->users())) {
    auto *Extract = dyn_cast<ExtractValueInst>(U);
    if (!Extract || Extract->getNumIndices() != 1)
      return false;
    unsigned Index = Extract->getIndices()[0];
    if (Index == 0) {
      if (!Extract->use_empty())
        return false;
      ToErase.push_back(Extract);
      continue;
    }
    if (Index != 1)
      return false;
    for (User *EU : make_early_inc_range(Extract->users())) {
      auto *Assume = dyn_cast<CallBase>(EU);
      if (!isAssume(Assume))
        return false;
      ToErase.push_back(Assume);
    }
    ToErase.push_back(Extract);
  }

  for (Instruction *I : ToErase) {
    if (!I->use_empty())
      return false;
    I->eraseFromParent();
  }
  if (!CheckedLoad->use_empty())
    return false;
  CheckedLoad->eraseFromParent();
  return true;
}

// Replace one generic "some method may be reflected" marker with a disjunction
// of name-specific markers. The name-specific type IDs are emitted on ABI
// method slots by LLGo, so GlobalDCE can now keep only methods whose names are
// proven reachable by this call.
void insertMethodNameChecks(CallBase *GenericLoad, ArrayRef<std::string> Names,
                            StringRef Prefix) {
  IRBuilder<> B(GenericLoad);
  LLVMContext &Ctx = GenericLoad->getContext();
  SmallVector<Value *, 8> OKs;
  for (const std::string &Name : Names) {
    std::string TypeID = (Prefix + Name).str();
    Value *MDVal = MetadataAsValue::get(Ctx, MDString::get(Ctx, TypeID));
    CallInst *Checked = B.CreateIntrinsic(
        Intrinsic::type_checked_load, {},
        {GenericLoad->getArgOperand(0), GenericLoad->getArgOperand(1), MDVal});
    OKs.push_back(B.CreateExtractValue(Checked, {1}));
  }
  if (OKs.empty())
    return;

  Value *AnyOK = OKs.front();
  for (Value *OK : ArrayRef<Value *>(OKs).drop_front())
    AnyOK = B.CreateOr(AnyOK, OK);
  B.CreateIntrinsic(Intrinsic::assume, {}, {AnyOK});
}

bool isReflectMethodCheckedLoad(CallBase *CheckedLoad,
                                StringRef GenericTypeID) {
  auto TypeID = checkedLoadTypeID(CheckedLoad);
  return TypeID && *TypeID == GenericTypeID;
}

bool isReflectTypeMethodCheckedLoad(CallBase *CheckedLoad) {
  auto TypeID = checkedLoadTypeID(CheckedLoad);
  return TypeID && (*TypeID == ReflectTypeMethodTypeID ||
                    TypeID->starts_with(ReflectTypeMethodTypeIDPrefix));
}

std::optional<uint64_t> aggregateByteOffset(Type *Ty,
                                            ArrayRef<unsigned> Indices,
                                            const DataLayout &DL) {
  uint64_t Offset = 0;
  for (unsigned Index : Indices) {
    if (auto *StructTy = dyn_cast<StructType>(Ty)) {
      if (!StructTy->isSized() || Index >= StructTy->getNumElements())
        return std::nullopt;
      Offset += DL.getStructLayout(StructTy)->getElementOffset(Index);
      Ty = StructTy->getElementType(Index);
      continue;
    }
    if (auto *ArrayTy = dyn_cast<ArrayType>(Ty)) {
      if (Index >= ArrayTy->getNumElements())
        return std::nullopt;
      Type *ElemTy = ArrayTy->getElementType();
      Offset += Index * DL.getTypeAllocSize(ElemTy);
      Ty = ElemTy;
      continue;
    }
    return std::nullopt;
  }
  return Offset;
}

struct ReflectFuncStorage {
  AllocaInst *Base;
  uint64_t Offset;
  uint64_t Size;
};

// Recover the byte occupied by Method.Func's code pointer. The frontend emits
// the checked-load from an extract of a value loaded out of the Method result,
// so this walk is independent of the concrete Go ABI aggregate shape.
std::optional<ReflectFuncStorage> findReflectFuncStorage(CallBase *CheckedLoad,
                                                         const DataLayout &DL) {
  if (!isReflectTypeMethodCheckedLoad(CheckedLoad) || CheckedLoad->arg_empty())
    return std::nullopt;

  Value *V = CheckedLoad->getArgOperand(0);
  uint64_t Offset = 0;
  uint64_t Size = DL.getTypeStoreSize(V->getType());
  for (unsigned Depth = 0; Depth != MaxReflectFuncAnalysisDepth; ++Depth) {
    if (auto *Cast = dyn_cast<CastInst>(V)) {
      if (!isa<BitCastInst>(Cast) && !isa<AddrSpaceCastInst>(Cast))
        return std::nullopt;
      if (DL.getTypeStoreSize(Cast->getType()) !=
          DL.getTypeStoreSize(Cast->getOperand(0)->getType()))
        return std::nullopt;
      V = Cast->getOperand(0);
      continue;
    }
    if (auto *Extract = dyn_cast<ExtractElementInst>(V)) {
      auto *Index = dyn_cast<ConstantInt>(Extract->getIndexOperand());
      auto *VectorTy =
          dyn_cast<FixedVectorType>(Extract->getVectorOperand()->getType());
      if (!Index || Index->isNegative() || !VectorTy ||
          Index->getZExtValue() >= VectorTy->getNumElements())
        return std::nullopt;
      Offset += Index->getZExtValue() *
                DL.getTypeAllocSize(VectorTy->getElementType());
      V = Extract->getVectorOperand();
      continue;
    }
    if (auto *Extract = dyn_cast<ExtractValueInst>(V)) {
      auto ExtractOffset = aggregateByteOffset(
          Extract->getAggregateOperand()->getType(), Extract->getIndices(), DL);
      if (!ExtractOffset)
        return std::nullopt;
      Offset += *ExtractOffset;
      V = Extract->getAggregateOperand();
      continue;
    }
    if (auto *Freeze = dyn_cast<FreezeInst>(V)) {
      V = Freeze->getOperand(0);
      continue;
    }
    auto *Load = dyn_cast<LoadInst>(V);
    if (!Load || !Load->isSimple())
      return std::nullopt;

    uint64_t LoadSize = DL.getTypeStoreSize(Load->getType());
    if (Offset > LoadSize || Size > LoadSize - Offset)
      return std::nullopt;
    int64_t LoadOffset = 0;
    Value *Base = GetPointerBaseWithConstantOffset(Load->getPointerOperand(),
                                                   LoadOffset, DL);
    auto *Alloca =
        Base ? dyn_cast<AllocaInst>(Base->stripPointerCasts()) : nullptr;
    if (!Alloca || Alloca->isArrayAllocation() || LoadOffset < 0)
      return std::nullopt;
    uint64_t AllocaSize = DL.getTypeAllocSize(Alloca->getAllocatedType());
    uint64_t FinalOffset = static_cast<uint64_t>(LoadOffset);
    if (FinalOffset > AllocaSize || Offset > AllocaSize - FinalOffset ||
        Size > AllocaSize - FinalOffset - Offset)
      return std::nullopt;
    return ReflectFuncStorage{Alloca, FinalOffset + Offset, Size};
  }
  return std::nullopt;
}

bool regionsOverlap(uint64_t AOffset, uint64_t ASize, uint64_t BOffset,
                    uint64_t BSize) {
  if (ASize == 0 || BSize == 0)
    return false;
  if (AOffset < BOffset)
    return BOffset - AOffset < ASize;
  return AOffset - BOffset < BSize;
}

struct TrackedReflectFuncRegion {
  Value *Base;
  uint64_t Offset;
  uint64_t Size;
};

bool markReflectFuncRegionSeen(SmallVectorImpl<TrackedReflectFuncRegion> &Seen,
                               Value *Base, uint64_t Offset, uint64_t Size) {
  for (const TrackedReflectFuncRegion &Region : Seen) {
    if (Region.Base == Base && Region.Offset == Offset && Region.Size == Size)
      return false;
  }
  Seen.push_back({Base, Offset, Size});
  return true;
}

// Follow only the bytes containing Method.Func's code pointer. Loads turn a
// storage region into an SSA-value region; local stores turn it back into a
// storage region. Extracts of sibling Method fields do not overlap and are
// ignored. Any call, return, non-local store, pointer escape, non-constant
// offset, or unrecognized IR shape fails closed and keeps the checked-load.
class ReflectFuncUseAnalysis {
  const DataLayout &DL;
  SmallVector<TrackedReflectFuncRegion, 32> SeenStorage;
  SmallVector<TrackedReflectFuncRegion, 64> SeenValues;

  bool analyzeValue(Value *V, uint64_t Offset, uint64_t Size, unsigned Depth) {
    if (Depth > MaxReflectFuncAnalysisDepth)
      return false;
    if (!markReflectFuncRegionSeen(SeenValues, V, Offset, Size))
      return true;

    for (User *U : V->users()) {
      if (auto *Extract = dyn_cast<ExtractElementInst>(U)) {
        if (Extract->getVectorOperand() != V)
          return false;
        auto *Index = dyn_cast<ConstantInt>(Extract->getIndexOperand());
        auto *VectorTy = dyn_cast<FixedVectorType>(V->getType());
        if (!Index || Index->isNegative() || !VectorTy ||
            Index->getZExtValue() >= VectorTy->getNumElements())
          return false;
        uint64_t ElemOffset = Index->getZExtValue() *
                              DL.getTypeAllocSize(VectorTy->getElementType());
        uint64_t ElemSize = DL.getTypeStoreSize(Extract->getType());
        if (!regionsOverlap(Offset, Size, ElemOffset, ElemSize))
          continue;
        uint64_t Start = std::max(Offset, ElemOffset);
        uint64_t End = std::min(Offset + Size, ElemOffset + ElemSize);
        if (!analyzeValue(Extract, Start - ElemOffset, End - Start, Depth + 1))
          return false;
        continue;
      }

      if (auto *Extract = dyn_cast<ExtractValueInst>(U)) {
        if (Extract->getAggregateOperand() != V)
          return false;
        auto ElemOffset =
            aggregateByteOffset(V->getType(), Extract->getIndices(), DL);
        if (!ElemOffset)
          return false;
        uint64_t ElemSize = DL.getTypeStoreSize(Extract->getType());
        if (!regionsOverlap(Offset, Size, *ElemOffset, ElemSize))
          continue;
        uint64_t Start = std::max(Offset, *ElemOffset);
        uint64_t End = std::min(Offset + Size, *ElemOffset + ElemSize);
        if (!analyzeValue(Extract, Start - *ElemOffset, End - Start, Depth + 1))
          return false;
        continue;
      }

      if (auto *Insert = dyn_cast<InsertValueInst>(U)) {
        auto ElemOffset =
            aggregateByteOffset(Insert->getType(), Insert->getIndices(), DL);
        if (!ElemOffset)
          return false;
        uint64_t ElemSize =
            DL.getTypeStoreSize(Insert->getInsertedValueOperand()->getType());
        if (Insert->getAggregateOperand() == V) {
          if (!regionsOverlap(Offset, Size, *ElemOffset, ElemSize)) {
            if (!analyzeValue(Insert, Offset, Size, Depth + 1))
              return false;
          } else if (!(*ElemOffset <= Offset &&
                       Offset + Size <= *ElemOffset + ElemSize)) {
            return false;
          }
          continue;
        }
        if (Insert->getInsertedValueOperand() == V) {
          if (!analyzeValue(Insert, *ElemOffset + Offset, Size, Depth + 1))
            return false;
          continue;
        }
        return false;
      }

      if (auto *Store = dyn_cast<StoreInst>(U)) {
        if (Store->getValueOperand() != V || !Store->isSimple())
          return false;
        int64_t StoreOffset = 0;
        Value *StoreBase = GetPointerBaseWithConstantOffset(
            Store->getPointerOperand(), StoreOffset, DL);
        auto *Alloca =
            StoreBase ? dyn_cast<AllocaInst>(StoreBase->stripPointerCasts())
                      : nullptr;
        if (!Alloca || Alloca->isArrayAllocation() || StoreOffset < 0)
          return false;
        uint64_t DestOffset = static_cast<uint64_t>(StoreOffset);
        uint64_t AllocaSize = DL.getTypeAllocSize(Alloca->getAllocatedType());
        if (DestOffset > AllocaSize || Offset > AllocaSize - DestOffset ||
            Size > AllocaSize - DestOffset - Offset ||
            !analyzeStorage(Alloca, DestOffset + Offset, Size, Depth + 1))
          return false;
        continue;
      }

      if (auto *Cast = dyn_cast<CastInst>(U)) {
        if ((!isa<BitCastInst>(Cast) && !isa<AddrSpaceCastInst>(Cast)) ||
            DL.getTypeStoreSize(Cast->getType()) !=
                DL.getTypeStoreSize(V->getType()) ||
            !analyzeValue(Cast, Offset, Size, Depth + 1))
          return false;
        continue;
      }
      if (auto *Freeze = dyn_cast<FreezeInst>(U)) {
        if (!analyzeValue(Freeze, Offset, Size, Depth + 1))
          return false;
        continue;
      }
      if (auto *Phi = dyn_cast<PHINode>(U)) {
        if (!analyzeValue(Phi, Offset, Size, Depth + 1))
          return false;
        continue;
      }
      if (auto *Select = dyn_cast<SelectInst>(U)) {
        if ((Select->getTrueValue() != V && Select->getFalseValue() != V) ||
            !analyzeValue(Select, Offset, Size, Depth + 1))
          return false;
        continue;
      }

      if (auto *CB = dyn_cast<CallBase>(U)) {
        if (!CB->arg_empty() && CB->getArgOperand(0) == V &&
            isReflectTypeMethodCheckedLoad(CB))
          continue;
        return false;
      }
      return false;
    }
    return true;
  }

  bool analyzeStoragePointer(Value *Ptr, AllocaInst *Base, uint64_t Offset,
                             uint64_t Size,
                             SmallPtrSetImpl<Value *> &SeenPointers,
                             unsigned Depth) {
    if (Depth > MaxReflectFuncAnalysisDepth)
      return false;
    if (!SeenPointers.insert(Ptr).second)
      return true;

    for (User *U : Ptr->users()) {
      if (isa<GetElementPtrInst>(U) || isa<BitCastInst>(U) ||
          isa<AddrSpaceCastInst>(U)) {
        auto *Derived = cast<Value>(U);
        int64_t DerivedOffset = 0;
        Value *DerivedBase =
            GetPointerBaseWithConstantOffset(Derived, DerivedOffset, DL);
        if (!DerivedBase || DerivedBase->stripPointerCasts() != Base ||
            DerivedOffset < 0 ||
            !analyzeStoragePointer(Derived, Base, Offset, Size, SeenPointers,
                                   Depth + 1))
          return false;
        continue;
      }

      if (auto *Load = dyn_cast<LoadInst>(U)) {
        if (Load->getPointerOperand() != Ptr || !Load->isSimple())
          return false;
        int64_t LoadOffset = 0;
        Value *LoadBase = GetPointerBaseWithConstantOffset(
            Load->getPointerOperand(), LoadOffset, DL);
        if (!LoadBase || LoadBase->stripPointerCasts() != Base ||
            LoadOffset < 0)
          return false;
        uint64_t LoadStart = static_cast<uint64_t>(LoadOffset);
        uint64_t LoadSize = DL.getTypeStoreSize(Load->getType());
        if (!regionsOverlap(Offset, Size, LoadStart, LoadSize))
          continue;
        uint64_t Start = std::max(Offset, LoadStart);
        uint64_t End = std::min(Offset + Size, LoadStart + LoadSize);
        if (!analyzeValue(Load, Start - LoadStart, End - Start, Depth + 1))
          return false;
        continue;
      }

      if (auto *Store = dyn_cast<StoreInst>(U)) {
        if (Store->getPointerOperand() == Ptr &&
            Store->getValueOperand() != Ptr && Store->isSimple())
          continue;
        return false;
      }

      if (auto *II = dyn_cast<IntrinsicInst>(U)) {
        if (II->getIntrinsicID() == Intrinsic::lifetime_start ||
            II->getIntrinsicID() == Intrinsic::lifetime_end)
          continue;
      }

      if (auto *CB = dyn_cast<CallBase>(U)) {
        if (CB->getCalledOperand() == Ptr)
          return false;
        bool HasSRetUse = false;
        bool OnlySRetUses = true;
        for (unsigned I = 0, E = CB->arg_size(); I != E; ++I) {
          if (CB->getArgOperand(I) == Ptr) {
            if (!CB->paramHasAttr(I, Attribute::StructRet)) {
              OnlySRetUses = false;
              break;
            }
            HasSRetUse = true;
          }
        }
        if (HasSRetUse && OnlySRetUses)
          continue;
      }
      return false;
    }
    return true;
  }

  bool analyzeStorage(AllocaInst *Base, uint64_t Offset, uint64_t Size,
                      unsigned Depth) {
    if (Depth > MaxReflectFuncAnalysisDepth)
      return false;
    if (!markReflectFuncRegionSeen(SeenStorage, Base, Offset, Size))
      return true;
    SmallPtrSet<Value *, 32> SeenPointers;
    return analyzeStoragePointer(Base, Base, Offset, Size, SeenPointers,
                                 Depth + 1);
  }

public:
  explicit ReflectFuncUseAnalysis(const DataLayout &DL) : DL(DL) {}

  bool isUnobserved(const ReflectFuncStorage &Storage) {
    return analyzeStorage(Storage.Base, Storage.Offset, Storage.Size, 0);
  }
};

bool eraseUnobservedReflectTypeMethodFuncs(Module &M, const DataLayout &DL) {
  SmallVector<CallBase *, 16> CheckedLoads;
  for (Function &F : M) {
    for (BasicBlock &BB : F) {
      for (Instruction &I : BB) {
        auto *CB = dyn_cast<CallBase>(&I);
        if (isReflectTypeMethodCheckedLoad(CB))
          CheckedLoads.push_back(CB);
      }
    }
  }

  SmallVector<CallBase *, 16> ToErase;
  for (CallBase *CheckedLoad : CheckedLoads) {
    auto Storage = findReflectFuncStorage(CheckedLoad, DL);
    if (!Storage)
      continue;
    ReflectFuncUseAnalysis Analysis(DL);
    if (Analysis.isUnobserved(*Storage))
      ToErase.push_back(CheckedLoad);
  }

  for (CallBase *CheckedLoad : ToErase) {
    if (!eraseGenericCheckedLoad(CheckedLoad))
      report_fatal_error(
          "llgo-lto-plugin: failed to erase unobserved reflect Method.Func "
          "check");
  }
  if (!ToErase.empty() && std::getenv("LLGO_LTO_PLUGIN_VERBOSE"))
    errs() << "llgo-lto-plugin: erased " << ToErase.size()
           << " unobserved reflect Type Method.Func checks\n";
  return !ToErase.empty();
}

Value *getSRetArg(CallBase *CB) {
  for (unsigned I = 0, E = CB->arg_size(); I != E; ++I) {
    if (CB->paramHasAttr(I, Attribute::StructRet))
      return CB->getArgOperand(I);
  }
  return nullptr;
}

Value *getSRetStorage(CallBase *CB) {
  Value *SRet = getSRetArg(CB);
  return SRet ? SRet->stripPointerCasts() : nullptr;
}

void addReflectMethodCheckedLoad(CallBase *CheckedLoad, StringRef GenericTypeID,
                                 SmallPtrSetImpl<CallBase *> &SeenLoads,
                                 SmallVectorImpl<CallBase *> &Loads) {
  if (isReflectMethodCheckedLoad(CheckedLoad, GenericTypeID) &&
      SeenLoads.insert(CheckedLoad).second)
    Loads.push_back(CheckedLoad);
}

// MethodByName returns through an sret slot after C ABI lowering. LLGo then
// emits the generic reflect checked-load from the method value extracted out of
// that slot. This use-def walk is intentionally limited to value-preserving IR
// around that slot: GEPs into the sret object, loads from those fields, and
// scalar/aggregate extractions from the loaded values. It avoids a broad CFG
// scan while still covering both lowered shapes:
//
//   reflect.Value.MethodByName:
//     call @reflect.Value.MethodByName(sret %ret, ...)
//     %raw = load <2 x ptr>, ptr %ret
//     %fn = extractelement <2 x ptr> %raw, 1
//     call @llvm.type.checked.load(ptr %fn, ..., !"go.method.value.reflect")
//
//   reflect.Type.MethodByName:
//     call %methodByName(sret %ret, ...)
//     ... branch on the ok field ...
//     %raw = load <2 x ptr>, ptr gep(%ret, method.func)
//     %fn = extractelement <2 x ptr> %raw, 1
//     call @llvm.type.checked.load(ptr %fn, ..., !"go.method.type.reflect")
void collectSRetReflectMethodCheckedLoads(
    Value *V, StringRef GenericTypeID, SmallPtrSetImpl<Value *> &SeenValues,
    SmallPtrSetImpl<CallBase *> &SeenLoads, SmallVectorImpl<CallBase *> &Loads,
    unsigned Depth = 0) {
  if (!V || Depth > 16 || !SeenValues.insert(V).second)
    return;

  for (User *U : V->users()) {
    if (auto *CB = dyn_cast<CallBase>(U)) {
      addReflectMethodCheckedLoad(CB, GenericTypeID, SeenLoads, Loads);
      continue;
    }

    auto *I = dyn_cast<Instruction>(U);
    if (!I)
      continue;
    if (isa<GetElementPtrInst>(I) || isa<LoadInst>(I) ||
        isa<ExtractElementInst>(I) || isa<ExtractValueInst>(I) ||
        isa<CastInst>(I) || isa<SelectInst>(I) || isa<PHINode>(I)) {
      collectSRetReflectMethodCheckedLoads(I, GenericTypeID, SeenValues,
                                           SeenLoads, Loads, Depth + 1);
    }
  }
}

class LLGOLTOPreGlobalDCEPass : public PassInfoMixin<LLGOLTOPreGlobalDCEPass> {
public:
  PreservedAnalyses run(Module &M, ModuleAnalysisManager &) {
    const DataLayout &DL = M.getDataLayout();
    bool Changed = llgo::devirtualizeStaticItabCalls(M, DL);
    Changed |= llgo::eraseStaticItabTemplatePreservation(M);
    Changed |= eraseUnobservedReflectTypeMethodFuncs(M, DL);

    SmallVector<CallBase *, 16> Calls;
    for (Function &F : M) {
      for (BasicBlock &BB : F) {
        for (Instruction &I : BB) {
          auto *CB = dyn_cast<CallBase>(&I);
          if (!CB || !CB->hasFnAttr(ReflectMethodByNameCallAttr))
            continue;
          Calls.push_back(CB);
        }
      }
    }

    DenseMap<CallBase *, Value *> SRetStorageByCall;
    DenseMap<Value *, SmallVector<CallBase *, 4>> CallsBySRetStorage;
    for (CallBase *ReflectCall : Calls) {
      Value *SRetStorage = getSRetStorage(ReflectCall);
      SRetStorageByCall.insert({ReflectCall, SRetStorage});
      if (SRetStorage)
        CallsBySRetStorage[SRetStorage].push_back(ReflectCall);
    }

    SmallPtrSet<CallBase *, 16> ProcessedCalls;
    for (CallBase *ReflectCall : Calls) {
      if (!ProcessedCalls.insert(ReflectCall).second)
        continue;

      Attribute KindAttr = ReflectCall->getFnAttr(ReflectMethodByNameCallAttr);
      if (!KindAttr.isStringAttribute() || ReflectCall->arg_empty())
        continue;

      StringRef Prefix;
      StringRef GenericTypeID;
      StringRef Kind = KindAttr.getValueAsString();
      if (Kind == ReflectMethodByNameValueKind) {
        Prefix = ReflectValueMethodTypeIDPrefix;
        GenericTypeID = ReflectValueMethodTypeID;
      } else if (Kind == ReflectMethodByNameTypeKind) {
        Prefix = ReflectTypeMethodTypeIDPrefix;
        GenericTypeID = ReflectTypeMethodTypeID;
      } else {
        continue;
      }

      // Optimizations can leave several MethodByName calls sharing one sret
      // slot. The checked-load walk starts from the slot and sees the loads for
      // all of those calls. Refine them with the union of every provable name
      // instead of letting the first call claim all of them. If any name is
      // unknown, leave the generic markers intact so GlobalDCE remains
      // conservative.
      Value *SRetStorage = SRetStorageByCall.lookup(ReflectCall);
      SmallVector<CallBase *, 4> SharedCalls{ReflectCall};
      if (SRetStorage) {
        for (CallBase *Candidate :
             CallsBySRetStorage.find(SRetStorage)->second) {
          if (Candidate == ReflectCall)
            continue;
          Attribute CandidateKind =
              Candidate->getFnAttr(ReflectMethodByNameCallAttr);
          if (!CandidateKind.isStringAttribute() ||
              CandidateKind.getValueAsString() != Kind)
            continue;
          SharedCalls.push_back(Candidate);
          ProcessedCalls.insert(Candidate);
        }
      }

      SmallVector<std::string, 4> Names;
      bool KnownNames = true;
      // The bounded name budget applies to the combined group. Exceeding it
      // leaves every generic marker intact.
      for (CallBase *SharedCall : SharedCalls) {
        if (!llgo::collectReflectMethodNameArgs(SharedCall, DL, Names)) {
          KnownNames = false;
          break;
        }
      }
      if (!KnownNames || Names.empty())
        continue;

      SmallVector<CallBase *, 2> GenericLoads;
      SmallPtrSet<CallBase *, 4> SeenLoads;
      SmallPtrSet<Value *, 16> SeenValues;
      collectSRetReflectMethodCheckedLoads(SRetStorage, GenericTypeID,
                                           SeenValues, SeenLoads, GenericLoads);
      for (CallBase *GenericLoad : GenericLoads) {
        insertMethodNameChecks(GenericLoad, Names, Prefix);
        if (!eraseGenericCheckedLoad(GenericLoad))
          report_fatal_error(
              "llgo-lto-plugin: failed to erase generic MethodByName check");
        Changed = true;
      }

      if (!GenericLoads.empty() && std::getenv("LLGO_LTO_PLUGIN_VERBOSE")) {
        errs() << "llgo-lto-plugin: refined " << Kind << " to";
        for (const std::string &Name : Names)
          errs() << " " << Name;
        errs() << "\n";
      }
    }

    return Changed ? PreservedAnalyses::none() : PreservedAnalyses::all();
  }
};

} // namespace

namespace llgo {

void addLLGOReflectMethodByNamePass(ModulePassManager &MPM) {
  MPM.addPass(LLGOLTOPreGlobalDCEPass());
}

} // namespace llgo
