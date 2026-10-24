# Preparing WinGet manifests

The Windows release pipeline builds one compiler per architecture and packages it into both ABI ZIPs, producing six tested variants. ZIP directory records are stored without compression, because WinGet's pure ZIP checker rejects directory entries with nonzero compressed data. `release_zip.py` compares every file and directory against the staging tree when building the ZIP and verifies the extracted contents during artifact tests; `test-release.ps1` executes the ZIP's compiler before and after native dependencies are installed.

The `prepare-winget` job emits the `llgo-winget-candidates` artifact after all Windows artifact tests pass. Stable releases then submit one PR per package (`XGo.LLGo.MSVC` and `XGo.LLGo.MinGW`) through `xgo-dev/winget-pkgs`, each with x86, x64 and ARM64 installers. Both expose `llgo`, so users should select one ABI profile on PATH.

## Automatic submission

Fork `microsoft/winget-pkgs` as `xgo-dev/winget-pkgs`. Set the LLGo Actions secret `WINGET_PR_TOKEN` to a classic PAT with `public_repo` scope and fork write access. The submitting account must complete the [Microsoft CLA](https://cla.opensource.microsoft.com). Use `gh secret set WINGET_PR_TOKEN --repo xgo-dev/llgo` to configure or rotate the token.

After a stable `vMAJOR.MINOR.PATCH` release succeeds, [winget.yml](../workflows/winget.yml) verifies all six public asset digests against the published checksum file and renders schema 1.12.0 manifests. It skips published versions and existing PRs. Interrupted submissions reuse their branch after checking its contents; unexpected edits or closed PRs require manual inspection. Microsoft's PR checks perform WinGet installation validation.

Preview or retry an existing release without rebuilding:

```sh
gh workflow run winget.yml --repo xgo-dev/llgo -f tag=v1.0.8 -F dry_run=true
# Submit both packages; results are linked in the workflow summary.
gh workflow run winget.yml --repo xgo-dev/llgo -f tag=v1.0.8 -F dry_run=false
```

## Local candidates and validation

To prepare the same candidates locally with Python 3.11 or newer:

```sh
python .github/windows/prepare-winget.py \
  --artifacts .windows-dist --version 1.2.3 --output .winget
```

`1.2.3` is an example: use the actual release version without `v`. The input directory must contain all six `llgo<VERSION>.windows-{386,amd64,arm64}-{msvc,mingw}.zip` files and their `.zip.sha256` sidecars. The generator hashes the actual ZIP bytes, checks release version/architecture/ABI metadata, requires a common source commit and identical compiler hashes for both profiles of each architecture, and emits standard version/installer/defaultLocale YAML files. The public installer URLs point to the matching `v<VERSION>` GitHub release.

Before submitting to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs):

1. Publish the tested ZIPs through the normal tag release. Confirm every public URL serves the bytes whose hashes appear in the candidates. PR/snapshot artifacts alone do not establish that those URLs exist.
2. Run `winget validate --manifest <version-directory>` for each profile, then test real installation, command startup, upgrade, and uninstall on a clean Windows environment. Keep WinGet's archive and hash checks enabled.
3. Submit the reviewed `manifests/x/XGo/LLGo/...` directories, or use the workflow above.

The MinGW installer manifest sets `ArchiveBinariesDependOnPath: true`; MSVC retains the portable alias. Both point at `bin/llgo.exe` and include the external dependency notes from [WINDOWS.md](../../WINDOWS.md). No complete SDK/toolchain bundle is implied.

Do not copy local validation `ProductCode` values ending in `__DefaultSource` into public manifests. A synthetic packaging upgrade using unchanged compiler bytes validates only the local lifecycle. In the local WinGet 1.29.290 test, upgrading a portable package moved it from the custom location to WinGet's default location; do not promise custom-location preservation from that test.
