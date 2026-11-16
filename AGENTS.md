# Working on LLGo

Read [CONTRIBUTION.md](CONTRIBUTION.md) for environment setup, repository
structure, test selection, and submission checks. Keep those instructions there.

## Make the repository a reliable reference

- Inspect the affected implementation, callers, tests, and CI before editing.
  Check existing patterns against the intended semantics before copying them.
- Fix the cause of a failure. Remove superseded paths and temporary scaffolding
  introduced by the change; avoid parallel implementations and silent fallbacks
  that conceal invalid states.
- When a correction exposes a repeatable mistake, add a focused regression,
  assertion, lint rule, or type constraint at the boundary that can enforce it.
  Prefer executable checks over adding another paragraph to this file.
- Use the supported APIs and repository tooling. Reject unsupported cases
  explicitly rather than bypassing validation or adding unexplained exceptions.
- Keep comments for contracts, non-obvious invariants, ABI constraints, and
  compiler/test directives. Explain why where code cannot; avoid narrating code,
  recording work history, or justifying a workaround in place of fixing it.

## Verify the result

- Reproduce a reported bug and add a regression that fails before the fix when
  practical. Run the affected checks, then the integration tests required by
  the changed behavior. Exercise compiler/runtime changes through a program
  compiled with the current checkout, not only host-side unit tests.
- Follow the contribution guide for nested runtime modules, target matrices,
  and IR checks. Review generated output; refresh only the affected fixtures.
- For performance, layout, or concurrency changes, supply matching before/after
  measurements or focused stress checks. Use stronger invariant checks when the
  affected contract warrants them.
- Review the complete diff and validation output before submission. Do not
  weaken assertions, add skips, or exempt production code to obtain a pass.
- Report commands, platforms, and results. Distinguish execution, cross-builds,
  emulator runs, and device tests; state unavailable checks and baseline failures.
  Include screenshots or recordings when a changed visual/debugger interaction
  needs them for review.

## Respect the working tree

- Preserve unrelated edits, untracked files, and other worktrees. Keep each
  contribution focused and inspect remotes and exact refs before publishing.
- Follow the user's repository write policy and granted scope. Use the
  contributor's fork and a pull request for upstream contributions; never infer
  authority to merge, release, or change repository settings from local access.
