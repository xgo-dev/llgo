# Using LLGo Windows releases

Choose the release matching your host architecture (`386`, `amd64` or `arm64`) and native ABI (`msvc` or `mingw`). The release workflow produces a ZIP for each variant; check the [release assets](https://github.com/xgo-dev/llgo/releases) for the formats available in your version. Extract the entire archive and add its `bin` directory to PATH.

For each architecture, the two ABI archives contain the same `llgo.exe`. The native C/C++ toolchain selects the ABI used by generated programs.

The ZIP (`llgo<VERSION>.windows-<ARCH>-<ABI>.zip`) contains LLGo, its DLL dependencies, `runtime`, `targets`, documentation and licenses. There is one compact archive per architecture/ABI; native C/C++ tools are installed separately. The matching, checksum-pinned ESP runtime archive is included under `crosscompile`.

The first ESP build verifies the bundled archive and unpacks it locally into the user cache, keyed by version and host architecture. Subsequent builds reuse that cache. Keeping the original xz archive preserves its tool aliases without duplicating large executables in ZIP. The toolchain includes its LLVM and component licenses. Extraction uses 7-Zip or a Windows tar with built-in xz support. Target support sources may still need network access.

The `386` archive contains a PE32 LLGo compiler and PE32 ESP tools. ARM64 archives contain native ARM64 ESP tools. The ESP tools' MinGW runtime does not select the ABI of user programs: use the native MSVC or MinGW setup below.

The installer uses the OS architecture by default. Set `LLGO_ARCH=386` to
select the x86 archive on x64 Windows. Automatic development-dependency setup
for x86 is unavailable; its installer defaults to `LLGO_INSTALL_DEPS=0`.
Configure Go, a maintained i686 C/C++ toolchain and matching native libraries
separately. The CI target setup in `.github/actions/setup-llvm-mingw`
records the pinned MinGW dependency recipe; it does not require a discontinued
MSYS2 CLANG32 installation.

The PowerShell installer prepends LLGo's `bin` directory and, when it installs
dependencies, Go, native LLVM, and vcpkg or MSYS2 tool directories to the
current process and persistent user PATH. This can change which `go`, `clang`,
or `pkg-config` a later terminal selects from the user PATH. Set
`LLGO_UPDATE_PATH=0` before installing to leave the persistent user PATH
unchanged; the installer still adds these paths to its own process to verify
the installation.

| Operation | Additional requirements |
|---|---|
| `llgo version` | No Go or native Clang. Keep the packaged DLLs beside `llgo.exe`. |
| Compile native Windows programs | Go, native Clang/Clang++ and a linker, matching SDK/CRT and native libraries, and pkg-config. |
| Compile ESP firmware | Go and 7-Zip (or a Windows tar with built-in xz support); the ESP toolchain is bundled and unpacked locally; target libraries such as newlib and compiler-rt may need downloading. |

Do not use `crosscompile/clang` as your native Windows Clang. Bundled ESP tools and their DLLs match the release host architecture; ARM64 hosts run them natively. Use one native ABI profile at a time; mixing MSVC and MinGW headers, libraries, DLLs, or pkg-config metadata can break builds.

## Versions and shared dependencies

The v1.0.2 release validation used Go **1.27.0** and native LLVM **22.1.8**. These are tested versions, not a claim about the minimum compatible versions. See the [v1.0.2 setup-deps action](https://github.com/xgo-dev/llgo/blob/v1.0.2/.github/actions/setup-deps/action.yml); for another release, select its tag to find the matching setup.

Native runtime dependencies include **BDWGC** and **libffi**. Depending on the standard-library and C/C++ packages used, also install **OpenSSL, libuv, zlib, cJSON, and SQLite**. The release tests install this complete set, including headers, link libraries, runtime DLLs, and pkg-config metadata.

Go may need network access for an empty module cache. ESP builds may also need to download target support sources. Installing LLGo alone does not make every first build offline.

## MSVC profile

1. Install Visual Studio **2022 Build Tools**, including C++ build tools and a Windows SDK. For ARM64, include the ARM64 compiler tools and SDK libraries.
2. Install native **LLVM 22.1.8** for your host. Prebuilt LLGo needs the native Clang drivers and linker; it does not require rebuilding LLGo or linking LLVM's development libraries yourself.
3. Install the native libraries with vcpkg using `x64-windows`, `arm64-windows`, or `x86-windows`. Use the [release's vcpkg manifest](https://github.com/xgo-dev/llgo/blob/v1.0.2/.github/windows/vcpkg/vcpkg.json) and baseline. In v1.0.2, the baseline is `ddd0023b0eee70986e42ed49d9d4afb8098f212e` and libffi is pinned to **3.4.6**. For x86, the [setup-deps action](.github/actions/setup-deps/action.yml) records the native Clang, `i686-pc-windows-msvc` target, compiler-rt builtins, and vcpkg configuration used by CI.
4. Enter the Visual Studio developer shell for the target architecture, providing `INCLUDE`, `LIB`, SDK tools, and CRT libraries.
5. Add Go, native LLVM, LLGo, and the vcpkg triplet's `bin` directory to the current terminal PATH. Select the native ABI explicitly:

```powershell
# For amd64; use aarch64-pc-windows-msvc for arm64.
$env:CC = 'clang --target=x86_64-pc-windows-msvc -fuse-ld=lld -fms-runtime-lib=dll'
$env:CXX = 'clang++ --target=x86_64-pc-windows-msvc -fuse-ld=lld -fms-runtime-lib=dll -std=c++17'
```

The VC++ Redistributable alone does not provide the SDK, headers, or link libraries. Git, CMake, Ninja, and vcpkg are tools used to prepare the dependencies; they are not all required simply to run the prebuilt LLGo executable.

## MinGW profile

Use MSYS2 **CLANG64** for amd64 or **CLANGARM64** for arm64. Do not substitute the GCC-based MINGW64 environment for this LLVM profile.

For `386`, use the maintained i686 toolchain and native dependency recipe in
[setup-llvm-mingw](.github/actions/setup-llvm-mingw/action.yml), activated through
[setup-deps](.github/actions/setup-deps/action.yml). The CLANG64 package commands
below describe amd64; they do not provide the x86 environment.

For the v1.0.2 tested toolchain, Clang, clang-libs, compiler-rt, LLVM, llvm-libs, llvm-tools, and LLD use package revision **22.1.8-2**; libc++ and libunwind use **22.1.8-1**. The [setup-deps action](https://github.com/xgo-dev/llgo/blob/v1.0.2/.github/actions/setup-deps/action.yml) contains the pinned package URLs. Install headers/CRT and the native libraries from the same MSYS2 environment, along with pkgconf. The v1.0.2 MSVC libffi pin does not apply to MinGW; MinGW validation also passed with libffi 3.8.0.

MSYS2 repositories change over time. Check installed package versions; an unqualified update to the newest LLVM does not reproduce the tested LLVM 22 configuration.

For ordinary PowerShell, add Go, LLGo, and your `msys64/clang64/bin` (or `clangarm64/bin`) to the current terminal PATH, then select:

```powershell
$env:CC = 'clang'
$env:CXX = 'clang++'
clang -dumpmachine
# amd64: x86_64-w64-windows-gnu
```

The native PowerShell build does not need MSYS2's POSIX `usr/bin` first on PATH. MSYS2's native Clang supplies the matching default header and library search paths.

## pkg-config in PowerShell

LLGo invokes `pkg-config`. If only `pkgconf.exe` is installed, or you want to select one profile's metadata explicitly, create `pkg-config.cmd` in a private tools directory and add that directory to the current terminal PATH. Replace these example paths with your actual installation paths.

MSVC/vcpkg:

```bat
@echo off
"C:\dev\vcpkg-installed\x64-windows\tools\pkgconf\pkgconf.exe" --dont-define-prefix "--with-path=C:\dev\vcpkg-installed\x64-windows\lib\pkgconfig" "--with-path=C:\dev\vcpkg-installed\x64-windows\share\pkgconfig" %*
```

MinGW/CLANG64:

```bat
@echo off
"C:\msys64\clang64\bin\pkgconf.exe" --define-prefix "--with-path=C:\msys64\clang64\lib\pkgconfig" %*
```

Use `--define-prefix` for MSYS2 packages in PowerShell so paths such as `/clang64/include/cjson` relocate to the actual Windows installation. Do not copy the vcpkg `--dont-define-prefix` setting into that environment. Clear unrelated `PKG_CONFIG_PATH`/`PKG_CONFIG_LIBDIR` overrides, and keep each wrapper in its own profile's activation environment.

## Check the activated environment

```powershell
go version
llgo version
clang --version
clang -dumpmachine
pkg-config --modversion bdw-gc libffi openssl
pkg-config --cflags --libs bdw-gc libffi openssl libcjson
llgo build -o hello.exe .
.\hello.exe
```

Run the generated program as well as compiling it. Its DLL dependencies are separate from those of `llgo.exe`; successful execution with the activated PATH does not imply the generated EXE can be copied alone to another machine.

## WinGet packaging

The native WinGet packages use a portable `llgo` command alias for both ABI profiles. Release tests run the shared compiler through a symbolic link outside the installation directory with an isolated PATH, and compile native programs through the same alias. Older MinGW manifests used `ArchiveBinariesDependOnPath: true` for DLL lookup; do not remove it from an older package without verifying that package's compiler and DLL dependencies.

The repository can [prepare WinGet candidates](https://github.com/xgo-dev/llgo/blob/main/.github/windows/WINGET.md) from tested release ZIPs. Candidate generation does not publish a package to the public WinGet source; check publication status before using a package ID. Each candidate explains the external development dependencies rather than installing a complete native development environment.
