#include "LLGOLTOPasses.h"

#include "llvm/ADT/DenseMap.h"
#include "llvm/ADT/SmallPtrSet.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/Analysis/ConstantFolding.h"
#include "llvm/Analysis/ValueTracking.h"
#include "llvm/Config/llvm-config.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/DataLayout.h"
#include "llvm/IR/Dominators.h"
#include "llvm/IR/IRBuilder.h"
#include "llvm/IR/InstrTypes.h"
#include "llvm/IR/Instructions.h"
#include "llvm/IR/IntrinsicInst.h"
#include "llvm/IR/Intrinsics.h"
#include "llvm/IR/Metadata.h"
#include "llvm/IR/Module.h"
#include "llvm/IR/Operator.h"
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
static constexpr char ReflectMethodByNameArgAttr[] =
    "llgo.reflect.methodbyname.name";
static constexpr char ReflectMethodByNameNamesAttr[] =
    "llgo.reflect.methodbyname.names";
static constexpr char ReflectMethodByNameValueKind[] = "value";
static constexpr char ReflectMethodByNameTypeKind[] = "type";
static constexpr char ReflectValueMethodTypeID[] = "go.method.value.reflect";
static constexpr char ReflectTypeMethodTypeID[] = "go.method.type.reflect";
static constexpr char ReflectValueMethodTypeIDPrefix[] =
    "go.method.value.reflect.";
static constexpr char ReflectTypeMethodTypeIDPrefix[] =
    "go.method.type.reflect.";
static constexpr char RuntimeStringCatSuffix[] =
    "runtime/internal/runtime.StringCat";
static constexpr char RuntimeStringSlice2Suffix[] =
    "runtime/internal/runtime.StringSlice2";
static constexpr unsigned MaxReflectMethodNames = 32;
static constexpr unsigned MaxStringAnalysisDepth = 12;
static constexpr unsigned MaxConstantGEPChoices = 32;

// Keep the replacement bounded. The pass is only meant to refine cases where
// optimization has exposed a small finite set of names. If the set grows too
// large we leave the original generic marker intact, which preserves the old
// conservative GlobalDCE behavior.
bool addName(SmallVectorImpl<std::string> &Names, StringRef Name) {
  for (const std::string &Existing : Names) {
    if (Existing == Name)
      return true;
  }
  if (Names.size() >= MaxReflectMethodNames)
    return false;
  Names.push_back(Name.str());
  return true;
}

// LLGo strings are lowered as (ptr, len). After LTO optimizations the pointer
// usually still points into a private global string literal, possibly with a
// constant GEP offset. Reading through the initializer lets the pass recover
// names that are no longer visible to the Go SSA builder, for example after
// inlining and scalar replacement.
std::optional<std::string> readConstantString(Value *Ptr, Value *Len,
                                              const DataLayout &DL) {
  auto *LenC = dyn_cast<ConstantInt>(Len);
  if (!LenC || LenC->isNegative())
    return std::nullopt;
  uint64_t Size = LenC->getZExtValue();

  int64_t Offset = 0;
  Value *Base = GetPointerBaseWithConstantOffset(Ptr, Offset, DL);
  if (!Base || Offset < 0)
    return std::nullopt;

  auto *GV = dyn_cast<GlobalVariable>(Base->stripPointerCasts());
  if (!GV || !GV->hasInitializer())
    return std::nullopt;

  auto *Data = dyn_cast<ConstantDataArray>(GV->getInitializer());
  if (!Data || !Data->isString())
    return std::nullopt;

  StringRef Bytes = Data->getAsString();
  uint64_t Start = static_cast<uint64_t>(Offset);
  if (Start > Bytes.size() || Size > Bytes.size() - Start)
    return std::nullopt;
  return Bytes.substr(Start, Size).str();
}

bool collectStringSet(Value *Ptr, Value *Len, const DataLayout &DL,
                      SmallVectorImpl<std::string> &Names, unsigned Depth = 0);

bool collectStringSetFromStringValue(Value *StringValue, const DataLayout &DL,
                                     SmallVectorImpl<std::string> &Names,
                                     unsigned Depth = 0);

bool collectIntegerChoices(Value *V, SmallVectorImpl<uint64_t> &Choices,
                           unsigned Depth = 0);

bool collectStringSetFromFunctionStringArg(Value *Ptr, Value *Len,
                                           const DataLayout &DL,
                                           SmallVectorImpl<std::string> &Names,
                                           unsigned Depth);

std::optional<unsigned> singleExtractValueIndex(ExtractValueInst *Extract) {
  if (!Extract || Extract->getNumIndices() != 1)
    return std::nullopt;
  return Extract->getIndices()[0];
}

std::optional<unsigned> singleInsertValueIndex(InsertValueInst *Insert) {
  if (!Insert || Insert->getNumIndices() != 1)
    return std::nullopt;
  return Insert->getIndices()[0];
}

Value *findInsertedValue(Value *Aggregate, unsigned Index) {
  for (Value *Cur = Aggregate;;) {
    auto *Insert = dyn_cast<InsertValueInst>(Cur);
    if (!Insert)
      return nullptr;
    auto InsertIndex = singleInsertValueIndex(Insert);
    if (InsertIndex && *InsertIndex == Index)
      return Insert->getInsertedValueOperand();
    Cur = Insert->getAggregateOperand();
  }
}

