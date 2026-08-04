# LLGo Agent Guide

Automated contributors must read and follow the shared [contribution guide](CONTRIBUTING.md). It defines the development environment, testing expectations, platform validation, code quality rules, and pull request record required for every contributor.

The rules below are additional safeguards for AI agents and other repository automation.

## Scope and working tree safety

- Keep changes within the requested scope and preserve unrelated edits, untracked files, and active worktrees.
- Inspect the complete diff against upstream before submission. Do not overwrite or discard existing work unless explicitly authorized.
- Use focused edits and tests first. Do not rewrite unrelated files, regenerate unrelated fixtures, or broaden a change merely to make validation pass.
- Diagnose baseline failures instead of hiding them with skips, exclusions, weakened checks, or undocumented environment changes.

## Repository and GitHub safety

- Treat `xgo-dev/*` as upstream. Do not push branches or tags directly to an `xgo-dev` repository, and do not merge its pull requests.
- Push code to the contributor's fork, then create or update a pull request against `xgo-dev/llgo:main`. Upstream issues may be created when requested.
- Do not publish upstream releases or change upstream repository settings. Inspect remotes and exact refs before any write operation when ownership is unclear.
- Prefer `gh issue view`, `gh pr view`, and `gh pr checks` for GitHub state. Use `gh api` for review threads, inline comments, check-run details, or fields not exposed by higher-level commands; do not scrape the website.
- Use explicit force-with-lease protection when a requested rebase requires rewriting a fork branch. Stop if the remote head changed unexpectedly.

## Validation and reporting

- Follow [CONTRIBUTING.md](CONTRIBUTING.md#testing-and-validation) for the affected package, nested runtime module, GOROOT cases, and target-specific validation.
- Report the exact commands and platforms exercised. Distinguish execution from build-only checks and state every omitted test with its reason; omission is not a pass.
- Do not claim repository-wide success from a focused test. When a required check cannot run locally, leave it to CI and say so explicitly.
- For generated IR, use the repository's `// LITTEST` and `chore/litgen` workflow described in the contribution guide; do not hand-edit or bulk-regenerate unrelated expectations.

`AGENTS.md` links to this file so supported agents receive the same automation-specific rules.
