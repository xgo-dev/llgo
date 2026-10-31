# Unix release payloads

The macOS and Linux tar.gz archives include LLGo's LLVM shared libraries, native
and ESP compiler/linker tools, compiler resources, target runtimes and licenses.
They omit LLVM/Clang development SDK files and benchmark tools. Identical files
use relative symlinks so the tar.gz writer preserves every tool/loader name
without storing duplicate library or executable contents.

All seven runtime payloads are pinned in
[`espressif.json`](../../internal/crosscompile/espressif.json). The release setup
runs `esp-toolchain.py` to verify producer archives and stage their runtime files
directly under `.release-toolchains`. GoReleaser's `stage-toolchains.sh` hook
only generates the Linux system LLVM target-list overlays. LLGo is still built
with the separately pinned complete SDK in `.sysroot`; staging never prunes or
replaces that SDK.

DEB/RPM include LLGo, runtime sources, targets, licenses and the original ESP
runtime archive. The first embedded build verifies and unpacks that archive
locally into the user cache. Native compilation continues to use system
LLVM/Clang/LLD 22 and native libraries. Separate package compilers omit Xtensa
initializers because Fedora's LLVM does not expose that experimental backend.
Their binaries use the system `libLLVM.so.22.1`; Debian/Ubuntu selects
`llvm-config-22`, while Fedora supplies unversioned LLVM commands.

Ubuntu 26.04 and Fedora 44 provide LLVM 22 packages. Earlier Debian/Ubuntu
releases need the matching apt.llvm.org repository enabled first. The release
container uses the same DEB payload and external native dependencies.

CI installs each distribution with declared dependencies, runs a mixed Go/C/C++
program and builds ESP32/ESP32-C3 firmware with default and Full LTO. Tests use
fresh caches and assert that the packaged ESP runtime is used without an ESP
network download. The four tar.gz tests also check the runtime profile and
loader paths. Logs and per-host firmware results are retained as CI artifacts.

Local packaging regression tests:

```sh
python3 -m unittest discover -s .github/unix -p 'test_*.py' -v
```
