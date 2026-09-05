# LLGo LTO Plugin

This directory contains the optional LLVM new pass manager plugin used by LLGo
full LTO builds. It is not required to build or use LLGo.

Build with the same LLVM 22 toolchain used by LLGo:

```sh
cmake -S ltoplugin -B ltoplugin/build \
  -DLLVM_DIR=/path/to/llvm-22/lib/cmake/llvm \
  -DCMAKE_BUILD_TYPE=Release
cmake --build ltoplugin/build
```

Load the plugin when building with full LTO on Linux/ELF or macOS/Mach-O:

```sh
llgo build -lto=full -lto-pass-plugin=/path/to/LLGOLTOPlugin.so ./...
```

On macOS the build produces `LLGOLTOPlugin.dylib`:

```sh
llgo build -lto=full -lto-pass-plugin=/path/to/LLGOLTOPlugin.dylib ./...
```

The plugin must match the host architecture and the LLVM toolchain used by the
linker.

The plugin registers `llgo-lto-pre-globaldce` and also inserts that pass through
LLVM's full LTO early extension point, so loading the plugin is enough for the
pass to run before the normal full LTO optimization pipeline proceeds.

LLGo forwards the plugin path through lld's `--load-pass-plugin` option using
Clang's `-Xlinker` forwarding, including paths containing spaces or commas.
Full LTO selects `-fuse-ld=lld`; on macOS this requires LLVM 22 `ld64.lld`
on the toolchain's search path. Apple's system linker does not support this
option.

Windows targets are not supported by the plugin yet. LLVM 22's MSVC-style
`lld-link` and MinGW lld driver do not expose a new pass manager LTO plugin
loading option. Ordinary full and thin LTO remain available on Windows.
Enabling plugins there requires linker-side support and a compatible Windows
plugin build; forwarding the ELF flag or loading a frontend pass is insufficient.

The relevant LLVM 22 interfaces are
[Mach-O options](https://github.com/llvm/llvm-project/blob/llvmorg-22.1.8/lld/MachO/Options.td),
[Mach-O LTO configuration](https://github.com/llvm/llvm-project/blob/llvmorg-22.1.8/lld/MachO/LTO.cpp),
[COFF options](https://github.com/llvm/llvm-project/blob/llvmorg-22.1.8/lld/COFF/Options.td),
and [MinGW options](https://github.com/llvm/llvm-project/blob/llvmorg-22.1.8/lld/MinGW/Options.td).
