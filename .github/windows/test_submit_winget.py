import base64
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock
from urllib.error import HTTPError

spec = importlib.util.spec_from_file_location('submit_winget', Path(__file__).with_name('submit-winget.py'))
winget = importlib.util.module_from_spec(spec)
spec.loader.exec_module(winget)

SOURCE = 'repos/xgo-dev/llgo'
UPSTREAM = 'repos/microsoft/winget-pkgs'
FORK = 'repos/xgo-dev/winget-pkgs'


class FakeGitHub(winget.GitHub):
    """Model interrupted writes as well as a fresh branch and upstream PR."""

    def __init__(self):
        super().__init__('test-only-token')
        self.calls = []
        self.own_prs = []
        self.other_prs = []
        self.other_files = []
        self.version_exists = False
        self.package_exists = True
        self.ref_exists = False
        self.branch_files = {}
        self.fork = {'fork': True, 'parent': {'full_name': winget.UPSTREAM}}
        self.fail_pull = False
        self.release = {'id': 1, 'tag_name': 'v1.2.3', 'draft': False,
                        'prerelease': False, 'published_at': '2026-10-22T00:00:00Z'}
        self.assets = []
        checksums = ''
        for profile in winget.manifests.PROFILES:
            for arch in winget.manifests.ARCHITECTURES:
                name = f'llgo1.2.3.windows-{arch}-{profile}.zip'
                data = name.encode()
                digest = hashlib.sha256(data).hexdigest()
                checksums += f'{digest}  {name}\n'
                self.assets.append(self.asset(name, data))
        self.checksums = checksums.encode()
        self.assets.append(self.asset('llgo1.2.3.checksums.txt', self.checksums))

    @staticmethod
    def asset(name, data):
        return {'name': name, 'state': 'uploaded', 'size': len(data),
                'browser_download_url': f'https://github.com/xgo-dev/llgo/releases/download/v1.2.3/{name}',
                'digest': 'sha256:' + hashlib.sha256(data).hexdigest()}

    @property
    def writes(self):
        return [call for call in self.calls if call[1] != 'GET']

    def request(self, path, method='GET', data=None, **params):
        self.calls.append((path, method, data, params))
        if method == 'POST':
            if path == f'{FORK}/git/trees':
                self.branch_files = {item['path']: item['content'] for item in data['tree']}
                return {'sha': 'new-tree'}
            if path == f'{FORK}/git/commits':
                return {'sha': 'new-commit'}
            if path == f'{FORK}/git/refs':
                self.ref_exists = True
                return {'object': {'sha': 'new-commit'}}
            if path == f'{UPSTREAM}/pulls':
                if self.fail_pull:
                    raise HTTPError(path, 503, 'temporary failure', {}, io.BytesIO())
                pull = {'state': 'open', 'html_url': 'https://github.com/microsoft/winget-pkgs/pull/123'}
                self.own_prs.append(pull)
                return pull
            raise AssertionError(f'Unexpected write: {path}')
        if path == f'{SOURCE}/releases/tags/v1.2.3':
            return self.release
        if path == f'{SOURCE}/releases/1/assets':
            return self.assets
        if path == UPSTREAM:
            return {'default_branch': 'master'}
        if path == FORK:
            return self.fork
        if path == f'{UPSTREAM}/pulls':
            return self.own_prs
        if path == 'search/issues':
            return {'total_count': len(self.other_prs), 'incomplete_results': False, 'items': self.other_prs}
        if path == f'{UPSTREAM}/pulls/99/files':
            return self.other_files
        if path.startswith(f'{UPSTREAM}/contents/'):
            exists = self.version_exists if path.endswith('/1.2.3') else self.package_exists
            if exists:
                return [{'name': 'existing'}]
        elif path.startswith(f'{FORK}/git/ref/heads/'):
            if self.ref_exists:
                return {'object': {'sha': 'new-commit'}}
        elif path == f'{UPSTREAM}/git/ref/heads/master':
            return {'object': {'sha': 'upstream-head'}}
        elif path == f'{UPSTREAM}/git/commits/upstream-head':
            return {'tree': {'sha': 'upstream-tree'}}
        elif path.startswith(f'{FORK}/compare/upstream-head...'):
            return {'files': [{'filename': name, 'status': 'added'} for name in self.branch_files]}
        elif path.startswith(f'{FORK}/contents/'):
            content = self.branch_files[path.removeprefix(f'{FORK}/contents/')]
            return {'content': base64.b64encode(content.encode()).decode()}
        else:
            raise AssertionError(f'Unexpected read: {path} {params}')
        raise HTTPError(path, 404, 'not found', {}, io.BytesIO())


class SubmissionTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.output = Path(temporary.name) / 'manifests'
        self.api = FakeGitHub()
        download = mock.patch.object(winget, 'public_bytes', side_effect=lambda url: self.api.checksums)
        self.download = download.start()
        self.addCleanup(download.stop)

    def submit(self, profile='msvc', dry_run=False):
        return winget.submit(self.api, 'v1.2.3', profile, self.output, dry_run)

    def test_new_submission_is_one_package_on_an_org_fork_branch(self):
        for profile, label in (('msvc', 'MSVC'), ('mingw', 'MinGW')):
            with self.subTest(profile=profile):
                self.api = FakeGitHub()
                self.api.package_exists = profile == 'msvc'
                self.assertTrue(self.submit(profile).endswith('/pull/123'))
                self.assertEqual([call[0] for call in self.api.writes], [
                    f'{FORK}/git/trees', f'{FORK}/git/commits', f'{FORK}/git/refs', f'{UPSTREAM}/pulls'])
                tree, commit, ref, pull = [call[2] for call in self.api.writes]
                self.assertEqual(tree['base_tree'], 'upstream-tree')
                self.assertEqual(commit['parents'], ['upstream-head'])
                self.assertEqual(len(tree['tree']), 3)
                expected = {path.relative_to(self.output).as_posix(): path.read_text() for path in
                            (self.output / f'manifests/x/XGo/LLGo/{label}/1.2.3').glob('*.yaml')}
                self.assertEqual(self.api.branch_files, expected)
                self.assertEqual(ref['ref'], f'refs/heads/llgo/{profile}/1.2.3')
                self.assertEqual(pull['head'], f'xgo-dev:llgo/{profile}/1.2.3')
                self.assertEqual(pull['base'], 'master')
                self.assertTrue(pull['title'].startswith('Update:' if profile == 'msvc' else 'New package:'))
                self.assertIn('- [ ] Tested with `winget install', pull['body'])
                self.assertNotIn('test-only-token', json.dumps(self.api.writes))

    def test_repeat_submission_leaves_review_changes_and_pr_untouched(self):
        self.submit()
        self.api.branch_files['review-edit'] = 'maintainer changes'
        self.api.calls.clear()
        self.assertIn('/pull/123', self.submit())
        self.assertEqual(self.api.writes, [])

    def test_partial_failure_reuses_branch_without_writing_it_again(self):
        self.api.fail_pull = True
        with self.assertRaises(HTTPError):
            self.submit()
        self.api.fail_pull = False
        self.api.calls.clear()
        self.assertTrue(self.submit().endswith('/pull/123'))
        self.assertEqual([call[0] for call in self.api.writes], [f'{UPSTREAM}/pulls'])
        self.assertTrue(any('/compare/upstream-head...' in call[0] for call in self.api.calls))

    def test_partial_branch_with_other_changes_is_not_published(self):
        for change in ('extra-file', 'edited-file'):
            with self.subTest(change=change):
                self.api = FakeGitHub()
                self.api.fail_pull = True
                with self.assertRaises(HTTPError):
                    self.submit()
                self.api.fail_pull = False
                if change == 'extra-file':
                    self.api.branch_files['README.md'] = 'unrelated'
                else:
                    self.api.branch_files[next(iter(self.api.branch_files))] += '# edited\n'
                self.api.calls.clear()
                with self.assertRaisesRegex(ValueError, 'inspect it manually'):
                    self.submit()
                self.assertEqual(self.api.writes, [])

    def test_dry_run_and_already_published_version_do_not_write(self):
        self.assertIn('Dry run:', self.submit(dry_run=True))
        self.assertEqual(self.api.writes, [])
        self.assertEqual(len(list(self.output.rglob('*.yaml'))), 6)
        self.api.version_exists = True
        self.assertIn('already published', self.submit())
        self.assertEqual(self.api.writes, [])

    def test_another_forks_matching_version_is_reused(self):
        self.api.other_prs = [{'number': 99, 'html_url': 'https://github.com/microsoft/winget-pkgs/pull/99'}]
        self.api.other_files = [{'filename': 'manifests/x/XGo/LLGo/MSVC/1.2.3/XGo.LLGo.MSVC.yaml'}]
        self.assertIn('/pull/99', self.submit())
        self.assertEqual(self.api.writes, [])

    def test_title_match_for_another_version_does_not_block_submission(self):
        self.api.other_prs = [{'number': 99, 'html_url': 'https://github.com/microsoft/winget-pkgs/pull/99'}]
        self.api.other_files = [{'filename': 'manifests/x/XGo/LLGo/MSVC/1.2.30/XGo.LLGo.MSVC.yaml'}]
        self.assertTrue(self.submit().endswith('/pull/123'))

    def test_closed_pr_is_not_reopened_or_replaced(self):
        self.api.own_prs = [{'state': 'closed', 'html_url': 'https://github.com/microsoft/winget-pkgs/pull/123'}]
        with self.assertRaisesRegex(ValueError, 'closed PR'):
            self.submit()
        self.assertEqual(self.api.writes, [])

    def test_incomplete_duplicate_search_cannot_create_a_pr(self):
        request = self.api.request
        def incomplete(path, *args, **kwargs):
            if path == 'search/issues':
                return {'incomplete_results': True, 'total_count': 0, 'items': []}
            return request(path, *args, **kwargs)
        with mock.patch.object(self.api, 'request', side_effect=incomplete):
            with self.assertRaisesRegex(ValueError, 'search was incomplete'):
                self.submit()
        self.assertEqual(self.api.writes, [])

    def test_invalid_fork_never_receives_writes(self):
        self.api.fork['parent']['full_name'] = 'someone/other'
        with self.assertRaisesRegex(ValueError, 'must be a fork'):
            self.submit()
        self.assertEqual(self.api.writes, [])

    def test_non_stable_tags_are_rejected_without_network_calls(self):
        for tag in ('main', '1.2.3', 'v1.2.3-rc.1', 'v1.2.3+build', '../other', 'v1.2.3\n'):
            with self.subTest(tag=tag), self.assertRaisesRegex(ValueError, 'stable'):
                winget.published_installers(self.api, tag)
        self.assertEqual(self.api.calls, [])

    def test_draft_prerelease_and_unpublished_release_are_rejected(self):
        for field, value in (('draft', True), ('prerelease', True), ('published_at', None), ('tag_name', 'v1.2.4')):
            with self.subTest(field=field):
                self.api = FakeGitHub()
                self.api.release[field] = value
                with self.assertRaisesRegex(ValueError, 'published and stable'):
                    self.submit()
                self.assertEqual(self.api.writes, [])

    def test_missing_or_mismatched_asset_from_either_profile_blocks_all_writes(self):
        cases = [('missing', None), ('digest', 'sha256:' + '0' * 64), ('state', 'open'), ('size', 0),
                 ('browser_download_url', 'https://example.com/asset.zip')]
        for field, value in cases:
            with self.subTest(field=field):
                self.api = FakeGitHub()
                # Deliberately damage MinGW while submitting MSVC.
                if field == 'missing':
                    del self.api.assets[-2]
                else:
                    self.api.assets[-2][field] = value
                with self.assertRaisesRegex(ValueError, 'asset|checksum'):
                    self.submit()
                self.assertEqual(self.api.writes, [])
                self.assertFalse(self.output.exists())

    def test_checksum_download_is_verified_before_parsing(self):
        self.api.checksums += b'corrupt'
        with self.assertRaisesRegex(ValueError, 'checksum file'):
            self.submit()
        self.assertEqual(self.api.writes, [])

    def test_duplicate_checksum_and_asset_names_are_rejected(self):
        self.api.checksums += self.api.checksums.splitlines(keepends=True)[0]
        self.api.assets[-1] = self.api.asset('llgo1.2.3.checksums.txt', self.api.checksums)
        with self.assertRaisesRegex(ValueError, 'duplicate published checksum'):
            self.submit()
        self.api = FakeGitHub()
        self.api.assets.append(self.api.assets[0])
        with self.assertRaisesRegex(ValueError, 'Duplicate release asset'):
            self.submit()
        self.assertEqual(self.api.writes, [])


