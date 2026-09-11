#include "LLGOLTOPasses.h"

#include "llvm/ADT/SmallVector.h"
#include "llvm/ADT/StringMap.h"
#include "llvm/ADT/StringSet.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/Instructions.h"
#include "llvm/IR/IntrinsicInst.h"
#include "llvm/IR/Intrinsics.h"
#include "llvm/IR/Metadata.h"
#include "llvm/IR/Module.h"
#include "llvm/IR/PassManager.h"
#include "llvm/Support/ErrorHandling.h"
#include "llvm/Support/raw_ostream.h"

#include <algorithm>
#include <cstdint>
#include <cstdlib>
#include <optional>
#include <string>
#include <utility>
#include <vector>

using namespace llvm;

namespace {

// Keep this call-attribute protocol in sync with the constants and emitter in
// ssa/globaldce.go.
static constexpr char InterfaceCallAttr[] = "llgo.interface.call";
static constexpr char InterfaceTypeIDAttr[] = "llgo.interface.id";
static constexpr char InterfaceMethodIndexAttr[] = "llgo.interface.index";
static constexpr char InterfaceMethodCountAttr[] = "llgo.interface.count";
static constexpr char InterfaceMethodAttrPrefix[] = "llgo.interface.method.";
static constexpr char InterfaceTypeIDPrefix[] = "go.method.i.";
static constexpr char MethodTypeIDPrefix[] = "go.method.";
static constexpr char ReflectValueMethodTypeID[] = "go.method.value.reflect";
static constexpr char ReflectTypeMethodTypeID[] = "go.method.type.reflect";
static constexpr char ReflectValueMethodTypeIDPrefix[] =
    "go.method.value.reflect.";
static constexpr char ReflectTypeMethodTypeIDPrefix[] =
    "go.method.type.reflect.";
static constexpr char InterfaceProtocolVersion[] = "1";

struct InterfaceMethodDecl {
  unsigned Index = 0;
  std::string ExactTypeID;
  std::string BroadTypeID;
};

struct InterfaceDecl {
  // Every checked load carries a protocol marker, interface type ID, called
  // method index, method count, and one broad method type ID per index.
  // The complete ordered method list makes each checked load self-describing.
  std::string TypeID;
  std::vector<InterfaceMethodDecl> Methods;
};

struct DescriptorInfo {
  GlobalVariable *GV = nullptr;
  StringMap<uint64_t> BroadSlots;
  StringMap<uint64_t> ExactSlots;
};

struct TypeCheckSite {
  CallBase *Call = nullptr;
  unsigned TypeIDArg = 0;
};

std::optional<uint64_t> metadataUInt(Metadata *MD) {
  auto *CAM = dyn_cast_or_null<ConstantAsMetadata>(MD);
  auto *CI = CAM ? dyn_cast<ConstantInt>(CAM->getValue()) : nullptr;
  if (!CI || CI->isNegative())
    return std::nullopt;
  return CI->getZExtValue();
}

std::optional<StringRef> callStringAttr(CallBase &CB, StringRef Kind) {
  Attribute Attr = CB.getFnAttr(Kind);
  if (!Attr.isStringAttribute())
    return std::nullopt;
  return Attr.getValueAsString();
}

std::optional<uint32_t> parseUInt32(StringRef Value) {
  uint32_t Result = 0;
  if (Value.empty() || Value.getAsInteger(10, Result))
    return std::nullopt;
  return Result;
}

std::optional<StringRef> metadataString(Metadata *MD) {
  auto *S = dyn_cast_or_null<MDString>(MD);
  if (!S)
    return std::nullopt;
  return S->getString();
}

std::optional<StringRef> checkedLoadTypeID(CallBase *CB) {
  if (!CB || CB->arg_size() < 3)
    return std::nullopt;
  Function *Callee = CB->getCalledFunction();
  if (!Callee || Callee->getIntrinsicID() != Intrinsic::type_checked_load)
    return std::nullopt;
  auto *MDValue = dyn_cast<MetadataAsValue>(CB->getArgOperand(2));
  auto *TypeID = MDValue ? dyn_cast<MDString>(MDValue->getMetadata()) : nullptr;
  return TypeID ? std::optional<StringRef>(TypeID->getString()) : std::nullopt;
}

std::optional<StringRef> typeTestTypeID(CallBase *CB) {
  if (!CB || CB->arg_size() < 2)
    return std::nullopt;
  Function *Callee = CB->getCalledFunction();
  if (!Callee || Callee->getIntrinsicID() != Intrinsic::type_test)
    return std::nullopt;
  auto *MDValue = dyn_cast<MetadataAsValue>(CB->getArgOperand(1));
  auto *TypeID = MDValue ? dyn_cast<MDString>(MDValue->getMetadata()) : nullptr;
  return TypeID ? std::optional<StringRef>(TypeID->getString()) : std::nullopt;
}

bool isReflectMethodTypeID(StringRef TypeID) {
  return TypeID == ReflectValueMethodTypeID ||
         TypeID.starts_with(ReflectValueMethodTypeIDPrefix) ||
         TypeID == ReflectTypeMethodTypeID ||
         TypeID.starts_with(ReflectTypeMethodTypeIDPrefix);
}

bool isInterfaceBaseTypeID(StringRef TypeID) {
  if (!TypeID.starts_with(InterfaceTypeIDPrefix))
    return false;
  StringRef Suffix = TypeID.drop_front(StringRef(InterfaceTypeIDPrefix).size());
  return !Suffix.empty() && !Suffix.contains('.');
}

[[noreturn]] void invalidMetadata(const Twine &Reason) {
  // Reject invalid input without invoking crash reporters or symbolizers.
  reportFatalUsageError(
      Twine("llgo-lto-plugin: invalid interface type-id metadata: ") + Reason);
}

bool sameDeclaration(const InterfaceDecl &A, const InterfaceDecl &B) {
  if (A.TypeID != B.TypeID || A.Methods.size() != B.Methods.size())
    return false;
  for (unsigned I = 0; I < A.Methods.size(); ++I) {
    const InterfaceMethodDecl &AM = A.Methods[I];
    const InterfaceMethodDecl &BM = B.Methods[I];
    if (AM.Index != BM.Index || AM.ExactTypeID != BM.ExactTypeID ||
        AM.BroadTypeID != BM.BroadTypeID)
      return false;
  }
  return true;
}

InterfaceDecl parseInterfaceCall(CallBase &CB, StringRef CheckedTypeID) {
  StringRef FunctionName = CB.getFunction()->getName();
  auto Version = callStringAttr(CB, InterfaceCallAttr);
  auto TypeID = callStringAttr(CB, InterfaceTypeIDAttr);
  auto MethodIndexText = callStringAttr(CB, InterfaceMethodIndexAttr);
  auto MethodCountText = callStringAttr(CB, InterfaceMethodCountAttr);
  auto MethodIndex =
      MethodIndexText ? parseUInt32(*MethodIndexText) : std::nullopt;
  auto MethodCount =
      MethodCountText ? parseUInt32(*MethodCountText) : std::nullopt;
  if (!Version || *Version != InterfaceProtocolVersion || !TypeID ||
      !isInterfaceBaseTypeID(*TypeID) || !MethodIndex || !MethodCount ||
      *MethodCount == 0 || *MethodIndex >= *MethodCount)
    invalidMetadata(Twine("unsupported call attributes in ") + FunctionName);

  InterfaceDecl Decl;
  Decl.TypeID = TypeID->str();
  Decl.Methods.reserve(*MethodCount);
  for (uint32_t I = 0; I < *MethodCount; ++I) {
    std::string MethodAttr =
        InterfaceMethodAttrPrefix + std::to_string(static_cast<uint64_t>(I));
    auto BroadTypeID = callStringAttr(CB, MethodAttr);
    if (!BroadTypeID || !BroadTypeID->starts_with(MethodTypeIDPrefix) ||
        BroadTypeID->starts_with(InterfaceTypeIDPrefix) ||
        isReflectMethodTypeID(*BroadTypeID))
      invalidMetadata(Twine("unsupported method attributes in ") +
                      FunctionName);
    std::string ExactTypeID =
        Decl.TypeID + ".m" + std::to_string(static_cast<uint64_t>(I));
    Decl.Methods.push_back(
        {static_cast<unsigned>(I), std::move(ExactTypeID), BroadTypeID->str()});
  }
  if (CheckedTypeID != Decl.Methods[*MethodIndex].ExactTypeID)
    invalidMetadata(Twine("inconsistent checked type id in ") + FunctionName);
  return Decl;
}

void removeInterfaceCallAttrs(CallBase &CB, unsigned MethodCount) {
  CB.removeFnAttr(InterfaceCallAttr);
  CB.removeFnAttr(InterfaceTypeIDAttr);
  CB.removeFnAttr(InterfaceMethodIndexAttr);
  CB.removeFnAttr(InterfaceMethodCountAttr);
  for (unsigned I = 0; I < MethodCount; ++I) {
    std::string MethodAttr =
        InterfaceMethodAttrPrefix + std::to_string(static_cast<uint64_t>(I));
    CB.removeFnAttr(MethodAttr);
  }
}

bool collectTypeSlots(GlobalVariable &GV, unsigned TypeKind,
                      DescriptorInfo &Info) {
  SmallVector<std::pair<unsigned, MDNode *>, 16> Metadata;
  GV.getAllMetadata(Metadata);
  for (auto [Kind, Node] : Metadata) {
    if (Kind != TypeKind || Node->getNumOperands() < 2)
      continue;
    auto Offset = metadataUInt(Node->getOperand(0));
    auto TypeID = metadataString(Node->getOperand(1));
    if (!Offset || !TypeID)
      continue;
    StringMap<uint64_t> *Slots = nullptr;
    if (TypeID->starts_with(InterfaceTypeIDPrefix))
      Slots = &Info.ExactSlots;
    else if (TypeID->starts_with(MethodTypeIDPrefix) &&
             !isReflectMethodTypeID(*TypeID))
      Slots = &Info.BroadSlots;
    else
      continue;
    auto [It, Inserted] = Slots->try_emplace(*TypeID, *Offset);
    if (!Inserted && It->second != *Offset)
      invalidMetadata(Twine("type id has multiple offsets on ") + GV.getName());
  }
  return !Info.BroadSlots.empty();
}

bool addExactTypeMetadata(DescriptorInfo &Info,
                          const InterfaceMethodDecl &Method,
                          unsigned TypeKind) {
  auto Broad = Info.BroadSlots.find(Method.BroadTypeID);
  if (Broad == Info.BroadSlots.end())
    return false;
  uint64_t Offset = Broad->second;
  auto [Existing, Inserted] =
      Info.ExactSlots.try_emplace(Method.ExactTypeID, Offset);
  if (!Inserted) {
    if (Existing->second != Offset)
      invalidMetadata(Twine("exact type id has multiple offsets on ") +
                      Info.GV->getName());
    return false;
  }

  LLVMContext &Ctx = Info.GV->getContext();
  Metadata *Ops[] = {
      ConstantAsMetadata::get(ConstantInt::get(Type::getInt64Ty(Ctx), Offset)),
      MDString::get(Ctx, Method.ExactTypeID),
  };
  Info.GV->addMetadata(TypeKind, *MDNode::get(Ctx, Ops));
  return true;
}

class LLGOInterfaceMethodTypeIDPass
    : public PassInfoMixin<LLGOInterfaceMethodTypeIDPass> {
public:
  PreservedAnalyses run(Module &M, ModuleAnalysisManager &) {
    bool Changed = false;
    StringSet<> ActiveExactTypeIDs;
    StringMap<SmallVector<TypeCheckSite, 4>> TypeChecksByExactTypeID;
    StringMap<InterfaceDecl> Declarations;
    for (Function &F : M) {
      for (BasicBlock &BB : F) {
        for (Instruction &I : BB) {
          auto *CB = dyn_cast<CallBase>(&I);
          auto TypeID = checkedLoadTypeID(CB);
          unsigned TypeIDArg = 2;
          if (!TypeID) {
            TypeID = typeTestTypeID(CB);
            TypeIDArg = 1;
          }
          if (!TypeID)
            continue;
          bool HasCallInfo = CB->hasFnAttr(InterfaceCallAttr);
          if (!TypeID->starts_with(InterfaceTypeIDPrefix)) {
            if (HasCallInfo)
              invalidMetadata(Twine("attributes on non-interface type id in ") +
                              F.getName());
            continue;
          }
          if (!HasCallInfo)
            invalidMetadata(Twine("missing call attributes in ") + F.getName());

          InterfaceDecl Parsed = parseInterfaceCall(*CB, *TypeID);
          removeInterfaceCallAttrs(
              *CB, static_cast<unsigned>(Parsed.Methods.size()));
          Changed = true;
          auto Existing = Declarations.find(Parsed.TypeID);
          if (Existing == Declarations.end()) {
            Declarations.try_emplace(Parsed.TypeID, std::move(Parsed));
          } else if (!sameDeclaration(Existing->second, Parsed)) {
            invalidMetadata(Twine("conflicting declarations for ") +
                            Existing->getKey());
          }
          ActiveExactTypeIDs.insert(*TypeID);
          TypeChecksByExactTypeID[*TypeID].push_back({CB, TypeIDArg});
        }
      }
    }

    unsigned TypeKind = M.getContext().getMDKindID("type");
    unsigned VCallVisibilityKind =
        M.getContext().getMDKindID("vcall_visibility");

    std::vector<DescriptorInfo> Descriptors;
    StringMap<SmallVector<unsigned, 8>> DescriptorsByBroadTypeID;
    for (GlobalVariable &GV : M.globals()) {
      if (!GV.getMetadata(VCallVisibilityKind))
        continue;
      DescriptorInfo Info;
      Info.GV = &GV;
      if (!collectTypeSlots(GV, TypeKind, Info))
        continue;
      unsigned Index = Descriptors.size();
      for (const auto &Slot : Info.BroadSlots)
        DescriptorsByBroadTypeID[Slot.getKey()].push_back(Index);
      Descriptors.push_back(std::move(Info));
    }

    unsigned AddedTypeIDs = 0;
    unsigned ActiveInterfaces = 0;
    unsigned BroadFallbacks = 0;
    for (const auto &Entry : Declarations) {
      const InterfaceDecl &Decl = Entry.second;
      SmallVector<const InterfaceMethodDecl *, 4> ActiveMethods;
      for (const InterfaceMethodDecl &Method : Decl.Methods) {
        if (ActiveExactTypeIDs.contains(Method.ExactTypeID))
          ActiveMethods.push_back(&Method);
      }
      if (ActiveMethods.empty())
        continue;
      ++ActiveInterfaces;

      const SmallVector<unsigned, 8> *Seed = nullptr;
      for (const InterfaceMethodDecl &Method : Decl.Methods) {
        auto It = DescriptorsByBroadTypeID.find(Method.BroadTypeID);
        if (It == DescriptorsByBroadTypeID.end()) {
          Seed = nullptr;
          break;
        }
        if (!Seed || It->second.size() < Seed->size())
          Seed = &It->second;
      }

      unsigned Implementers = 0;
      if (Seed) {
        for (unsigned DescriptorIndex : *Seed) {
          DescriptorInfo &Info = Descriptors[DescriptorIndex];
          bool Implements = llvm::all_of(
              Decl.Methods, [&](const InterfaceMethodDecl &Method) {
                return Info.BroadSlots.contains(Method.BroadTypeID);
              });
          if (!Implements)
            continue;
          ++Implementers;
          for (const InterfaceMethodDecl *Method : ActiveMethods) {
            if (addExactTypeMetadata(Info, *Method, TypeKind)) {
              Changed = true;
              ++AddedTypeIDs;
            }
          }
        }
      }

      // Some runtime/reflection interfaces are constructed through special
      // paths that do not materialize a complete concrete descriptor in the
      // LTO module. An empty proven set is therefore not enough to conclude
      // that the call is unreachable. Fall back to the old signature-wide
      // capability for those calls; exact refinement remains enabled for every
      // interface with at least one closed-world implementer certificate.
      if (Implementers == 0) {
        for (const InterfaceMethodDecl *Method : ActiveMethods) {
          auto Sites = TypeChecksByExactTypeID.find(Method->ExactTypeID);
          if (Sites == TypeChecksByExactTypeID.end())
            continue;
          for (TypeCheckSite Site : Sites->second) {
            Site.Call->setArgOperand(
                Site.TypeIDArg,
                MetadataAsValue::get(
                    M.getContext(),
                    MDString::get(M.getContext(), Method->BroadTypeID)));
            Changed = true;
            ++BroadFallbacks;
          }
        }
      }

      if (std::getenv("LLGO_LTO_PLUGIN_VERBOSE"))
        errs() << "llgo-lto-plugin: interface " << Decl.TypeID << " has "
               << Implementers << " implementers for " << ActiveMethods.size()
               << " active methods"
               << (Implementers == 0 ? " (broad fallback)" : "") << "\n";
    }

    if (std::getenv("LLGO_LTO_PLUGIN_VERBOSE"))
      errs() << "llgo-lto-plugin: materialized " << AddedTypeIDs
             << " exact interface method type ids for " << ActiveInterfaces
             << " active interfaces; " << BroadFallbacks
             << " type checks used broad fallback\n";
    return Changed ? PreservedAnalyses::none() : PreservedAnalyses::all();
  }
};

} // namespace

namespace llgo {

void addLLGOInterfaceMethodTypeIDPass(ModulePassManager &MPM) {
  MPM.addPass(LLGOInterfaceMethodTypeIDPass());
}

} // namespace llgo