bool collectStringSetFromStringConstant(Constant *C, const DataLayout &DL,
                                        SmallVectorImpl<std::string> &Names,
                                        unsigned Depth) {
  if (isa<ConstantAggregateZero>(C))
    return addName(Names, "");

  Constant *Ptr = C->getAggregateElement(0U);
  Constant *Len = C->getAggregateElement(1U);
  if (!Ptr || !Len)
    return false;
  return collectStringSet(Ptr, Len, DL, Names, Depth + 1);
}

bool addConcatenatedNames(SmallVectorImpl<std::string> &Names,
                          ArrayRef<std::string> Prefixes,
                          ArrayRef<std::string> Suffixes) {
  for (const std::string &Prefix : Prefixes) {
    for (const std::string &Suffix : Suffixes) {
      if (!addName(Names, Prefix + Suffix))
        return false;
    }
  }
  return true;
}

bool isRuntimeStringCat(CallBase *CB) {
  if (!CB)
    return false;
  Function *Callee = CB->getCalledFunction();
  return Callee && Callee->getName().ends_with(RuntimeStringCatSuffix);
}

bool isRuntimeStringSlice2(CallBase *CB) {
  if (!CB)
    return false;
  Function *Callee = CB->getCalledFunction();
  return Callee && Callee->getName().ends_with(RuntimeStringSlice2Suffix);
}

bool consumeStringCallArgument(CallBase *CB, unsigned &NextArgIndex,
                               const DataLayout &DL,
                               SmallVectorImpl<std::string> &Names,
                               unsigned Depth) {
  if (NextArgIndex >= CB->arg_size())
    return false;

  Value *Arg = CB->getArgOperand(NextArgIndex);
  if (Arg->getType()->isStructTy()) {
    ++NextArgIndex;
    return collectStringSetFromStringValue(Arg, DL, Names, Depth + 1);
  }

  if (NextArgIndex + 1 >= CB->arg_size())
    return false;
  Value *Len = CB->getArgOperand(NextArgIndex + 1);
  if (!Arg->getType()->isPointerTy() || !Len->getType()->isIntegerTy())
    return false;
  NextArgIndex += 2;
  return collectStringSet(Arg, Len, DL, Names, Depth + 1);
}

bool collectStringSetFromStringCat(CallBase *CB, const DataLayout &DL,
                                   SmallVectorImpl<std::string> &Names,
                                   unsigned Depth) {
  if (!isRuntimeStringCat(CB))
    return false;

  SmallVector<std::string, 4> Prefixes;
  SmallVector<std::string, 4> Suffixes;
  unsigned NextArgIndex = 0;
  if (!consumeStringCallArgument(CB, NextArgIndex, DL, Prefixes, Depth) ||
      !consumeStringCallArgument(CB, NextArgIndex, DL, Suffixes, Depth) ||
      NextArgIndex != CB->arg_size())
    return false;
  return addConcatenatedNames(Names, Prefixes, Suffixes);
}

bool collectStringSetFromStringSlice2(CallBase *CB, const DataLayout &DL,
                                      SmallVectorImpl<std::string> &Names,
                                      unsigned Depth) {
  if (!isRuntimeStringSlice2(CB))
    return false;

  SmallVector<std::string, 4> Bases;
  unsigned NextArgIndex = 0;
  if (!consumeStringCallArgument(CB, NextArgIndex, DL, Bases, Depth) ||
      CB->arg_size() != NextArgIndex + 4)
    return false;

  SmallVector<uint64_t, 4> Lows;
  SmallVector<uint64_t, 4> Highs;
  if (!collectIntegerChoices(CB->getArgOperand(NextArgIndex), Lows,
                             Depth + 1) ||
      !collectIntegerChoices(CB->getArgOperand(NextArgIndex + 1), Highs,
                             Depth + 1))
    return false;

  for (const std::string &Base : Bases) {
    for (uint64_t I : Lows) {
      for (uint64_t J : Highs) {
        if (I > J || J > Base.size())
          return false;
        if (!addName(Names, StringRef(Base).substr(I, J - I)))
          return false;
      }
    }
  }
  return true;
}

bool hasOnlyReadUses(User *U, SmallPtrSetImpl<User *> &Seen,
                     bool FollowGlobalPointerLoads = false) {
  if (!Seen.insert(U).second)
    return true;

  if (auto *Load = dyn_cast<LoadInst>(U)) {
    if (!Load->isSimple())
      return false;
    if (!FollowGlobalPointerLoads || !Load->getType()->isPointerTy() ||
        !isa<GlobalVariable>(
            Load->getPointerOperand()->stripPointerCasts()))
      return true;
    for (User *Next : Load->users()) {
      if (!hasOnlyReadUses(Next, Seen, false))
        return false;
    }
    return true;
  }

  // A static slice backing array is referenced by the constant slice-header
  // initializer, which in turn initializes the package global. Follow that
  // chain and prove that pointers loaded from the slice header are only used
  // for reads before consulting the backing initializer.
  if (isa<ConstantAggregate>(U) || isa<GlobalVariable>(U)) {
    for (User *Next : U->users()) {
      if (!hasOnlyReadUses(Next, Seen, true))
        return false;
    }
    return true;
  }

  if (isa<GEPOperator>(U) || isa<BitCastOperator>(U) ||
      isa<AddrSpaceCastOperator>(U) || isa<BitCastInst>(U) ||
      isa<AddrSpaceCastInst>(U)) {
    for (User *Next : U->users()) {
      if (!hasOnlyReadUses(Next, Seen, FollowGlobalPointerLoads))
        return false;
    }
    return true;
  }

  return false;
}

