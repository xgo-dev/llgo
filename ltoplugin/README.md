# LLGo LTO Plugin

This directory contains the optional LLVM new pass manager plugin used by LLGo
full LTO builds. It is not required to build or use LLGo.

Build with the same LLVM 19 or LLVM 21 toolchain selected for LLGo. For an
LLVM 21 build:

```sh
cmake -S ltoplugin -B ltoplugin/build \
  -DLLVM_DIR=/path/to/llvm-21/lib/cmake/llvm \
  -DCMAKE_BUILD_TYPE=Release
cmake --build ltoplugin/build
```

Load the plugin when building with full LTO on Linux/ELF:

```sh
llgo build -lto=full -lto-pass-plugin=/path/to/LLGOLTOPlugin.so ./...
```

On macOS, the built plugin usually uses the `.dylib` suffix.

The plugin registers `llgo-lto-pre-globaldce` and also inserts that pass through
LLVM's full LTO early extension point, so loading the plugin is enough for the
pass to run before the normal full LTO optimization pipeline proceeds.

LLGo forwards the plugin path through lld's `--load-pass-plugin` option. The
supported LLVM 19 and LLVM 21 Mach-O linkers do not expose an equivalent new
pass manager LTO plugin loading option, so LLGo rejects `-lto-pass-plugin` for
Darwin targets until the linker side grows that support.
