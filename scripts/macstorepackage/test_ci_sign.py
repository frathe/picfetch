"""CI credential lifetime tests: no real Keychain or private keys are used."""
import base64
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import ci_sign


class CICredentials(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.app = self.root / 'qualified/PicFetch.app'
        self.app.mkdir(parents=True)
        self.env = {'GITHUB_ACTIONS': 'true', 'RUNNER_ENVIRONMENT': 'github-hosted',
                    'RUNNER_OS': 'macOS', 'GITHUB_REF': 'refs/heads/main',
                    'GITHUB_REPOSITORY': 'frathe/picfetch', 'GITHUB_SHA': 'a' * 40,
                    'RUNNER_TEMP': str(self.root), 'APPLE_STORE_APP': str(self.app),
                    'APPLE_STORE_SIGNED_OUTPUT_DIR': str(self.root / 'signed'),
                    'APPLE_STORE_TEAM_ID': 'ABCDEFGHIJ', 'APPLE_STORE_APP_IDENTITY': 'A' * 40,
                    'APPLE_STORE_INSTALLER_IDENTITY': 'B' * 40}
        for name in ('APP_P12', 'INSTALLER_P12', 'PROFILE', 'WORKER_PROFILE'):
            self.env['APPLE_STORE_' + name + '_BASE64'] = base64.b64encode(name.encode()).decode()
        self.env['APPLE_STORE_APP_P12_PASSWORD'] = 'app-secret'
        self.env['APPLE_STORE_INSTALLER_P12_PASSWORD'] = 'installer-secret'
        self.manifest = {'source': self.env['GITHUB_SHA'], 'dirty': False,
                         'source_diff_sha256': hashlib.sha256(b'').hexdigest(), 'untracked_sources': {}}
        self.write_manifest()
        self.calls = []
        self.fail_operation = None

    def write_manifest(self):
        (self.app.parent / 'manifest.json').write_text(json.dumps(self.manifest))

    def security(self, *args):
        self.calls.append(args)
        if args[0] == self.fail_operation:
            raise RuntimeError('synthetic keychain failure')
        if args == ('list-keychains', '-d', 'user'):
            return '"/Users/runner/Library/Keychains/login.keychain-db"\n'
        if args[0] == 'create-keychain':
            Path(args[-1]).touch()
        return ''

    def assert_clean(self):
        self.assertFalse((self.root / 'picfetch-apple-signing').exists())
        self.assertIn(('list-keychains', '-d', 'user', '-s',
                       '/Users/runner/Library/Keychains/login.keychain-db'), self.calls)
        self.assertTrue(any(call[0] == 'delete-keychain' for call in self.calls))

    def test_signs_with_scoped_keys_and_no_secret_environment(self):
        def package(*args, **kwargs):
            child = kwargs['env']
            self.assertFalse(any('BASE64' in key or 'PASSWORD' in key for key in child))
            self.assertEqual(child['APPLE_STORE_TESTFLIGHT'], '1')
            self.assertEqual(Path(child['APPLE_STORE_PROFILE']).read_bytes(), b'PROFILE')
            self.assertEqual(Path(child['APPLE_STORE_WORKER_PROFILE']).read_bytes(), b'WORKER_PROFILE')
            self.assertEqual((self.root / 'picfetch-apple-signing').stat().st_mode & 0o777, 0o700)
            self.assertFalse(list((self.root / 'picfetch-apple-signing').glob('*.p12')))
        with patch.object(ci_sign, 'security', self.security), patch.object(ci_sign.subprocess, 'run', package):
            ci_sign.sign(self.env)
        imports = [call for call in self.calls if call[0] == 'import']
        self.assertEqual(len(imports), 2)
        for call in imports:
            self.assertNotIn('-A', call)
            self.assertIn('/usr/bin/codesign', call)
            self.assertIn('/usr/bin/productbuild', call)
        self.assert_clean()

    def test_rejects_missing_or_bad_inputs_before_keychain(self):
        for key, value in [('APPLE_STORE_APP_P12_PASSWORD', ''), ('APPLE_STORE_PROFILE_BASE64', '!'),
                           ('APPLE_STORE_WORKER_PROFILE_BASE64', ''), ('GITHUB_REF', 'refs/heads/topic'),
                           ('RUNNER_ENVIRONMENT', 'self-hosted'), ('RUNNER_OS', 'Linux'),
                           ('GITHUB_REPOSITORY', 'someone/fork'), ('APPLE_STORE_TEAM_ID', 'bad'),
                           ('APPLE_STORE_APP_IDENTITY', 'name'), ('GITHUB_SHA', '')]:
            with self.subTest(key=key), patch.object(ci_sign, 'security', self.security):
                with self.assertRaises(ValueError):
                    ci_sign.sign(self.env | {key: value})
                self.assertEqual(self.calls, [])

    def test_rejects_unqualified_source(self):
        for key, value in [('source', 'b' * 40), ('dirty', True),
                           ('source_diff_sha256', 'wrong'), ('untracked_sources', {'extra': 'hash'})]:
            with self.subTest(key=key), patch.object(ci_sign, 'security', self.security):
                original = self.manifest.copy()
                self.manifest[key] = value
                self.write_manifest()
                with self.assertRaises(ValueError):
                    ci_sign.sign(self.env)
                self.assertEqual(self.calls, [])
                self.manifest = original

    def test_cleans_partial_import_and_signing_failure_or_cancellation(self):
        for failure in ('import', 'set-key-partition-list', 'package', 'cancel', 'sigterm'):
            with self.subTest(failure=failure):
                self.calls = []
                self.fail_operation = failure
                error = KeyboardInterrupt() if failure == 'cancel' else RuntimeError('failed package')
                if failure == 'sigterm':
                    error = lambda *_args, **_kwargs: ci_sign.cancelled(15, None)
                with patch.object(ci_sign, 'security', self.security), \
                        patch.object(ci_sign.subprocess, 'run', side_effect=error):
                    with self.assertRaises((RuntimeError, KeyboardInterrupt, SystemExit)):
                        ci_sign.sign(self.env)
                self.assert_clean()

    def test_cleanup_retries_failed_search_list_restore(self):
        work = self.root / 'picfetch-apple-signing'
        work.mkdir()
        (work / 'search-list.json').write_text(json.dumps(['/original.keychain-db']))
        (work / 'signing.keychain-db').touch()
        self.fail_operation = 'list-keychains'
        with patch.object(ci_sign, 'security', self.security):
            with self.assertRaises(RuntimeError):
                ci_sign.cleanup(self.env)
            self.assertTrue((work / 'search-list.json').exists())
            self.assertTrue(any(call[0] == 'delete-keychain' for call in self.calls))
            self.fail_operation = None
            ci_sign.cleanup(self.env)
        self.assertFalse(work.exists())

    def test_keychain_failure_does_not_disclose_command_or_output(self):
        error = subprocess.CalledProcessError(1, ['security', 'import', 'private-secret'],
                                             output=b'private-secret', stderr=b'private-secret')
        with patch.object(ci_sign.subprocess, 'run', side_effect=error):
            with self.assertRaises(RuntimeError) as caught:
                ci_sign.security('import', 'private-secret')
        self.assertNotIn('private-secret', str(caught.exception))
        self.assertTrue(caught.exception.__suppress_context__)


class CIWorkflow(unittest.TestCase):
    def test_candidate_actions_use_immutable_commits(self):
        workflow = (Path(__file__).resolve().parents[2] / '.github/workflows/apple-store.yml').read_text()
        actions = re.findall(r'^\s*(?:-\s+)?uses:\s*(\S+)', workflow, re.MULTILINE)
        self.assertTrue(actions)
        for action in actions:
            if action == './.github/workflows/ci.yml':
                continue
            with self.subTest(action=action):
                self.assertRegex(action, r'^actions/[a-z-]+@[0-9a-f]{40}$')

    def test_signing_requires_main_ci_build_and_environment_approval(self):
        workflow = (Path(__file__).resolve().parents[2] / '.github/workflows/apple-store.yml').read_text()
        self.assertEqual(workflow.split('on:\n', 1)[1].split('\npermissions:', 1)[0].strip(),
                         'workflow_dispatch:')
        self.assertIn("github.repository == 'frathe/picfetch' && github.ref == 'refs/heads/main'", workflow)
        self.assertIn('uses: ./.github/workflows/ci.yml', workflow)
        build, signing = workflow.split('  build:\n', 1)[1].split('  sign:\n', 1)
        self.assertIn('needs: test', build)
        self.assertNotIn('secrets.', build)
        self.assertIn('needs: build', signing)
        self.assertIn('artifact-ids: ${{ needs.build.outputs.artifact-id }}', signing)
        self.assertIn('artifact-id: ${{ steps.qualified.outputs.artifact-id }}', workflow)
        self.assertIn('environment: apple-store-signing', signing)
        self.assertIn('if: ${{ always() }}', signing)
        self.assertIn('ci_sign.py --cleanup', signing)
        self.assertEqual(workflow.count('ref: ${{ github.sha }}'), 2)
        self.assertEqual(workflow.count('persist-credentials: false'), 2)
        self.assertNotIn('secrets: inherit', workflow)
        self.assertNotIn('contents: write', workflow)
        for name in ci_sign.SECRET_NAMES:
            self.assertIn('${{ secrets.' + name + ' }}', signing)


if __name__ == '__main__':
    unittest.main()