bool canReadGlobalInitializer(GlobalVariable *GV) {
  if (!GV || !GV->hasInitializer() || GV->isExternallyInitialized())
    return false;
  if (GV->isConstant())
    return true;

  // LLGo can emit pure package literals as mutable globals with initializers.
  // After full LTO, if every remaining use is a simple load through
  // value-preserving pointer operations, the initializer is the only value this
  // pass can observe.
  SmallPtrSet<User *, 16> Seen;
  for (User *U : GV->users()) {
    if (!hasOnlyReadUses(U, Seen))
      return false;
  }
  return true;
}

Constant *constantAtByteOffset(Constant *Init, uint64_t Offset, Type *LoadTy,
                               const DataLayout &DL) {
  if (!Init)
    return nullptr;
  if (Offset == 0 && Init->getType() == LoadTy)
    return Init;

  if (isa<ConstantAggregateZero>(Init)) {
    uint64_t LoadSize = DL.getTypeStoreSize(LoadTy);
    uint64_t InitSize = DL.getTypeAllocSize(Init->getType());
    if (Offset <= InitSize && LoadSize <= InitSize - Offset)
      return Constant::getNullValue(LoadTy);
    return nullptr;
  }

  Type *Ty = Init->getType();
  if (auto *StructTy = dyn_cast<StructType>(Ty)) {
    if (!StructTy->isSized())
      return nullptr;
    const StructLayout *Layout = DL.getStructLayout(StructTy);
    if (Offset >= Layout->getSizeInBytes())
      return nullptr;
    unsigned Index = Layout->getElementContainingOffset(Offset);
    uint64_t ElemOffset = Layout->getElementOffset(Index);
    Constant *Elem = Init->getAggregateElement(Index);
    if (!Elem)
      return nullptr;
    return constantAtByteOffset(Elem, Offset - ElemOffset, LoadTy, DL);
  }

  if (auto *ArrayTy = dyn_cast<ArrayType>(Ty)) {
    Type *ElemTy = ArrayTy->getElementType();
    uint64_t ElemSize = DL.getTypeAllocSize(ElemTy);
    if (ElemSize == 0)
      return nullptr;
    uint64_t Index = Offset / ElemSize;
    if (Index >= ArrayTy->getNumElements())
      return nullptr;
    Constant *Elem = Init->getAggregateElement(static_cast<unsigned>(Index));
    if (!Elem)
      return nullptr;
    return constantAtByteOffset(Elem, Offset - Index * ElemSize, LoadTy, DL);
  }

  return nullptr;
}

Constant *foldLoadFromConstantPtr(Constant *Ptr, Type *LoadTy,
                                  const DataLayout &DL) {
  if (Constant *Folded = ConstantFoldLoadFromConstPtr(Ptr, LoadTy, DL))
    return Folded;

  int64_t Offset = 0;
  Value *Base = GetPointerBaseWithConstantOffset(Ptr, Offset, DL);
  if (!Base || Offset < 0)
    return nullptr;
  auto *GV = dyn_cast<GlobalVariable>(Base->stripPointerCasts());
  if (!canReadGlobalInitializer(GV))
    return nullptr;
  return constantAtByteOffset(GV->getInitializer(),
                              static_cast<uint64_t>(Offset), LoadTy, DL);
}

Constant *foldConstantLoad(LoadInst *Load, const DataLayout &DL) {
  if (!Load || !Load->isSimple())
    return nullptr;
  auto *Ptr =
      dyn_cast<Constant>(Load->getPointerOperand()->stripPointerCasts());
  if (!Ptr)
    return nullptr;
  return foldLoadFromConstantPtr(Ptr, Load->getType(), DL);
}

bool addIntegerChoice(SmallVectorImpl<uint64_t> &Choices, uint64_t Value) {
  for (uint64_t Existing : Choices) {
    if (Existing == Value)
      return true;
  }
  if (Choices.size() >= MaxConstantGEPChoices)
    return false;
  Choices.push_back(Value);
  return true;
}

bool collectIntegerChoices(Value *V, SmallVectorImpl<uint64_t> &Choices,
                           unsigned Depth) {
  if (Depth > MaxStringAnalysisDepth)
    return false;

  if (auto *C = dyn_cast<ConstantInt>(V)) {
    if (C->isNegative())
      return false;
    return addIntegerChoice(Choices, C->getZExtValue());
  }

  if (auto *ZExt = dyn_cast<ZExtInst>(V)) {
    if (ZExt->getSrcTy()->isIntegerTy(1))
      return addIntegerChoice(Choices, 0) && addIntegerChoice(Choices, 1);
  }

  if (auto *Sel = dyn_cast<SelectInst>(V)) {
    return collectIntegerChoices(Sel->getTrueValue(), Choices, Depth + 1) &&
           collectIntegerChoices(Sel->getFalseValue(), Choices, Depth + 1);
  }

  if (auto *Phi = dyn_cast<PHINode>(V)) {
    for (Value *Incoming : Phi->incoming_values()) {
      if (!collectIntegerChoices(Incoming, Choices, Depth + 1))
        return false;
    }
    return true;
  }

  return false;
}

bool isSelfIncrementingPhi(Value *V, unsigned Depth = 0);

std::optional<int64_t> signedConstantInt(Value *V) {
  auto *C = dyn_cast<ConstantInt>(V);
  if (!C || C->getBitWidth() > 64)
    return std::nullopt;
  return C->getSExtValue();
}

