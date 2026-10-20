# Shared CI policy

Workflows call `ci-prepare.yml` as their `prepare` job. Its outputs are job
outputs, not environment variables: consumers declare `needs: prepare` and use
`needs.prepare.outputs.run_code_ci == 'true'` or
`needs.prepare.outputs.run_doc_checks == 'true'`. Existing artifact dependencies
must stay in `needs` alongside `prepare`.

The implementation is in `scripts/ci_policy.py`. Keep change rules there rather
than adding path filters or independent diff commands to individual workflows.
Each workflow executes its own small prepare job; the rule implementation is
shared. The prepare job needs only Git and Python, not the LLGo toolchain. On a
PR, it loads both `ci_policy.py` and `ci_changes.py` from the event's base
commit, so edits to those modules in the PR cannot change this step's skip
decision. If either base file is unavailable (including before this policy lands
on main), prepare enables both suites. Main, tags and manual runs execute the
checked-out policy and enable both suites.

## Execution decisions

- Only `pull_request` runs may skip checks. Pushes to main, tag builds, scheduled
  runs and manual runs enable both outputs without inspecting a diff. This does
  not add new triggers to schedule-only workflows or enable tag-only publishing
  on branches.
- A PR is compared from the merge base of its event base/head SHAs to its head,
  including every commit and every changed path. Changes subsequently pushed to
  the base branch do not alter that comparison.
- Added/modified, regular non-executable `.md`, `.markdown` and `.rst` files are
  prose. Other extensions and unknown paths require code CI. `doc/_readme/**`
  and `testdata` directories are code/test inputs even when named Markdown.
- Documentation changes, README example inputs and `.github/**` changes enable
  document checks. The existing check still verifies README embedded sources
  and README links; this policy does not expand its scanning scope.
- Deletions, renames and type changes run both suites because they can remove
  linked files or packaged documents. Unavailable or invalid PR diffs also run
  both suites, with a warning. An empty diff is handled conservatively.
- The **Docs** workflow installs LLGo and executes examples, so it consumes
  `run_code_ci`, not `run_doc_checks`.

Prepare also exports `pr_head_sha` and `pr_merge_base` for benchmark consumers.
These are empty on non-PR events. If PR Git history cannot be resolved, code CI
is enabled but benchmark consumers must reject an empty merge base rather than
silently measuring a different commit.

## Runners and concurrency

`linux_runner` and `linux_large_runner` contain JSON runner labels. The upstream
`xgo-dev` repository uses its qiniu runners; workflows running in forks use
GitHub-hosted Ubuntu 24.04. Consume them with `fromJSON(...)`.

Prepare and the independent CI policy check select their runner directly from
the workflow repository owner: they cannot wait for prepare outputs. These are
the only bootstrap exceptions to centralized runner selection, and the contract
tests check that their labels match the policy. CI gates use `linux_runner`,
with a hosted fallback if preparation fails before producing outputs. This
keeps ordinary PR startup and completion jobs out of the hosted runner queue.

The amd64 Mise/Pixi environment checks, release artifact verification, Linux
sysroot generation, snapshot cross-build, package/container validation, and
installer distro containers use `linux_large_runner`. WinGet manifest
preparation uses `linux_runner`. The sysroot job first verifies Docker daemon
access and a privileged container's mount capability, even on cache hits;
cache misses also exercise QEMU/binfmt registration before cross-building.
Native ARM64, Windows and macOS coverage keeps its existing architecture and
runner selection. Public-network link checks and tag publishing keep their
explicitly selected hosted runners.

Release artifact tests explicitly install Clang on Linux for the native
assembly shared-library fixture; `install-llvm: false` does not provide it on
minimal runners. The main Linux LLGo test job uses a private LLDB launcher that
skips initialization files and clears the debuginfod server list. Ubuntu's
system LLDB initialization file otherwise enables its symbol server even when
`DEBUGINFOD_URLS` is empty. Tests retain local symbol lookup without waiting
for external symbol servers.

PR updates share a concurrency group and cancel superseded runs. Main pushes
use a separate main group and also cancel superseded runs. Tags and other
events use `github.run_id` to keep their runs independent. Workflow-level concurrency cannot consume
prepare outputs; `test_ci_workflows.py` enforces the common expression instead.
The reusable prepare workflow has no concurrency group of its own.

Every workflow that uses prepare has a uniquely named `CI gate (...)` job. It
runs even if prepare fails and reports that failure. Branch protection should
require the gates for PR workflows, the relevant code/document checks, and
**CI policy**. Requiring only downstream jobs does not distinguish a prepare
failure from an intentional docs-only skip. The gate checks preparation, not
the result of every downstream job; it does not replace those required checks.

The benchmark publisher checks the triggering run's artifacts separately for
native and WASM results before invoking either publisher. A successful docs-only
run has no benchmark artifacts and is a normal no-op. External benchmark
notifications run for every upstream main push, including documentation changes.

## Release publishing

PR and main builds create Linux DEB/RPM packages and install them in clean
Ubuntu/Fedora containers on amd64 and arm64. They also build and run the release
container images. These checks gate tag publishing and require no Fury token.

Stable `vX.Y.Z` tags upload DEB/RPM packages to Gemfury as public packages after
the release succeeds.
Set the repository Actions secret `FURY_TOKEN` to an upload-only Gemfury Push
token. The upstream `xgo-dev` organization publishes to the Gemfury account
`goplus`. Forks default to their repository owner. Set the repository variable
`FURY_ACCOUNT` to override the target account. Prerelease tags publish their
GitHub assets and container version tags without uploading to Fury.

GHCR publishing uses `GITHUB_TOKEN` with `packages: write`. After the first
publish, set the container package visibility to public to allow anonymous pulls.

## Validation

The independent **CI policy** workflow runs on every PR and main push. It tests
Git histories, failure fallbacks, runner selection, output propagation, matrix
coverage, release dependencies, main concurrency, and missing benchmark artifacts.
It is deliberately not gated by the code it validates.

Locally, install `scripts/requirements-ci.txt` in a Python environment and run:

```sh
python -m unittest discover -s .github/scripts -p 'test_*.py' -v
actionlint -shellcheck= .github/workflows/*.yml
```
