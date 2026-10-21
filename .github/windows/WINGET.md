# Preparing WinGet manifests

The Windows release pipeline builds one compiler per architecture and packages it into both ABI ZIPs, producing six tested variants. ZIP directory records are stored without compression, because WinGet's pure ZIP checker rejects directory entries with nonzero compressed data. `release_zip.py` compares every file and directory against the staging tree when building the ZIP and verifies the extracted contents during artifact tests; `test-release.ps1` executes the ZIP's compiler before and after native dependencies are installed.

The `prepare-winget` job emits the `llgo-winget-candidates` workflow artifact only after all Windows artifact tests pass. It does not upload release assets or submit a winget-pkgs PR. The candidates use two proposed package IDs, `XGo.LLGo.MSVC` and `XGo.LLGo.MinGW`, each with x86, x64 and ARM64 installers. Review the IDs before the initial public submission. Both expose `llgo`, so users should select one ABI profile on PATH.

To prepare the same candidates locally with Python 3.11 or newer:

```sh
python .github/windows/prepare-winget.py \
  --artifacts .windows-dist --version 1.2.3 --output .winget
```

`1.2.3` is an example: use the actual release version without `v`. The input directory must contain all six `llgo<VERSION>.windows-{386,amd64,arm64}-{msvc,mingw}.zip` files and their `.zip.sha256` sidecars. The generator hashes the actual ZIP bytes, checks release version/architecture/ABI metadata, requires a common source commit and identical compiler hashes for both profiles of each architecture, and emits standard version/installer/defaultLocale YAML files. The public installer URLs point to the matching `v<VERSION>` GitHub release.

Before submitting to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs):

1. Publish the tested ZIPs through the normal tag release. Confirm every public URL serves the bytes whose hashes appear in the candidates. PR/snapshot artifacts alone do not establish that those URLs exist.
2. Run `winget validate --manifest <version-directory>` for each profile, then test real installation, command startup, upgrade, and uninstall on a clean Windows environment. Keep WinGet's archive and hash checks enabled.
3. Submit the reviewed `manifests/x/XGo/LLGo/...` directories. Candidate package IDs and public-source upgrade behavior still require initial publication review.

The MinGW installer manifest sets `ArchiveBinariesDependOnPath: true`; MSVC retains the portable alias. Both point at `bin/llgo.exe` and include the external dependency notes from [WINDOWS.md](../../WINDOWS.md). No complete SDK/toolchain bundle is implied.

Do not copy local validation `ProductCode` values ending in `__DefaultSource` into public manifests. A synthetic packaging upgrade using unchanged compiler bytes validates only the local lifecycle. In the local WinGet 1.29.290 test, upgrading a portable package moved it from the custom location to WinGet's default location; do not promise custom-location preservation from that test.
