#include "native.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/IR/Attributes.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/DataLayout.h"
#include "llvm/IR/DerivedTypes.h"
#include "llvm/IR/Function.h"
#include "llvm/IR/Intrinsics.h"
#include "llvm/IR/Module.h"
#include "llvm/Support/Error.h"
#include <algorithm>
#include <cstdlib>
#include <cstring>
#include <numeric>

using namespace llvm;

static char *copyString(StringRef text) {
  char *out = static_cast<char *>(std::malloc(text.size() + 1));
  std::memcpy(out, text.data(), text.size());
  out[text.size()] = 0;
  return out;
}

LLGOIntrinsicResult llgo_intrinsic_resolve(unsigned id, void *type,
                                           void *module,
                                           const char *required_name) {
  LLGOIntrinsicResult result{};
  auto fail = [&](StringRef message) {
    result.error = copyString(message);
    return result;
  };
  if (!id || id >= Intrinsic::num_intrinsics)
    return fail("unknown LLVM intrinsic");
  auto *ft = dyn_cast<FunctionType>(static_cast<Type *>(type));
  if (!ft)
    return fail("expected an LLVM function type");
  SmallVector<Type *, 8> overloads;
  if (!Intrinsic::getIntrinsicSignature(id, ft, overloads))
    return fail("signature does not match LLVM's intrinsic signature");
  auto &context = ft->getContext();
  auto *canonical = Intrinsic::getType(context, id, overloads);
  if (ft != canonical)
    return fail("signature does not match LLVM's canonical intrinsic type");
  Module scratch("intrinsic.types", context);
  auto *mod = module ? static_cast<Module *>(module) : &scratch;
  result.name = copyString(Intrinsic::getName(id, overloads, mod, canonical));
  if (required_name && StringRef(result.name) != required_name)
    return fail("explicit intrinsic name does not match destination module");
  auto attrs = Intrinsic::getAttributes(context, id, canonical);
  SmallVector<unsigned, 4> immargs;
  for (unsigned i = 0; i < ft->getNumParams(); ++i)
    if (attrs.hasParamAttr(i, Attribute::ImmArg))
      immargs.push_back(i);
  result.immarg_count = immargs.size();
  if (!immargs.empty()) {
    result.immargs =
        static_cast<unsigned *>(std::malloc(sizeof(unsigned) * immargs.size()));
    std::memcpy(result.immargs, immargs.data(),
                sizeof(unsigned) * immargs.size());
  }
  result.type = canonical;
  if (module) {
    if (auto *existing = mod->getNamedValue(result.name)) {
      auto *fn = dyn_cast<Function>(existing);
      if (!fn || fn->getFunctionType() != canonical || !fn->isDeclaration() ||
          fn->getCallingConv() != CallingConv::C)
        return fail("conflicting LLVM intrinsic declaration");
    }
    result.declaration = Intrinsic::getOrInsertDeclaration(mod, id, overloads);
  }
  return result;
}

void llgo_intrinsic_dispose(LLGOIntrinsicResult result) {
  std::free(result.name);
  std::free(result.error);
  std::free(result.immargs);
}

uint64_t llgo_intrinsic_array_length(void *type) {
  return cast<ArrayType>(static_cast<Type *>(type))->getNumElements();
}
unsigned llgo_intrinsic_scalar_bits(void *type) {
  return static_cast<Type *>(type)->getScalarSizeInBits();
}
int llgo_intrinsic_vector_shape(void *type, unsigned *minimum, int *scalable) {
  auto *vec = dyn_cast<VectorType>(static_cast<Type *>(type));
  if (!vec)
    return 0;
  auto count = vec->getElementCount();
  *minimum = count.getKnownMinValue();
  *scalable = count.isScalable();
  return 1;
}
void *llgo_intrinsic_vector_type(void *element, unsigned minimum,
                                 int scalable) {
  return VectorType::get(static_cast<Type *>(element),
                         ElementCount::get(minimum, scalable));
}
int llgo_intrinsic_integer_range(void *value, uint64_t low, uint64_t high,
                                 int unsigned_range) {
  auto *integer = dyn_cast<ConstantInt>(static_cast<Value *>(value));
  if (!integer)
    return 0;
  unsigned width = std::max(64u, integer->getBitWidth());
  if (unsigned_range) {
    auto actual = integer->getValue().zextOrTrunc(width);
    return actual.uge(APInt(64, low).zextOrTrunc(width)) &&
           actual.ult(APInt(64, high).zextOrTrunc(width));
  }
  auto actual = integer->getValue().sextOrTrunc(width);
  return actual.sge(APInt(64, low).sextOrTrunc(width)) &&
         actual.slt(APInt(64, high).sextOrTrunc(width));
}
unsigned llgo_intrinsic_pointer_index_bits(void *type, const char *layout) {
  auto parsed = DataLayout::parse(layout);
  if (!parsed) {
    consumeError(parsed.takeError());
    return 0;
  }
  auto *ty = static_cast<Type *>(type);
  return ty->isPtrOrPtrVectorTy() ? parsed->getIndexTypeSizeInBits(ty) : 0;
}

