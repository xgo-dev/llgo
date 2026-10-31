# Preparing WinGet manifests

The Windows release pipeline builds one compiler per architecture and packages it for both ABIs. Each of the six variants has one compact ZIP named `llgo<VERSION>.windows-<ARCH>-<ABI>.zip`, containing the compiler, bundled DLLs, runtime, targets and licenses. `release.json` identifies `distribution: full`; the matching ESP runtime archive is bundled under `crosscompile`, verified and unpacked locally on the first embedded build.

ZIP directory records are stored without compression for WinGet's pure ZIP checker. Release tests verify every extracted entry, DLL closure, standalone startup through a portable alias with an isolated PATH, native Go/C/C++ compilation, and ESP32/ESP32-C3 firmware compilation with default and Full LTO after local extraction into an empty cache, without downloading ESP tools. Inspection tools travel in CI-only compiler artifacts and are never included in release ZIPs.

The `prepare-winget` job emits the `llgo-winget-candidates` artifact after all Windows artifact tests pass. Stable releases then submit one PR per package (`XGo.LLGo.MSVC` and `XGo.LLGo.MinGW`) through `xgo-dev/winget-pkgs`, each with x86, x64 and ARM64 installers. The shell/PowerShell installers and WinGet select the same compact ZIPs. Both expose `llgo`, so users should select one ABI profile on PATH.

## Automatic submission

Fork `microsoft/winget-pkgs` as `xgo-dev/winget-pkgs`. Set the LLGo Actions secret `WINGET_PR_TOKEN` to a classic PAT with `public_repo` scope and fork write access. The submitting account must complete the [Microsoft CLA](https://cla.opensource.microsoft.com). Use `gh secret set WINGET_PR_TOKEN --repo xgo-dev/llgo` to configure or rotate the token.

After a stable `vMAJOR.MINOR.PATCH` release succeeds, [winget.yml](../workflows/winget.yml) verifies all six public compact ZIP asset digests against the published checksum file and renders schema 1.12.0 manifests. It skips published versions and existing PRs. Interrupted submissions reuse their branch after checking its contents; unexpected edits or closed PRs require manual inspection. Microsoft's PR checks perform WinGet installation validation.

Preview or retry a release that includes the compact ZIPs without rebuilding (replace the example tag with that release):

```sh
gh workflow run winget.yml --repo xgo-dev/llgo -f tag=v1.1.0 -F dry_run=true
# Submit both packages; results are linked in the workflow summary.
gh workflow run winget.yml --repo xgo-dev/llgo -f tag=v1.1.0 -F dry_run=false
```

## Local candidates and validation

To prepare the same candidates locally with Python 3.11 or newer:

```sh
python .github/windows/prepare-winget.py \
  --artifacts .windows-dist --version 1.2.3 --output .winget
```

`1.2.3` is an example: use the actual release version without `v`. The input directory must contain all six `llgo<VERSION>.windows-{386,amd64,arm64}-{msvc,mingw}.zip` files and their `.zip.sha256` sidecars. The generator hashes the actual ZIP bytes, checks release version/architecture/ABI metadata, requires the bundled ESP archive and its matching checksum/host metadata, full distribution metadata, a common source commit and identical compiler hashes for both profiles of each architecture, and emits standard version/installer/defaultLocale YAML files. The public installer URLs point to the matching `v<VERSION>` GitHub release.

Before submitting to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs):

1. Publish the tested ZIPs through the normal tag release. Confirm every public URL serves the bytes whose hashes appear in the candidates. PR/snapshot artifacts alone do not establish that those URLs exist.
2. Run `winget validate --manifest <version-directory>` for each profile, then test real installation, command startup, upgrade, and uninstall on a clean Windows environment. Keep WinGet's archive and hash checks enabled.
3. Submit the reviewed `manifests/x/XGo/LLGo/...` directories, or use the workflow above.

Both installer manifests use the portable `llgo` alias. The shared host compiler is tested through that alias for both ABI profiles, without adding the package directory to PATH. Both point at `bin/llgo.exe` and include the external dependency notes from [WINDOWS.md](../../WINDOWS.md). The ESP runtime is bundled; native Windows SDK/CRT and libraries remain external dependencies.

Do not copy local validation `ProductCode` values ending in `__DefaultSource` into public manifests. A synthetic packaging upgrade using unchanged compiler bytes validates only the local lifecycle. In the local WinGet 1.29.290 test, upgrading a portable package moved it from the custom location to WinGet's default location; do not promise custom-location preservation from that test.

The compact archives address the large local-manifest scan failure reported in [winget-cli#6632](https://github.com/microsoft/winget-cli/issues/6632). Local candidate generation requires the bundled ESP runtime archive and verifies its SHA-256. Local and public ZIPs must fit the repository's 128 MiB package budget; this also rejects older full SDK bundles that use the same unsuffixed filename. Existing published ZIPs are not repacked or replaced. The x86 1.0.9 WinGet entries were temporarily excluded; future ZIPs still require successful x86 WinGet installation validation before publication.
