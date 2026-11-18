// This module runs only from the publisher's trusted default-branch checkout.
// Downloaded benchmark artifacts are data; PR code is never executed here.
const fs = require('node:fs');

async function resolvePullRequest({ github, context, core }) {
  const run = context.payload.workflow_run;
  if (run.event !== 'pull_request' || run.conclusion !== 'success') return null;
  const repository = `${context.repo.owner}/${context.repo.repo}`;
  const headRepository = run.head_repository?.full_name;
  if (!headRepository || !/^[0-9a-f]{40}$/.test(run.head_sha)) {
    throw new Error('Missing benchmark run source identity');
  }
  // workflow_run.pull_requests may be empty for fork contributions.
  const linked = run.pull_requests || [];
  let candidates = linked;
  if (!candidates.length) {
    candidates = await github.paginate(github.rest.pulls.list, {
      ...context.repo, state: 'open', head: `${headRepository.split('/')[0]}:${run.head_branch}`,
      per_page: 100,
    });
  }
  for (const candidate of candidates) {
    const { data: pr } = await github.rest.pulls.get({
      ...context.repo, pull_number: candidate.number,
    });
    if (pr.state !== 'open' || pr.base.repo.full_name !== repository ||
        pr.head.repo?.full_name !== headRepository || pr.head.ref !== run.head_branch ||
        pr.head.sha !== run.head_sha) continue;
    // Use the target snapshot at measurement time, not a configuration supplied
    // by the PR. If GitHub omitted the association, use the trusted current base.
    const base = candidate.base?.sha || pr.base.sha;
    const { data: comparison } = await github.rest.repos.compareCommitsWithBasehead({
      ...context.repo, basehead: `${base}...${run.head_sha}`,
    });
    const source = {
      number: pr.number, sha: run.head_sha, repository: headRepository,
      branch: run.head_branch, base_sha: base, base_ref: pr.base.ref,
      baseline_sha: comparison.merge_base_commit.sha,
    };
    for (const [key, value] of Object.entries(source)) core.setOutput(key, String(value));
    core.setOutput('publish', 'true');
    return source;
  }
  core.info('Skipping benchmark comments: pull request was closed or its head changed');
  return null;
}

async function comment({ github, context, core }, { number, sha, repository, path, marker }) {
  if (!marker?.startsWith('<!-- go-benchmark:')) throw new Error('Invalid benchmark marker');
  const { data: pr } = await github.rest.pulls.get({ ...context.repo, pull_number: Number(number) });
  // A new commit can arrive while artifacts are being rendered.
  if (pr.state !== 'open' || pr.head.sha !== sha || pr.head.repo?.full_name !== repository) {
    core.info('Skipping stale benchmark report');
    return;
  }
  const body = fs.readFileSync(path, 'utf8');
  if (!body.includes(marker)) throw new Error('Benchmark report is missing its marker');
  const comments = await github.paginate(github.rest.issues.listComments, {
    ...context.repo, issue_number: Number(number), per_page: 100,
  });
  const previous = comments.find(c => c.user?.login === 'github-actions[bot]' && c.body?.includes(marker));
  if (previous) {
    await github.rest.issues.updateComment({ ...context.repo, comment_id: previous.id, body });
  } else {
    await github.rest.issues.createComment({ ...context.repo, issue_number: Number(number), body });
  }
}

module.exports = { resolvePullRequest, comment };
