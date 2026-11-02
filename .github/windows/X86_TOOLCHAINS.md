# Windows x86 release toolchains

Official upstream LLVM 22 does not publish the native i686 development SDK
needed for a 32-bit LLGo host. LLGo downloads the Windows x86 LLVM SDK from
[xgo-dev/espressif-llvm-project-prebuilt's 22.1.4_20261020 release](https://github.com/xgo-dev/espressif-llvm-project-prebuilt/releases/tag/22.1.4_20261020).
ESP payloads for all seven supported hosts have a separate runtime release pin. The producer commits are recorded in
[`espressif.json`](../../internal/crosscompile/espressif.json).

[`espressif.json`](../../internal/crosscompile/espressif.json) pins the releases, asset names, and independent SHA-256
values. Compiler builds, release staging and fallback source-build downloads
consume these pins. Release staging checks downloaded and restored archives
against the committed hash and extracts a fresh copy; release checksum sidecars
and extracted directory markers are not trust anchors. The compiler verifies
the archive on first installation, then reuses its extracted toolchain cache.
LLGo consumes published runtime assets independently of its development SDKs.

The host SDK deliberately uses upstream LLVM 22.1.8, matching LLGo's other
host SDKs. The separate ESP tools use Espressif LLVM 22.1.4 and run out of
process; their libraries are not linked into LLGo. The x86 SDK's manifest must
still identify the pinned source revision, i686 host, and static link mode.
The compiler embeds the ESP version and SHA-256 pins from this file. Release packages carry the ESP archive; the first embedded build verifies and
unpacks it locally. Source checkouts without a bundle retain the download fallback. The host compiler recipe uses `sdk.release`, so a
packaging-only ESP update can retain the existing, validated LLVM 22.1.8 SDK.

To update the pins, wait for the producer release build and native Windows
payload/SDK validation to pass, confirm the complete release asset list, then
verify downloaded bytes before committing their hashes. SDK and ESP payload
checksums and release tags are independent. Windows archive cache keys use the
selected asset's SHA-256. The shared Unix cache
uses the Unix payload digests and restores unchanged archives across updates;
unrelated SDK or host pin edits do not force their download again.

Compiler, packaging, and native artifact tests remain separate for x86 and
amd64/arm64. WinGet candidates and publication require all six compact Windows ZIPs
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