bool isAddOfPhi(Value *V, PHINode *Phi, int64_t &Step, unsigned Depth) {
  if (Depth > MaxStringAnalysisDepth)
    return false;

  if (auto *Cast = dyn_cast<CastInst>(V)) {
    if (Cast->getType()->isIntegerTy() &&
        Cast->getOperand(0)->getType()->isIntegerTy())
      return isAddOfPhi(Cast->getOperand(0), Phi, Step, Depth + 1);
  }

  auto *Bin = dyn_cast<BinaryOperator>(V);
  if (!Bin || Bin->getOpcode() != Instruction::Add)
    return false;

  Value *LHS = Bin->getOperand(0);
  Value *RHS = Bin->getOperand(1);
  if (LHS == Phi) {
    if (auto C = signedConstantInt(RHS)) {
      Step = *C;
      return true;
    }
  }
  if (RHS == Phi) {
    if (auto C = signedConstantInt(LHS)) {
      Step = *C;
      return true;
    }
  }
  return false;
}

bool isIncrementOfPhi(Value *V, PHINode *Phi, unsigned Depth) {
  int64_t Step = 0;
  return isAddOfPhi(V, Phi, Step, Depth);
}

bool isSelfIncrementingPhi(Value *V, unsigned Depth) {
  if (Depth > MaxStringAnalysisDepth)
    return false;

  if (auto *Cast = dyn_cast<CastInst>(V)) {
    if (Cast->getType()->isIntegerTy() &&
        Cast->getOperand(0)->getType()->isIntegerTy())
      return isSelfIncrementingPhi(Cast->getOperand(0), Depth + 1);
  }

  auto *Phi = dyn_cast<PHINode>(V);
  if (!Phi)
    return false;

  bool HasSeed = false;
  bool HasStep = false;
  for (Value *Incoming : Phi->incoming_values()) {
    if (auto *C = dyn_cast<ConstantInt>(Incoming)) {
      if (C->isNegative())
        return false;
      HasSeed = true;
      continue;
    }
    if (isIncrementOfPhi(Incoming, Phi, Depth + 1)) {
      HasStep = true;
      continue;
    }
  }
  return HasSeed && HasStep;
}

bool isNormalizedInductionIndex(Value *V, unsigned Depth = 0) {
  if (isSelfIncrementingPhi(V, Depth))
    return true;
  if (Depth > MaxStringAnalysisDepth)
    return false;

  auto *Bin = dyn_cast<BinaryOperator>(V);
  if (!Bin || Bin->getOpcode() != Instruction::Add)
    return false;

  PHINode *Phi = nullptr;
  int64_t Adjustment = 0;
  if ((Phi = dyn_cast<PHINode>(Bin->getOperand(0)))) {
    auto C = signedConstantInt(Bin->getOperand(1));
    if (!C)
      return false;
    Adjustment = *C;
  } else if ((Phi = dyn_cast<PHINode>(Bin->getOperand(1)))) {
    auto C = signedConstantInt(Bin->getOperand(0));
    if (!C)
      return false;
    Adjustment = *C;
  } else {
    return false;
  }

  bool HasSeed = false;
  bool HasStep = false;
  for (Value *Incoming : Phi->incoming_values()) {
    if (auto Seed = signedConstantInt(Incoming)) {
      if (*Seed + Adjustment != 0)
        return false;
      HasSeed = true;
      continue;
    }

    int64_t Step = 0;
    if (isAddOfPhi(Incoming, Phi, Step, Depth + 1) && Step == 1) {
      HasStep = true;
      continue;
    }
    return false;
  }
  return HasSeed && HasStep;
}

bool collectBoundedInductionChoices(Value *Index, uint64_t Bound,
                                    SmallVectorImpl<uint64_t> &Choices) {
  if (Bound > MaxConstantGEPChoices || !isNormalizedInductionIndex(Index))
    return false;
  for (uint64_t I = 0; I != Bound; ++I) {
    if (!addIntegerChoice(Choices, I))
      return false;
  }
  return true;
}

bool collectIndexConstants(Value *Index, SmallVectorImpl<Constant *> &Constants,
                           std::optional<uint64_t> Bound = std::nullopt) {
  if (auto *C = dyn_cast<ConstantInt>(Index)) {
    if (Bound && C->getZExtValue() >= *Bound)
      return false;
    Constants.push_back(C);
    return true;
  }

  SmallVector<uint64_t, 4> Choices;
  if (!collectIntegerChoices(Index, Choices)) {
    if (!Bound || !collectBoundedInductionChoices(Index, *Bound, Choices))
      return false;
  }

  Type *IndexTy = Index->getType();
  for (uint64_t Choice : Choices) {
    if (Bound && Choice >= *Bound)
      return false;
    Constants.push_back(ConstantInt::get(IndexTy, Choice));
  }
  return true;
}

std::optional<uint64_t>
firstIndexArrayBound(ArrayRef<Constant *> Bases, Type *ElemTy,
                     const DataLayout &DL) {
  std::optional<uint64_t> Bound;
  for (Constant *Base : Bases) {
    int64_t Offset = 0;
    Value *Root = GetPointerBaseWithConstantOffset(Base, Offset, DL);
    auto *GV = Root ? dyn_cast<GlobalVariable>(Root->stripPointerCasts())
                    : nullptr;
    auto *ArrayTy = GV ? dyn_cast<ArrayType>(GV->getValueType()) : nullptr;
    if (!ArrayTy || ArrayTy->getElementType() != ElemTy || Offset != 0)
      return std::nullopt;
    uint64_t Current = ArrayTy->getNumElements();
    if (Bound && *Bound != Current)
      return std::nullopt;
    Bound = Current;
  }
  return Bound;
}

