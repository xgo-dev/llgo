#ifndef LLGO_REFLECT_STRING_ANALYSIS_H
#define LLGO_REFLECT_STRING_ANALYSIS_H

#include "llvm/ADT/SmallVector.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/DataLayout.h"
#include "llvm/IR/GlobalVariable.h"
#include "llvm/IR/InstrTypes.h"
#include "llvm/IR/Type.h"

#include <cstdint>
#include <string>

namespace llgo {

inline constexpr char ReflectMethodByNameArgAttr[] =
    "llgo.reflect.methodbyname.name";

// Report whether GV's initializer can be safely read as a constant after full
// LTO, including package-literal mutable globals whose remaining uses are all
// simple reads through value-preserving pointer operations.
bool canReadGlobalInitializer(llvm::GlobalVariable *GV);

// Extract a typed constant at byte offset Offset inside Init, returning nullptr
// if the layout does not match or the offset is out of bounds.
llvm::Constant *constantAtByteOffset(llvm::Constant *Init, uint64_t Offset,
                                     llvm::Type *LoadTy,
                                     const llvm::DataLayout &DL);

// Collect every possible method name passed to a marked reflect MethodByName
// call. Returns false as soon as any incoming name cannot be bounded.
bool collectReflectMethodNameArgs(llvm::CallBase *ReflectCall,
                                  const llvm::DataLayout &DL,
                                  llvm::SmallVectorImpl<std::string> &Names);

} // namespace llgo

#endif // LLGO_REFLECT_STRING_ANALYSIS_H
