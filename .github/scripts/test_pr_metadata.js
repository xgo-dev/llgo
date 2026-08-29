'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const {parseTemplate, validatePullRequest} = require('./pr_metadata');

const VALID_BODY = `## Summary

Require staged CI for upstream pull requests.

## Validation

- Workflow lint passed in the fork.

## Fork CI

- [x] I created a test PR in my fork and its relevant CI passed.
- Fork test PR: https://github.com/contributor/llgo/pull/42
`;

function githubWithHead({ref = 'staged-ci', sha = 'abc123'} = {}) {
  return {
    rest: {
      repos: {
        get: async () => ({
          data: {fork: true, parent: {full_name: 'xgo-dev/llgo'}},
        }),
      },
      pulls: {
        get: async () => ({
          data: {
            head: {
              repo: {full_name: 'contributor/llgo'},
              ref,
              sha,
            },
          },
        }),
      },
    },
  };
}

const SOURCE = {
  head: {
    repo: {full_name: 'contributor/llgo'},
    ref: 'staged-ci',
    sha: 'abc123',
  },
};

test('parses a completed pull request template', () => {
  assert.deepEqual(parseTemplate(VALID_BODY), {
    url: 'https://github.com/contributor/llgo/pull/42',
    owner: 'contributor',
    repo: 'llgo',
    number: 42,
  });
});

test('rejects missing template content and unchecked evidence', () => {
  assert.throws(
    () => parseTemplate(VALID_BODY.replace('Require staged CI for upstream pull requests.', '<!-- unchanged -->')),
    /Fill in the Summary section/,
  );
  assert.throws(
    () => parseTemplate(VALID_BODY.replace('- [x]', '- [ ]')),
    /fork-CI confirmation/,
  );
});

test('accepts evidence from the same repository, branch, and commit', async () => {
  const evidence = await validatePullRequest({
    body: VALID_BODY,
    upstreamRepository: 'xgo-dev/llgo',
    source: SOURCE,
    github: githubWithHead(),
  });
  assert.equal(evidence.number, 42);
});

test('rejects a fork PR for another branch or commit', async () => {
  await assert.rejects(
    validatePullRequest({
      body: VALID_BODY,
      upstreamRepository: 'xgo-dev/llgo',
      source: SOURCE,
      github: githubWithHead({ref: 'other-branch'}),
    }),
    /branch other-branch does not match contribution branch staged-ci/,
  );
  await assert.rejects(
    validatePullRequest({
      body: VALID_BODY,
      upstreamRepository: 'xgo-dev/llgo',
      source: SOURCE,
      github: githubWithHead({sha: 'outdated'}),
    }),
    /checks outdated, but this contribution uses abc123/,
  );
});