bool enumerateConstantPointers(Value *Ptr, const DataLayout &DL,
                               SmallVectorImpl<Constant *> &Pointers) {
  Ptr = Ptr->stripPointerCasts();
  if (auto *C = dyn_cast<Constant>(Ptr)) {
    Pointers.push_back(C);
    return true;
  }

  if (auto *Load = dyn_cast<LoadInst>(Ptr)) {
    Constant *Folded = foldConstantLoad(Load, DL);
    return Folded && enumerateConstantPointers(Folded, DL, Pointers);
  }

  auto *GEP = dyn_cast<GEPOperator>(Ptr);
  if (!GEP)
    return false;

  SmallVector<Constant *, 4> Bases;
  if (!enumerateConstantPointers(GEP->getPointerOperand(), DL, Bases))
    return false;

  SmallVector<SmallVector<Constant *, 4>, 4> IndexChoices;
  Type *IndexedTy = GEP->getSourceElementType();
  bool FirstIndex = true;
  for (Value *Index : GEP->indices()) {
    SmallVector<Constant *, 4> Choices;
    std::optional<uint64_t> Bound;
    if (FirstIndex) {
      Bound = firstIndexArrayBound(Bases, IndexedTy, DL);
    } else {
      if (auto *ArrayTy = dyn_cast<ArrayType>(IndexedTy))
        Bound = ArrayTy->getNumElements();
    }
    if (!collectIndexConstants(Index, Choices, Bound))
      return false;
    IndexChoices.push_back(std::move(Choices));

    if (FirstIndex) {
      FirstIndex = false;
      continue;
    }
    if (auto *StructTy = dyn_cast<StructType>(IndexedTy)) {
      auto *C = dyn_cast<ConstantInt>(Index);
      if (!C || C->getZExtValue() >= StructTy->getNumElements())
        return false;
      IndexedTy =
          StructTy->getElementType(static_cast<unsigned>(C->getZExtValue()));
    } else if (auto *ArrayTy = dyn_cast<ArrayType>(IndexedTy)) {
      IndexedTy = ArrayTy->getElementType();
    }
  }

  SmallVector<SmallVector<Constant *, 4>, 8> Work;
  Work.push_back({});
  for (ArrayRef<Constant *> Choices : IndexChoices) {
    SmallVector<SmallVector<Constant *, 4>, 8> Next;
    for (const auto &Prefix : Work) {
      for (Constant *Choice : Choices) {
        if (Next.size() >= MaxConstantGEPChoices)
          return false;
        SmallVector<Constant *, 4> Combined(Prefix.begin(), Prefix.end());
        Combined.push_back(Choice);
        Next.push_back(std::move(Combined));
      }
    }
    Work = std::move(Next);
  }

  for (Constant *Base : Bases) {
    for (const auto &Indices : Work) {
      if (Pointers.size() >= MaxConstantGEPChoices)
        return false;
      Pointers.push_back(ConstantExpr::getGetElementPtr(
          GEP->getSourceElementType(), Base, Indices, GEP->isInBounds()));
    }
  }
  return true;
}

struct LocalStringSlot {
  AllocaInst *Base;
  uint64_t Offset;
};

bool enumerateLocalStringSlots(Value *Ptr, const DataLayout &DL,
                               SmallVectorImpl<LocalStringSlot> &Slots) {
  Ptr = Ptr->stripPointerCasts();
  auto *GEP = dyn_cast<GEPOperator>(Ptr);
  if (!GEP || GEP->getNumIndices() != 1)
    return false;

  auto *Base =
      dyn_cast<AllocaInst>(GEP->getPointerOperand()->stripPointerCasts());
  if (!Base || Base->isArrayAllocation())
    return false;

  auto *ArrayTy = dyn_cast<ArrayType>(Base->getAllocatedType());
  if (!ArrayTy || ArrayTy->getElementType() != GEP->getSourceElementType())
    return false;

  SmallVector<Constant *, 4> Indices;
  if (!collectIndexConstants(*GEP->idx_begin(), Indices,
                             ArrayTy->getNumElements()))
    return false;

  uint64_t ElemSize = DL.getTypeAllocSize(ArrayTy->getElementType());
  for (Constant *Index : Indices) {
    auto *IndexC = dyn_cast<ConstantInt>(Index);
    if (!IndexC || IndexC->isNegative())
      return false;
    Slots.push_back({Base, IndexC->getZExtValue() * ElemSize});
  }
  return true;
}

bool hasOnlyConstantLocalSlotUses(Value *Ptr, AllocaInst *Base,
                                  const DataLayout &DL,
                                  SmallPtrSetImpl<User *> &Seen) {
  for (User *U : Ptr->users()) {
    if (!Seen.insert(U).second)
      continue;

    if (auto *Load = dyn_cast<LoadInst>(U)) {
      if (!Load->isSimple())
        return false;
      continue;
    }

    if (auto *Store = dyn_cast<StoreInst>(U)) {
      if (!Store->isSimple() || !isa<Constant>(Store->getValueOperand()))
        return false;
      int64_t Offset = 0;
      Value *FoundBase = GetPointerBaseWithConstantOffset(
          Store->getPointerOperand(), Offset, DL);
      if (FoundBase != Base || Offset < 0)
        return false;
      continue;
    }

    if (isa<GEPOperator>(U) || isa<BitCastOperator>(U) ||
        isa<AddrSpaceCastOperator>(U) || isa<BitCastInst>(U) ||
        isa<AddrSpaceCastInst>(U)) {
      if (!hasOnlyConstantLocalSlotUses(U, Base, DL, Seen))
        return false;
      continue;
    }

    auto *II = dyn_cast<IntrinsicInst>(U);
    if (II && (II->getIntrinsicID() == Intrinsic::lifetime_start ||
               II->getIntrinsicID() == Intrinsic::lifetime_end))
      continue;
    return false;
  }
  return true;
}

