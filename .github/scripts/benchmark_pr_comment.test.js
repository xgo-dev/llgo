const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { resolvePullRequest, comment } = require('./benchmark_pr_comment.js');

function fixture({ linked = true, state = 'open', sha = 'a'.repeat(40) } = {}) {
  const source = { repo: { full_name: 'cpunion/llgo' }, ref: 'bench', sha };
  const pr = { number: 42, state, head: source,
    base: { repo: { full_name: 'xgo-dev/llgo' }, ref: 'main', sha: 'b'.repeat(40) } };
  const run = { event: 'pull_request', conclusion: 'success', head_sha: 'a'.repeat(40),
    head_repository: source.repo, head_branch: 'bench',
    pull_requests: linked ? [{ number: 42, base: { sha: 'c'.repeat(40) } }] : [] };
  const outputs = {}, calls = [], comments = [];
  const github = {
    rest: {
      pulls: { list: 'pulls', get: async args => { assert.equal(args.pull_number, 42); return { data: pr }; } },
      repos: { compareCommitsWithBasehead: async args => {
        assert.equal(args.owner, 'xgo-dev');
        assert.equal(args.basehead, `${linked ? 'c' : 'b'}`.repeat(40) + '...' + run.head_sha);
        return { data: { merge_base_commit: { sha: 'd'.repeat(40) } } };
      } },
      issues: {
        listComments: 'comments',
        createComment: async args => calls.push(['create', args]),
        updateComment: async args => calls.push(['update', args]),
      },
    },
    paginate: async (method, args) => {
      assert.equal(args.per_page, 100);
      if (method === 'comments') return comments;
      assert.equal(method, 'pulls');
      assert.equal(args.head, 'cpunion:bench');
      return [pr];
    },
  };
  return { github, context: { repo: { owner: 'xgo-dev', repo: 'llgo' }, payload: { workflow_run: run } },
    core: { setOutput: (key, value) => { outputs[key] = value; }, info() {} },
    pr, run, outputs, calls, comments };
}

test('resolve exact fork head, including an empty workflow_run PR association', async () => {
  for (const linked of [true, false]) {
    const f = fixture({ linked });
    const result = await resolvePullRequest(f);
    assert.equal(result.number, 42);
    assert.equal(result.base_sha, (linked ? 'c' : 'b').repeat(40));
    assert.equal(result.baseline_sha, 'd'.repeat(40));
    assert.equal(f.outputs.publish, 'true');
  }
});

test('skip closed, stale, foreign-base, wrong-repository and non-PR runs', async () => {
  for (const alter of [f => { f.pr.state = 'closed'; },
    f => { f.pr.head.sha = 'e'.repeat(40); },
    f => { f.pr.base.repo.full_name = 'other/llgo'; },
    f => { f.pr.head = { ...f.pr.head, repo: { full_name: 'other/llgo' } }; },
    f => { f.pr.head.ref = 'another'; }, f => { f.run.event = 'push'; },
    f => { f.run.conclusion = 'failure'; }]) {
    const f = fixture(); alter(f);
    assert.equal(await resolvePullRequest(f), null);
    assert.equal(f.outputs.publish, undefined);
  }
});

test('create/update only the bot report for this suite and recheck current head', async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'benchmark-comment-'));
  try {
    const marker = '<!-- go-benchmark:llgo-baseline -->';
    const file = path.join(dir, 'report.md');
    fs.writeFileSync(file, marker + '\nresult');
    for (const mode of ['create', 'update', 'stale', 'closed', 'repository']) {
      const f = fixture();
      f.comments.push({ id: 1, user: { login: 'human' }, body: marker });
      f.comments.push({ id: 2, user: { login: 'github-actions[bot]' }, body: '<!-- go-benchmark:llgo-wasm -->' });
      if (mode === 'update') f.comments.push({ id: 3, user: { login: 'github-actions[bot]' }, body: marker });
      if (mode === 'stale') f.pr.head.sha = 'e'.repeat(40);
      if (mode === 'closed') f.pr.state = 'closed';
      if (mode === 'repository') f.pr.head.repo = { full_name: 'other/llgo' };
      await comment(f, { number: 42, sha: 'a'.repeat(40), repository: 'cpunion/llgo', path: file, marker });
      if (['create', 'update'].includes(mode)) {
        assert.equal(f.calls.length, 1);
        assert.equal(f.calls[0][0], mode);
        if (mode === 'update') assert.equal(f.calls[0][1].comment_id, 3);
      } else assert.equal(f.calls.length, 0);
    }
    fs.writeFileSync(file, 'missing marker');
    await assert.rejects(comment(fixture(), { number: 42, sha: 'a'.repeat(40),
      repository: 'cpunion/llgo', path: file, marker }), /missing its marker/);
  } finally { fs.rmSync(dir, { recursive: true }); }
});
