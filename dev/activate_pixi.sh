#!/usr/bin/env bash

export GOFLAGS=-tags=byollvm
export CGO_ENABLED=1
case "$(uname -s)" in
  Linux)
    target="$(uname -m)-conda-linux-gnu"
    export CC="${target}-clang"
    export CXX="${target}-clang++"
    export PATH="$PIXI_PROJECT_ROOT/pixi-tools:$PATH"
    ;;
  Darwin)
    export CC=clang
    export CXX=clang++
    ;;
esac
LLVM_CONFIG="$(command -v llvm-config)"
export LLVM_CONFIG
CGO_CPPFLAGS="$($LLVM_CONFIG --cppflags)"
export CGO_CPPFLAGS
export CGO_CXXFLAGS=-std=c++17
CGO_LDFLAGS="$($LLVM_CONFIG --ldflags --link-shared --libs all --system-libs)"
LLGO_ROOT="$(cd "$PIXI_PROJECT_ROOT/.." && pwd)"
export LLGO_ROOT

# The compiled compiler must find Pixi's shared LLVM library on later runs.
export CGO_LDFLAGS="$CGO_LDFLAGS -Wl,-rpath,$CONDA_PREFIX/lib"