Constant *findDominatingLocalConstant(AllocaInst *Base, uint64_t WantOffset,
                                      LoadInst *Load, const DataLayout &DL,
                                      DominatorTree &DT) {
  uint64_t LoadSize = DL.getTypeStoreSize(Load->getType());
  Constant *Found = nullptr;
  for (BasicBlock &BB : *Base->getFunction()) {
    for (Instruction &I : BB) {
      auto *Store = dyn_cast<StoreInst>(&I);
      if (!Store)
        continue;

      int64_t Offset = 0;
      Value *FoundBase = GetPointerBaseWithConstantOffset(
          Store->getPointerOperand(), Offset, DL);
      if (FoundBase != Base || Offset < 0)
        continue;

      uint64_t StoreOffset = static_cast<uint64_t>(Offset);
      uint64_t StoreSize =
          DL.getTypeStoreSize(Store->getValueOperand()->getType());
      if (StoreOffset + StoreSize <= WantOffset ||
          WantOffset + LoadSize <= StoreOffset)
        continue;

      auto *C = dyn_cast<Constant>(Store->getValueOperand());
      if (!Store->isSimple() || StoreOffset != WantOffset ||
          Store->getValueOperand()->getType() != Load->getType() || !C ||
          !DT.dominates(Store, Load) || (Found && Found != C))
        return nullptr;
      Found = C;
    }
  }
  return Found;
}

bool isByteOffsetFrom(Value *Ptr, Value *Base, int64_t WantOffset,
                      const DataLayout &DL) {
  int64_t Offset = 0;
  Value *FoundBase = GetPointerBaseWithConstantOffset(Ptr, Offset, DL);
  return FoundBase == Base->stripPointerCasts() && Offset == WantOffset;
}

Constant *byteOffsetPointer(Constant *Base, uint64_t Offset, LLVMContext &Ctx) {
  Type *Int8Ty = Type::getInt8Ty(Ctx);
  Type *Int64Ty = Type::getInt64Ty(Ctx);
  Constant *Index = ConstantInt::get(Int64Ty, Offset);
  return ConstantExpr::getGetElementPtr(Int8Ty, Base,
                                        ArrayRef<Constant *>{Index});
}

bool collectStringSetFromConstantSlots(LoadInst *PtrLoad, LoadInst *LenLoad,
                                       const DataLayout &DL,
                                       SmallVectorImpl<std::string> &Names,
                                       unsigned Depth) {
  if (!PtrLoad || !LenLoad || !PtrLoad->isSimple() || !LenLoad->isSimple())
    return false;

  Value *StringSlot = PtrLoad->getPointerOperand()->stripPointerCasts();
  if (!isByteOffsetFrom(LenLoad->getPointerOperand(), StringSlot, 8, DL))
    return false;

  SmallVector<Constant *, 4> Slots;
  if (!enumerateConstantPointers(StringSlot, DL, Slots))
    return false;

  LLVMContext &Ctx = PtrLoad->getContext();
  for (Constant *Slot : Slots) {
    Constant *Ptr = foldLoadFromConstantPtr(Slot, PtrLoad->getType(), DL);
    Constant *LenPtr = byteOffsetPointer(Slot, 8, Ctx);
    Constant *Len = foldLoadFromConstantPtr(LenPtr, LenLoad->getType(), DL);
    if (!Ptr || !Len || !collectStringSet(Ptr, Len, DL, Names, Depth + 1))
      return false;
  }
  return true;
}

bool collectStringSetFromLocalSlots(LoadInst *PtrLoad, LoadInst *LenLoad,
                                    const DataLayout &DL,
                                    SmallVectorImpl<std::string> &Names,
                                    unsigned Depth) {
  if (!PtrLoad || !LenLoad || !PtrLoad->isSimple() || !LenLoad->isSimple())
    return false;

  Value *StringSlot = PtrLoad->getPointerOperand()->stripPointerCasts();
  if (!isByteOffsetFrom(LenLoad->getPointerOperand(), StringSlot, 8, DL))
    return false;

  SmallVector<LocalStringSlot, 4> Slots;
  if (!enumerateLocalStringSlots(StringSlot, DL, Slots) || Slots.empty())
    return false;

  AllocaInst *Base = Slots.front().Base;
  SmallPtrSet<User *, 16> Seen;
  if (!hasOnlyConstantLocalSlotUses(Base, Base, DL, Seen))
    return false;

  DominatorTree DT(*Base->getFunction());
  for (const LocalStringSlot &Slot : Slots) {
    if (Slot.Base != Base)
      return false;
    Constant *Ptr =
        findDominatingLocalConstant(Base, Slot.Offset, PtrLoad, DL, DT);
    Constant *Len =
        findDominatingLocalConstant(Base, Slot.Offset + 8, LenLoad, DL, DT);
    if (!Ptr || !Len || !collectStringSet(Ptr, Len, DL, Names, Depth + 1))
      return false;
  }
  return true;
}

