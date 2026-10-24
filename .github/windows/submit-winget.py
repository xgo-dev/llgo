#!/usr/bin/env python3
"""Submit a stable LLGo release through xgo-dev/winget-pkgs, one PR per ABI."""

import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import time
from urllib.error import HTTPError
from urllib.parse import quote, urlencode
from urllib.request import Request, urlopen

spec = importlib.util.spec_from_file_location('prepare_winget', Path(__file__).with_name('prepare-winget.py'))
manifests = importlib.util.module_from_spec(spec)
spec.loader.exec_module(manifests)

SOURCE = 'xgo-dev/llgo'
UPSTREAM = 'microsoft/winget-pkgs'
FORK = 'xgo-dev/winget-pkgs'


class GitHub:
    def __init__(self, token='', dry_run=False):
        self.token = token
        self.dry_run = dry_run

    def request(self, path, method='GET', data=None, **params):
        if method != 'GET' and (self.dry_run or not self.token):
            raise ValueError('Writes require WINGET_PR_TOKEN and a non-dry run')
        query = '?' + urlencode(params) if params else ''
        headers = {'Accept': 'application/vnd.github+json', 'X-GitHub-Api-Version': '2022-11-28'}
        if self.token:
            headers['Authorization'] = f'Bearer {self.token}'
        payload = json.dumps(data).encode() if data is not None else None
        if payload is not None:
            headers['Content-Type'] = 'application/json'
        request = Request(f'https://api.github.com/{path}{query}', data=payload, headers=headers, method=method)
        for attempt in range(3):
            try:
                with urlopen(request, timeout=60) as response:
                    return json.load(response)
            except HTTPError as error:
                # Writes are recovered by the next run, never blindly replayed.
                if method != 'GET' or error.code not in (429, 500, 502, 503, 504) or attempt == 2:
                    raise
                error.close()
                time.sleep(2 ** attempt)
        raise RuntimeError('GitHub API retries exhausted')

    def optional(self, path, **params):
        try:
            return self.request(path, **params)
        except HTTPError as error:
            if error.code != 404:
                raise
            error.close()
            return None

    def pages(self, path, **params):
        page = 1
        while True:
            items = self.request(path, per_page=100, page=page, **params)
            yield from items
            if len(items) < 100:
                return
            page += 1


def public_bytes(url):
    # Keep credentials out of public asset downloads and their redirects.
    with urlopen(url, timeout=60) as response:
        return response.read()


def published_installers(api, tag):
    if not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+', tag):
        raise ValueError('WinGet submission requires a stable vMAJOR.MINOR.PATCH tag')
    release = api.request(f'repos/{SOURCE}/releases/tags/{tag}')
    if release['tag_name'] != tag or release['draft'] or release['prerelease'] or not release['published_at']:
        raise ValueError('Release must be published and stable')
    assets = list(api.pages(f'repos/{SOURCE}/releases/{release["id"]}/assets'))
    by_name = {asset['name']: asset for asset in assets}
    if len(by_name) != len(assets):
        raise ValueError('Duplicate release asset names')
    prefix = f'https://github.com/{SOURCE}/releases/download/{tag}/'

    def asset(name):
        value = by_name.get(name, {})
        if (value.get('state') != 'uploaded' or value.get('size', 0) <= 0 or
                value.get('browser_download_url') != prefix + name or
                not re.fullmatch(r'sha256:[0-9a-f]{64}', value.get('digest') or '')):
            raise ValueError(f'Missing or invalid published asset: {name}')
        return value

    version = tag[1:]
    checksums = asset(f'llgo{version}.checksums.txt')
    content = public_bytes(checksums['browser_download_url'])
    if 'sha256:' + hashlib.sha256(content).hexdigest() != checksums['digest']:
        raise ValueError('Published checksum file does not match its GitHub digest')
    hashes = {}
    for line in content.decode('utf-8').splitlines():
        match = re.fullmatch(r'([0-9a-fA-F]{64})  (\S+)', line)
        if not match or match[2] in hashes:
            raise ValueError('Malformed or duplicate published checksum entry')
        hashes[match[2]] = match[1].lower()
    installers = {}
    # Check all six profile/architecture ZIPs before either PR can be submitted.
    for profile in manifests.PROFILES:
        for arch in manifests.ARCHITECTURES:
            name = f'llgo{version}.windows-{arch}-{profile}.zip'
            value = asset(name)
            if value['digest'] != 'sha256:' + hashes.get(name, ''):
                raise ValueError(f'Published installer checksum mismatch: {name}')
            installers[profile, arch] = (value['browser_download_url'], hashes[name])
    return installers


def existing_pull(api, branch, directory, identifier, version):
    own = list(api.pages(f'repos/{UPSTREAM}/pulls', state='all', head=f'xgo-dev:{branch}'))
    for pull in own:
        if pull['state'] == 'open':
            return pull['html_url']
    if own:
        raise ValueError(f'The submission branch already has a closed PR: {own[0]["html_url"]}')
    result = api.request('search/issues', q=f'repo:{UPSTREAM} is:pr is:open "{identifier}" "{version}" in:title', per_page=100)
    if result.get('incomplete_results') or result['total_count'] > 100:
        raise ValueError('Duplicate PR search was incomplete; retry before submitting')
    for pull in result['items']:
        files = api.pages(f'repos/{UPSTREAM}/pulls/{pull["number"]}/files')
        if any(item['filename'].startswith(directory + '/') for item in files):
            return pull['html_url']
    return None


