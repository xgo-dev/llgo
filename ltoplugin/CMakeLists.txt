cmake_minimum_required(VERSION 3.20)

project(LLGOLTOPlugin LANGUAGES C CXX)

find_package(LLVM REQUIRED CONFIG)

if(NOT LLVM_VERSION_MAJOR EQUAL 22)
  message(FATAL_ERROR "LLGo LTO plugin requires LLVM 22.x, found ${LLVM_PACKAGE_VERSION}")
endif()

message(STATUS "Found LLVM ${LLVM_PACKAGE_VERSION}")
message(STATUS "Using LLVMConfig.cmake in: ${LLVM_DIR}")

list(APPEND CMAKE_MODULE_PATH "${LLVM_CMAKE_DIR}")
include(AddLLVM)
add_definitions(${LLVM_DEFINITIONS})

add_llvm_pass_plugin(LLGOLTOPlugin
  LLGOLTOPlugin.cpp
  LLGOInterfaceMethodTypeIDPass.cpp
  LLGOReflectMethodByNamePass.cpp
  LLGOTypeIDExportCleanupPass.cpp
)

target_compile_features(LLGOLTOPlugin PRIVATE cxx_std_17)
target_include_directories(LLGOLTOPlugin PRIVATE ${LLVM_INCLUDE_DIRS})