bool collectStringSetFromFunctionStringArg(Value *Ptr, Value *Len,
                                           const DataLayout &DL,
                                           SmallVectorImpl<std::string> &Names,
                                           unsigned Depth) {
  auto *PtrArg = dyn_cast<Argument>(Ptr);
  auto *LenArg = dyn_cast<Argument>(Len);
  if (!PtrArg || !LenArg)
    return false;

  Function *F = PtrArg->getParent();
  if (!F || F != LenArg->getParent())
    return false;

  unsigned PtrArgNo = PtrArg->getArgNo();
  unsigned LenArgNo = LenArg->getArgNo();
  if (LenArgNo != PtrArgNo + 1 || !PtrArg->getType()->isPointerTy() ||
      !LenArg->getType()->isIntegerTy())
    return false;

  SmallVector<CallBase *, 8> Callers;
  for (User *U : F->users()) {
    auto *CB = dyn_cast<CallBase>(U);
    if (!CB || CB->getCalledFunction() != F)
      return false;
    Callers.push_back(CB);
  }
  if (Callers.empty())
    return false;

  for (CallBase *CB : Callers) {
    if (LenArgNo >= CB->arg_size())
      return false;
    Value *CallPtr = CB->getArgOperand(PtrArgNo);
    Value *CallLen = CB->getArgOperand(LenArgNo);
    if (!CallPtr->getType()->isPointerTy() || !CallLen->getType()->isIntegerTy())
      return false;
    if (!collectStringSet(CallPtr, CallLen, DL, Names, Depth + 1))
      return false;
  }
  return true;
}

bool collectStringSetFromFunctionStringValueArg(
    Value *StringValue, const DataLayout &DL,
    SmallVectorImpl<std::string> &Names, unsigned Depth) {
  auto *Arg = dyn_cast<Argument>(StringValue);
  if (!Arg || !Arg->getType()->isStructTy())
    return false;

  Function *F = Arg->getParent();
  if (!F)
    return false;

  unsigned ArgNo = Arg->getArgNo();
  SmallVector<CallBase *, 8> Callers;
  for (User *U : F->users()) {
    auto *CB = dyn_cast<CallBase>(U);
    if (!CB || CB->getCalledFunction() != F || ArgNo >= CB->arg_size())
      return false;
    Callers.push_back(CB);
  }
  if (Callers.empty())
    return false;

  for (CallBase *CB : Callers) {
    Value *CallValue = CB->getArgOperand(ArgNo);
    if (!CallValue->getType()->isStructTy() ||
        !collectStringSetFromStringValue(CallValue, DL, Names, Depth + 1))
      return false;
  }
  return true;
}

bool collectStringSetFromStringValue(Value *StringValue, const DataLayout &DL,
                                     SmallVectorImpl<std::string> &Names,
                                     unsigned Depth) {
  if (Depth > MaxStringAnalysisDepth)
    return false;

  if (auto *C = dyn_cast<Constant>(StringValue))
    return collectStringSetFromStringConstant(C, DL, Names, Depth);

  if (auto *CB = dyn_cast<CallBase>(StringValue)) {
    if (collectStringSetFromStringSlice2(CB, DL, Names, Depth))
      return true;
    if (collectStringSetFromStringCat(CB, DL, Names, Depth))
      return true;
  }

  if (auto *Load = dyn_cast<LoadInst>(StringValue)) {
    if (Constant *Folded = foldConstantLoad(Load, DL))
      return collectStringSetFromStringConstant(Folded, DL, Names, Depth + 1);
  }

  if (collectStringSetFromFunctionStringValueArg(StringValue, DL, Names, Depth))
    return true;

  if (auto *Insert = dyn_cast<InsertValueInst>(StringValue)) {
    Value *Ptr = findInsertedValue(Insert, 0);
    Value *Len = findInsertedValue(Insert, 1);
    if (!Ptr || !Len)
      return false;
    return collectStringSet(Ptr, Len, DL, Names, Depth + 1);
  }

  if (auto *Sel = dyn_cast<SelectInst>(StringValue)) {
    return collectStringSetFromStringValue(Sel->getTrueValue(), DL, Names,
                                           Depth + 1) &&
           collectStringSetFromStringValue(Sel->getFalseValue(), DL, Names,
                                           Depth + 1);
  }

  if (auto *Phi = dyn_cast<PHINode>(StringValue)) {
    for (Value *Incoming : Phi->incoming_values()) {
      if (!collectStringSetFromStringValue(Incoming, DL, Names, Depth + 1))
        return false;
    }
    return true;
  }

  return false;
}

bool collectExtractedStringSet(Value *Ptr, Value *Len, const DataLayout &DL,
                               SmallVectorImpl<std::string> &Names,
                               unsigned Depth) {
  auto *PtrExtract = dyn_cast<ExtractValueInst>(Ptr);
  auto *LenExtract = dyn_cast<ExtractValueInst>(Len);
  auto PtrIndex = singleExtractValueIndex(PtrExtract);
  auto LenIndex = singleExtractValueIndex(LenExtract);
  if (!PtrIndex || !LenIndex || *PtrIndex != 0 || *LenIndex != 1 ||
      PtrExtract->getAggregateOperand() != LenExtract->getAggregateOperand())
    return false;
  return collectStringSetFromStringValue(PtrExtract->getAggregateOperand(), DL,
                                         Names, Depth + 1);
}

