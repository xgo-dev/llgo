# Windows x86 release toolchains

Official upstream LLVM 22 does not publish the native i686 development SDK
needed for a 32-bit LLGo host. LLGo downloads the Windows x86 LLVM SDK from
[xgo-dev/espressif-llvm-project-prebuilt's 22.1.4_20261020 release](https://github.com/xgo-dev/espressif-llvm-project-prebuilt/releases/tag/22.1.4_20261020).
The same release supplies ESP tools for all three Windows host architectures.
The producer's merge commit is recorded in `espressif-toolchain.json`.

`espressif-toolchain.json` pins the release, asset names, and independent SHA-256
values. Both compiler building and packaging consume these pins. Downloads
and cache hits are checked against the committed hash before extraction;
release checksum sidecars and extracted directory markers are not trust anchors.
Each use extracts a fresh copy. LLGo no longer builds these toolchains from a
personal fork or transfers source-built archives between CI jobs.

The host SDK deliberately uses upstream LLVM 22.1.8, matching LLGo's other
host SDKs. The separate ESP tools use Espressif LLVM 22.1.4 and run out of
process; their libraries are not linked into LLGo. The x86 SDK's manifest must
still identify the pinned source revision, i686 host, and static link mode.
`release.json` records the pinned ESP version and SHA-256 for every ABI archive.

To update the pins, wait for the producer release build and native Windows
payload/SDK validation to pass, confirm the complete release asset list, then
verify downloaded bytes before committing their hashes. SDK and ESP payload
checksums are distinct even though they share a release tag. Changes to the
pin file invalidate the archive caches without disabling integrity checks.

Compiler, packaging, and native artifact tests remain separate for x86 and
amd64/arm64. WinGet candidates and publication require all six Windows ZIPs
and all architecture tests to pass.

The llvm-mingw runtime dependencies apply `bdwgc-windows-fault-guard.patch` to
the pinned BDWGC 8.2.12 port. Its GNU scan guard calls CRT `signal()`, whose
first use on a collecting thread can allocate from the process heap after the
collector suspends a thread holding that heap's lock
([BDWGC #924](https://github.com/bdwgc/bdwgc/issues/924)). The patch prepares
and restores that thread's signal state before stopping the world, so scanning
reuses the CRT's existing exception-action table.
`test-bdwgc-heap-lock.c` covers collection on a fresh thread while another
registered thread holds the heap lock for a bounded interval, as well as
application exception-handler preservation. The native MinGW x86 job runs the
check with an external timeout. Dependency cache keys include the patch and
port preparation script; upgrading BDWGC requires reviewing the patch.
