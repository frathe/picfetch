"""Sign a CI candidate with temporary credentials on a GitHub-hosted Mac.

The workflow's protected environment supplies approval. This runner guard is
additional validation, not a replacement for GitHub environment protection.
"""
import argparse
import base64
import binascii
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shlex
import shutil
import signal
import subprocess
import sys


SECRET_NAMES = tuple('APPLE_STORE_' + name for name in (
    'APP_P12_BASE64', 'APP_P12_PASSWORD', 'INSTALLER_P12_BASE64',
    'INSTALLER_P12_PASSWORD', 'PROFILE_BASE64', 'WORKER_PROFILE_BASE64'))


def public_environment(env):
    return {key: value for key, value in env.items() if key not in SECRET_NAMES}


def security(*args):
    # Do not use package.run: it prints arguments, including Keychain passwords.
    try:
        result = subprocess.run(['/usr/bin/security', *map(str, args)], check=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                env=public_environment(os.environ))
    except (OSError, subprocess.CalledProcessError):
        raise RuntimeError('Apple Keychain operation failed: ' + args[0]) from None
    return result.stdout.decode()


def work_directory(env):
    if (env.get('GITHUB_ACTIONS') != 'true' or env.get('RUNNER_ENVIRONMENT') != 'github-hosted' or
            env.get('RUNNER_OS') != 'macOS' or env.get('GITHUB_REF') != 'refs/heads/main' or
            env.get('GITHUB_REPOSITORY') != 'frathe/picfetch'):
        raise ValueError('CI signing requires the main branch on a GitHub-hosted macOS runner')
    root = Path(env.get('RUNNER_TEMP', ''))
    if not root.is_absolute() or not root.is_dir():
        raise ValueError('absolute RUNNER_TEMP directory required')
    work = root / 'picfetch-apple-signing'
    if work.is_symlink():
        raise ValueError('unexpected signing directory symlink')
    return work


def inputs(env):
    for name in (*SECRET_NAMES, 'APPLE_STORE_APP', 'APPLE_STORE_SIGNED_OUTPUT_DIR'):
        if not env.get(name):
            raise ValueError('missing CI signing input: ' + name)
    for name, pattern in (('APPLE_STORE_TEAM_ID', r'[A-Z0-9]{10}'),
                          ('APPLE_STORE_APP_IDENTITY', r'[A-F0-9]{40}'),
                          ('APPLE_STORE_INSTALLER_IDENTITY', r'[A-F0-9]{40}'),
                          ('GITHUB_SHA', r'[a-f0-9]{40}')):
        if not re.fullmatch(pattern, env.get(name, '')):
            raise ValueError('invalid CI signing input: ' + name)
    decoded = {}
    for name in SECRET_NAMES:
        if name.endswith('_BASE64'):
            try:
                decoded[name] = base64.b64decode(''.join(env[name].split()), validate=True)
            except (ValueError, binascii.Error):
                raise ValueError('invalid base64 input: ' + name) from None
            if not decoded[name]:
                raise ValueError('empty decoded input: ' + name)
    manifest = json.loads((Path(env['APPLE_STORE_APP']).parent / 'manifest.json').read_text())
    if (manifest.get('source') != env['GITHUB_SHA'] or manifest.get('dirty') is not False or
            manifest.get('source_diff_sha256') != hashlib.sha256(b'').hexdigest() or
            manifest.get('untracked_sources') != {}):
        raise ValueError('qualification artifact must match the clean workflow revision')
    return decoded


def cleanup(env):
    work = work_directory(env)
    if not work.exists():
        return
    errors = []
    saved = work / 'search-list.json'
    if saved.exists():
        try:
            previous = json.loads(saved.read_text())
            security('list-keychains', '-d', 'user', '-s', *previous)
        except (OSError, ValueError, RuntimeError) as error:
            errors.append(error)
    keychain = work / 'signing.keychain-db'
    if keychain.exists():
        try:
            security('delete-keychain', keychain)
        except RuntimeError as error:
            errors.append(error)
    # Remove raw credentials even if restoring Keychain state needs a retry.
    for pattern in ('*.p12', '*.provisionprofile'):
        for path in work.glob(pattern):
            path.unlink()
    if errors:
        raise RuntimeError('CI credential cleanup incomplete; always() step must retry') from None
    shutil.rmtree(work)


def sign(env):
    work = work_directory(env)
    decoded = inputs(env)
    work.mkdir(mode=0o700)  # Never reuse another run's credential directory.
    try:
        previous = shlex.split(security('list-keychains', '-d', 'user'))
        (work / 'search-list.json').write_text(json.dumps(previous))
        keychain = work / 'signing.keychain-db'
        password = secrets.token_urlsafe(32)
        security('create-keychain', '-p', password, keychain)
        security('set-keychain-settings', '-lut', '3600', keychain)
        security('unlock-keychain', '-p', password, keychain)
        for role in ('APP', 'INSTALLER'):
            p12 = work / (role.lower() + '.p12')
            p12.write_bytes(decoded['APPLE_STORE_' + role + '_P12_BASE64'])
            security('import', p12, '-P', env['APPLE_STORE_' + role + '_P12_PASSWORD'],
                     '-t', 'agg', '-f', 'pkcs12', '-k', keychain,
                     '-T', '/usr/bin/codesign', '-T', '/usr/bin/productbuild', '-T', '/usr/bin/productsign')
            p12.unlink()
        security('set-key-partition-list', '-S', 'apple-tool:,apple:,codesign:', '-k', password, keychain)
        security('list-keychains', '-d', 'user', '-s', keychain)
        child = public_environment(env) | {'APPLE_STORE_TESTFLIGHT': '1'}
        for role in ('PROFILE', 'WORKER_PROFILE'):
            profile = work / (role.lower() + '.provisionprofile')
            profile.write_bytes(decoded['APPLE_STORE_' + role + '_BASE64'])
            child['APPLE_STORE_' + role] = str(profile)
        subprocess.run([sys.executable, str(Path(__file__).with_name('distribution.py'))],
                       check=True, env=child)
    finally:
        cleanup(env)


def cancelled(number, _):
    raise SystemExit(128 + number)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cleanup', action='store_true')
    options = parser.parse_args()
    signal.signal(signal.SIGTERM, cancelled)
    signal.signal(signal.SIGINT, cancelled)
    try:
        (cleanup if options.cleanup else sign)(os.environ)
    except (ValueError, RuntimeError, OSError, subprocess.CalledProcessError) as failure:
        print('CI signing failed: ' + str(failure), file=sys.stderr)
        sys.exit(1)