// Collect every possible string from a lowered (ptr, len) pair.
//
// The pass is intentionally all-or-nothing: it returns false as soon as any
// incoming value is unknown. That way a partially understood dynamic
// MethodByName call cannot accidentally lose the generic reflect marker and
// make GlobalDCE drop a method that may still be reachable at runtime.
bool collectStringSet(Value *Ptr, Value *Len, const DataLayout &DL,
                      SmallVectorImpl<std::string> &Names, unsigned Depth) {
  if (Depth > MaxStringAnalysisDepth)
    return false;

  Ptr = Ptr->stripPointerCasts();

  bool FoldedLoad = false;
  if (auto *PtrLoad = dyn_cast<LoadInst>(Ptr)) {
    if (Constant *Folded = foldConstantLoad(PtrLoad, DL)) {
      Ptr = Folded;
      FoldedLoad = true;
    }
  }
  if (auto *LenLoad = dyn_cast<LoadInst>(Len)) {
    if (Constant *Folded = foldConstantLoad(LenLoad, DL)) {
      Len = Folded;
      FoldedLoad = true;
    }
  }
  if (FoldedLoad)
    return collectStringSet(Ptr, Len, DL, Names, Depth + 1);

  if (collectStringSetFromFunctionStringArg(Ptr, Len, DL, Names, Depth))
    return true;

  if (collectStringSetFromConstantSlots(
          dyn_cast<LoadInst>(Ptr), dyn_cast<LoadInst>(Len), DL, Names, Depth))
    return true;

  if (collectStringSetFromLocalSlots(dyn_cast<LoadInst>(Ptr),
                                     dyn_cast<LoadInst>(Len), DL, Names, Depth))
    return true;

  if (auto *LenC = dyn_cast<ConstantInt>(Len); LenC && LenC->isZero())
    return addName(Names, "");

  if (auto Name = readConstantString(Ptr, Len, DL))
    return addName(Names, *Name);

  if (collectExtractedStringSet(Ptr, Len, DL, Names, Depth))
    return true;

  auto *PtrSel = dyn_cast<SelectInst>(Ptr);
  auto *LenSel = dyn_cast<SelectInst>(Len);
  if (PtrSel && LenSel && PtrSel->getCondition() == LenSel->getCondition()) {
    return collectStringSet(PtrSel->getTrueValue(), LenSel->getTrueValue(), DL,
                            Names, Depth + 1) &&
           collectStringSet(PtrSel->getFalseValue(), LenSel->getFalseValue(),
                            DL, Names, Depth + 1);
  }
  if (PtrSel && isa<ConstantInt>(Len)) {
    return collectStringSet(PtrSel->getTrueValue(), Len, DL, Names,
                            Depth + 1) &&
           collectStringSet(PtrSel->getFalseValue(), Len, DL, Names, Depth + 1);
  }

  auto *PtrPhi = dyn_cast<PHINode>(Ptr);
  auto *LenPhi = dyn_cast<PHINode>(Len);
  if (PtrPhi && LenPhi && PtrPhi->getParent() == LenPhi->getParent() &&
      PtrPhi->getNumIncomingValues() == LenPhi->getNumIncomingValues()) {
    for (unsigned I = 0, E = PtrPhi->getNumIncomingValues(); I != E; ++I) {
      BasicBlock *Incoming = PtrPhi->getIncomingBlock(I);
      int LenIndex = LenPhi->getBasicBlockIndex(Incoming);
      if (LenIndex < 0)
        return false;
      if (!collectStringSet(PtrPhi->getIncomingValue(I),
                            LenPhi->getIncomingValue(LenIndex), DL, Names,
                            Depth + 1))
        return false;
    }
    return true;
  }

  return false;
}

std::optional<unsigned> findReflectMethodNameArg(CallBase *ReflectCall) {
  for (unsigned I = 0, E = ReflectCall->arg_size(); I != E; ++I) {
    Attribute Attr = ReflectCall->getParamAttr(I, ReflectMethodByNameArgAttr);
    if (Attr.isStringAttribute())
      return I;
  }
  return std::nullopt;
}

// Optimized C ABI lowering splits a Go string into (ptr, len), with the name
// attribute on the pointer. Debug-preserving builds can retain the original
// aggregate string argument instead. Accept both representations so enabling
// DWARF does not change MethodByName reachability.
bool collectStringSetFromCallArgs(CallBase *ReflectCall, const DataLayout &DL,
                                  SmallVectorImpl<std::string> &Names) {
  std::optional<unsigned> NameArg = findReflectMethodNameArg(ReflectCall);
  if (!NameArg)
    return false;

  unsigned NextArgIndex = *NameArg;
  return consumeStringCallArgument(ReflectCall, NextArgIndex, DL, Names, 0);
}

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

    bool Changed = false;
    const DataLayout &DL = M.getDataLayout();
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
        if (!collectStringSetFromCallArgs(SharedCall, DL, Names)) {
          KnownNames = false;
          break;
        }
      }
      if (!KnownNames || Names.empty())
        continue;

      // Preserve the analysis result for LLGo's ThinLTO feedback planner.
      // Go method names cannot contain commas, so the compact encoding is
      // unambiguous and remains easy to inspect in saved backend bitcode.
      std::string EncodedNames;
      for (const std::string &Name : Names) {
        if (!EncodedNames.empty())
          EncodedNames.push_back(',');
        EncodedNames.append(Name);
      }
      ReflectCall->addFnAttr(Attribute::get(ReflectCall->getContext(),
                                            ReflectMethodByNameNamesAttr,
                                            EncodedNames));
      Changed = true;

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

      if (std::getenv("LLGO_LTO_PLUGIN_VERBOSE")) {
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
