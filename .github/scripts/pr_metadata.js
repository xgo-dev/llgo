'use strict';

const REQUIRED_HEADINGS = ['Summary', 'Validation', 'Fork CI'];
const CONFIRMATION =
  '- [x] I created a test PR in my fork and its relevant CI passed.';

function stripHtmlComments(value) {
  return value.replace(/<!--[\s\S]*?-->/g, '').trim();
}

function parseTemplate(body) {
  const text = typeof body === 'string' ? body : '';
  const headingPattern = /^##[ \t]+(Summary|Validation|Fork CI)[ \t]*$/gm;
  const headings = [...text.matchAll(headingPattern)];
  const names = headings.map((match) => match[1]);

  if (
    names.length !== REQUIRED_HEADINGS.length ||
    !names.every((name, index) => name === REQUIRED_HEADINGS[index])
  ) {
    throw new Error(
      'Keep the required headings exactly once and in this order: Summary, Validation, Fork CI.',
    );
  }

  const sections = {};
  headings.forEach((heading, index) => {
    const start = heading.index + heading[0].length;
    const end = index + 1 < headings.length ? headings[index + 1].index : text.length;
    sections[heading[1]] = text.slice(start, end);
  });

  for (const name of ['Summary', 'Validation']) {
    const visible = stripHtmlComments(sections[name]);
    if (!/[\p{L}\p{N}]/u.test(visible)) {
      throw new Error(`Fill in the ${name} section instead of leaving only the template comment.`);
    }
  }

  const forkLines = stripHtmlComments(sections['Fork CI'])
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);

  if (forkLines.length !== 2 || forkLines[0] !== CONFIRMATION) {
    throw new Error('Check the fork-CI confirmation and keep the required Fork CI fields.');
  }

  const link = forkLines[1].match(
    /^- Fork test PR: (https:\/\/github\.com\/([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+)\/pull\/([1-9][0-9]*))\/?$/i,
  );
  if (!link) {
    throw new Error('Provide one valid GitHub fork pull request URL after "Fork test PR:".');
  }

  return {
    url: link[1],
    owner: link[2],
    repo: link[3],
    number: Number(link[4]),
  };
}

function sameRepository(left, right) {
  return Boolean(left && right && left.toLowerCase() === right.toLowerCase());
}

async function validatePullRequest({body, upstreamRepository, source, github}) {
  const evidence = parseTemplate(body);
  const linkedRepository = `${evidence.owner}/${evidence.repo}`;

  if (!sameRepository(linkedRepository, source.head.repo?.full_name)) {
    throw new Error(
      `Fork test PR repository ${linkedRepository} does not match contribution repository ${source.head.repo?.full_name}.`,
    );
  }

  const {data: fork} = await github.rest.repos.get({
    owner: evidence.owner,
    repo: evidence.repo,
  });
  if (!fork.fork || !sameRepository(fork.parent?.full_name, upstreamRepository)) {
    throw new Error(`${evidence.url} is not a pull request in a fork of ${upstreamRepository}.`);
  }

  const {data: testPR} = await github.rest.pulls.get({
    owner: evidence.owner,
    repo: evidence.repo,
    pull_number: evidence.number,
  });
  if (!sameRepository(testPR.head.repo?.full_name, source.head.repo?.full_name)) {
    throw new Error(
      `Fork test PR head repository ${testPR.head.repo?.full_name} does not match ${source.head.repo?.full_name}.`,
    );
  }
  if (testPR.head.ref !== source.head.ref) {
    throw new Error(
      `Fork test PR branch ${testPR.head.ref} does not match contribution branch ${source.head.ref}.`,
    );
  }
  if (testPR.head.sha !== source.head.sha) {
    throw new Error(
      `Fork test PR checks ${testPR.head.sha}, but this contribution uses ${source.head.sha}.`,
    );
  }

  return evidence;
}

module.exports = {parseTemplate, validatePullRequest};