def check_branch(api, branch, base, files):
    comparison = api.request(f'repos/{FORK}/compare/{base}...{quote(branch, safe="")}')
    changes = comparison.get('files', [])
    # Compare caps the file list at 300; larger diffs also fail this exact set check.
    if {item['filename'] for item in changes} != set(files) or any(item['status'] != 'added' for item in changes):
        raise ValueError(f'Existing branch {branch} contains unexpected changes; inspect it manually')
    for path, content in files.items():
        blob = api.request(f'repos/{FORK}/contents/{path}', ref=branch)
        if base64.b64decode(blob['content']).decode('utf-8') != content:
            raise ValueError(f'Existing branch {branch} differs from the release manifests; inspect it manually')


def submit(api, tag, profile, output, dry_run=False):
    installers = published_installers(api, tag)
    version = tag[1:]
    manifests.write_manifests(output, version, installers)
    label = manifests.PROFILES[profile]
    identifier = f'XGo.LLGo.{label}'
    package = f'manifests/x/XGo/LLGo/{label}'
    directory = f'{package}/{version}'
    branch = f'llgo/{profile}/{version}'
    upstream = api.request(f'repos/{UPSTREAM}')
    base = upstream['default_branch']
    if api.optional(f'repos/{UPSTREAM}/contents/{directory}', ref=base) is not None:
        return f'{identifier} {version} is already published'
    if url := existing_pull(api, branch, directory, identifier, version):
        return f'Existing submission: {url}'
    is_update = api.optional(f'repos/{UPSTREAM}/contents/{package}', ref=base) is not None
    title = f'Update: {identifier} to {version}' if is_update else f'New package: {identifier} version {version}'
    if dry_run:
        return f'Dry run: {title}; {FORK}:{branch} -> {UPSTREAM}:{base}'
    fork = api.request(f'repos/{FORK}')
    if not fork['fork'] or fork.get('parent', {}).get('full_name') != UPSTREAM:
        raise ValueError(f'{FORK} must be a fork of {UPSTREAM}')
    files = {f'{directory}/{name}': (output / directory / name).read_text(encoding='utf-8') for name in
             (f'{identifier}.yaml', f'{identifier}.installer.yaml', f'{identifier}.locale.en-US.yaml')}
    ref = api.optional(f'repos/{FORK}/git/ref/heads/{branch}')
    parent = api.request(f'repos/{UPSTREAM}/git/ref/heads/{quote(base, safe="")}')['object']['sha']
    if ref is None:
        commit = api.request(f'repos/{UPSTREAM}/git/commits/{parent}')
        tree = api.request(f'repos/{FORK}/git/trees', method='POST', data={
            'base_tree': commit['tree']['sha'],
            'tree': [{'path': path, 'mode': '100644', 'type': 'blob', 'content': content}
                     for path, content in files.items()],
        })
        commit = api.request(f'repos/{FORK}/git/commits', method='POST', data={
            'message': title, 'tree': tree['sha'], 'parents': [parent],
        })
        api.request(f'repos/{FORK}/git/refs', method='POST', data={
            'ref': f'refs/heads/{branch}', 'sha': commit['sha'],
        })
    else:
        # Recover interrupted submissions without overwriting review edits.
        check_branch(api, branch, parent, files)
    body = f'''## 📖 Description

{identifier} {version} for x86, x64 and ARM64, from [LLGo {tag}](https://github.com/{SOURCE}/releases/tag/{tag}).

The release pipeline tested all six Windows ZIPs. Installer hashes match both the published checksum file and GitHub asset digests. MSVC retains the portable alias; MinGW adds the archive binaries to PATH. WinGet installation validation runs on this PR.

## ✅ Checklist

- [ ] Signed the [Contributor License Agreement](https://cla.opensource.microsoft.com)
- [x] Linked to an issue: xgo-dev/llgo#2325

## 📦 Manifest Checklist

- [x] Checked for another open PR for this package version
- [x] This PR modifies one package version only
- [ ] Validated with `winget validate --manifest <path>`
- [ ] Tested with `winget install --manifest <path>`
- [x] Manifest conforms to schema {manifests.SCHEMA}
'''
    pull = api.request(f'repos/{UPSTREAM}/pulls', method='POST', data={
        'title': title, 'head': f'xgo-dev:{branch}',
        'base': base, 'body': body,
    })
    return pull['html_url']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    parser.add_argument('--profile', choices=manifests.PROFILES, required=True)
    parser.add_argument('--output', type=Path, default=Path('.winget'))
    parser.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()
    token = os.environ.get('WINGET_PR_TOKEN', '')
    if not args.dry_run and not token:
        parser.error('Set the WINGET_PR_TOKEN Actions secret; GITHUB_TOKEN cannot submit cross-repository PRs')
    api = GitHub(token or os.environ.get('GH_TOKEN', ''), dry_run=args.dry_run)
    result = submit(api, args.tag, args.profile, args.output, args.dry_run)
    print(result)
    if summary := os.environ.get('GITHUB_STEP_SUMMARY'):
        with open(summary, 'a', encoding='utf-8') as stream:
            stream.write(f'- {result}\n')


if __name__ == '__main__':
    try:
        main()
    except (HTTPError, ValueError) as error:
        print(f'WinGet submission failed: {error}', file=sys.stderr)
        sys.exit(1)