unsigned llgo_intrinsic_count(void) { return Intrinsic::num_intrinsics; }
int llgo_intrinsic_is_target(unsigned id) {
  return Intrinsic::isTargetIntrinsic(id);
}
const char *llgo_intrinsic_base_name(unsigned id, size_t *length) {
  auto name = Intrinsic::getBaseName(id);
  *length = name.size();
  return name.data();
}

size_t llgo_intrinsic_descriptors(unsigned id, LLGOIntrinsicDescriptor *out) {
  SmallVector<Intrinsic::IITDescriptor, 16> entries;
  Intrinsic::getIntrinsicInfoTableEntries(id, entries);
  if (!out)
    return entries.size();
  using D = Intrinsic::IITDescriptor;
  for (size_t i = 0; i < entries.size(); ++i) {
    auto d = entries[i];
    LLGOIntrinsicDescriptor r{};
    switch (d.Kind) {
    case D::Void:
      r.kind = "void";
      break;
    case D::VarArg:
      r.kind = "vararg";
      break;
    case D::Token:
      r.kind = "token";
      break;
    case D::Metadata:
      r.kind = "metadata";
      break;
    case D::Double:
      r.kind = "double";
      break;
    case D::Integer:
      r.kind = "integer";
      r.extra = d.Integer_Width;
      break;
    case D::Pointer:
      r.kind = "pointer";
      r.extra = d.Pointer_AddressSpace;
      break;
    case D::Struct:
      r.kind = "struct";
      r.extra = d.Struct_NumElements;
      break;
    case D::Argument:
      r.kind = "argument";
      r.argument = d.getArgumentNumber();
      r.argument_kind = d.getArgumentKind();
      break;
    case D::SameVecWidthArgument:
      r.kind = "shape";
      r.argument = d.getArgumentNumber();
      break;
    case D::VecElementArgument:
      r.kind = "element";
      r.argument = d.getArgumentNumber();
      break;
    case D::OneNthEltsVecArgument:
      r.kind = "divide_lanes";
      r.argument = d.getRefArgNumber();
      r.extra = d.getVectorDivisor();
      break;
    case D::VecOfAnyPtrsToElt:
      r.kind = "pointer_vector";
      r.argument = d.getOverloadArgNumber();
      r.extra = d.getRefArgNumber();
      break;
    default:
      r.kind = "unhandled";
      r.extra = d.Kind;
      break;
    }
    out[i] = r;
  }
  return entries.size();
}

void *llgo_intrinsic_sample(void *context, unsigned id, unsigned variant) {
  auto &ctx = *static_cast<LLVMContext *>(context);
  SmallVector<Intrinsic::IITDescriptor, 16> entries;
  Intrinsic::getIntrinsicInfoTableEntries(id, entries);
  using D = Intrinsic::IITDescriptor;
  unsigned count = 4, size = 0;
  for (auto d : entries) {
    if (d.Kind == D::OneNthEltsVecArgument)
      count = std::lcm(count, 4 * d.getVectorDivisor());
    if (d.Kind == D::Argument && d.getArgumentKind() != D::AK_MatchType)
      size = std::max(size, d.getArgumentNumber() + 1);
    if (d.Kind == D::VecOfAnyPtrsToElt)
      size = std::max(size, d.getOverloadArgNumber() + 1);
  }
  SmallVector<Type *, 8> types(size, nullptr);
  auto *i32 = Type::getInt32Ty(ctx);
  auto *f32 = Type::getFloatTy(ctx);
  auto vector = [&](Type *element) {
    return VectorType::get(element, ElementCount::get(count, variant == 3));
  };
  for (auto d : entries) {
    if (d.Kind != D::Argument || d.getArgumentKind() == D::AK_MatchType)
      continue;
    Type *type = nullptr;
    switch (d.getArgumentKind()) {
    case D::AK_Any:
      if (variant == 1)
        type = ArrayType::get(Type::getInt8Ty(ctx), 3);
      else if (variant == 2)
        type = StructType::get(ctx, {i32, Type::getDoubleTy(ctx)});
      else if (variant == 3)
        type = ArrayType::get(Type::getInt8Ty(ctx), 0);
      else
        type = i32;
      break;
    case D::AK_AnyInteger:
      type = variant ? static_cast<Type *>(vector(i32)) : i32;
      break;
    case D::AK_AnyFloat:
      type = variant ? static_cast<Type *>(vector(f32)) : f32;
      break;
    case D::AK_AnyVector:
      type = vector(variant % 2 ? f32 : i32);
      break;
    case D::AK_AnyPointer:
      type = PointerType::get(ctx, variant % 2);
      break;
    default:
      return nullptr;
    }
    types[d.getArgumentNumber()] = type;
  }
  for (auto d : entries) {
    if (d.Kind != D::VecOfAnyPtrsToElt)
      continue;
    auto *source = cast<VectorType>(types[d.getRefArgNumber()]);
    types[d.getOverloadArgNumber()] = VectorType::get(
        PointerType::get(ctx, variant % 2), source->getElementCount());
  }
  for (auto *type : types)
    if (!type)
      return nullptr;
  return Intrinsic::getType(ctx, id, types);
}
