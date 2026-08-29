# Contributing to LLGo

LLGo uses fork-first CI so work-in-progress commits do not consume the
`xgo-dev` GitHub Actions capacity.

## Validate a change

1. Sync the default branch of your fork with `xgo-dev/llgo` and enable GitHub
   Actions in the fork.
2. Create a feature branch and push it to the fork. Relevant full CI workflows
   run on that push with the fork's Actions capacity. Documentation-only
   changes run formatting, link, and documentation checks without the general
   compiler and test matrices.
3. Open a test pull request in the fork and wait for its relevant checks to
   pass. Keep this pull request available for the upstream reviewer.
4. Open the upstream pull request from the same feature branch. Complete the
   fork-CI checkbox and paste the fork test PR URL in the template. The
   `review readiness` check verifies the required template format and that
   both pull requests use the same head repository, branch, and commit.

The pull request is not ready for review while `review readiness` fails. After
the lightweight upstream checks pass, a reviewer inspects the change and the
linked fork CI. The reviewer adds `need-review` to authorize the relevant full
CI in `xgo-dev/llgo`. Pushing another commit reruns fork CI and invalidates the
fork-PR evidence until both pull requests point to the same branch and commit
again.

Until `need-review` is present, the upstream `full CI authorization` check
fails intentionally. Repository maintainers must configure both `review
readiness` and `full CI authorization` as required status checks on the default
branch so a pull request cannot merge with an incomplete template or without
reviewer-authorized full CI. The `need-review` label must also be created once
in `xgo-dev/llgo`; only users with triage or write permission should be able to
apply it.

## Request GOROOT compatibility tests

GOROOT is an expensive opt-in suite. Fork labels are not inherited from the
upstream repository, so create the label once in your fork:

```sh
gh label create go-test-compat \
  --repo <your-account>/llgo \
  --description "Go standard-library and GOROOT test compatibility" \
  --color 1D76DB
```

Apply `go-test-compat` to the test PR in your fork to run GOROOT with the
fork's capacity. In `xgo-dev/llgo`, GOROOT runs only when a reviewer has also
added `need-review`. The daily scheduled and manual GOROOT runs are unchanged.