class TransportTests(unittest.TestCase):
    def test_missing_write_secret_fails_before_any_api_call(self):
        with mock.patch.dict(winget.os.environ, {'GH_TOKEN': 'read-only-token'}, clear=True):
            with mock.patch.object(winget.sys, 'argv', ['submit-winget.py', '--tag', 'v1.2.3', '--profile', 'msvc']):
                with mock.patch.object(winget, 'urlopen') as open_url, mock.patch('sys.stderr', new_callable=io.StringIO) as stderr:
                    with self.assertRaises(SystemExit) as result:
                        winget.main()
                    self.assertEqual(result.exception.code, 2)
                    self.assertIn('WINGET_PR_TOKEN', stderr.getvalue())
                    open_url.assert_not_called()

    def test_dry_run_cannot_write_even_with_a_token(self):
        api = winget.GitHub('test-only-token', dry_run=True)
        with mock.patch.object(winget, 'urlopen') as open_url, self.assertRaises(ValueError):
            api.request(f'{FORK}/git/refs', method='POST', data={})
        open_url.assert_not_called()

    def test_only_reads_retry_transient_http_errors(self):
        for method, attempts in (('GET', 3), ('POST', 1)):
            with self.subTest(method=method), mock.patch.object(winget, 'time'):
                with mock.patch.object(winget, 'urlopen', side_effect=HTTPError('url', 503, 'failure', {}, io.BytesIO())) as open_url:
                    with self.assertRaises(HTTPError):
                        winget.GitHub('test-only-token').request('repos/xgo-dev/winget-pkgs', method=method)
                    self.assertEqual(open_url.call_count, attempts)

    def test_public_asset_download_does_not_forward_credentials(self):
        with mock.patch.object(winget, 'urlopen') as open_url:
            open_url.return_value.__enter__.return_value.read.return_value = b'checksums'
            self.assertEqual(winget.public_bytes('https://github.com/xgo-dev/llgo/releases/download/v1.2.3/checksums'), b'checksums')
            self.assertIsInstance(open_url.call_args.args[0], str)
            self.assertEqual(open_url.call_args.kwargs, {'timeout': 60})


if __name__ == '__main__':
    unittest.main()
