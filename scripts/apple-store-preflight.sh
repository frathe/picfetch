#!/bin/bash
# Read-only developer-tool and signing-input checks. This is not a validation
# of a signed app, its sandbox behavior, or its App Store submission.
set -eu

export DEVELOPER_DIR="${APPLE_STORE_DEVELOPER_DIR:-/Applications/Xcode.app/Contents/Developer}"
failures=0
fail() {
    printf 'NEEDED: %s\n' "$1" >&2
    failures=$((failures + 1))
}

if [ "$(uname -s)" != Darwin ]; then
    printf 'Mac App Store preflight requires macOS.\n' >&2
    exit 1
fi

if /usr/bin/xcodebuild -version; then
    printf 'Xcode developer directory: %s\n' "$DEVELOPER_DIR"
else
    fail 'Install full Xcode, complete its first launch, or set APPLE_STORE_DEVELOPER_DIR.'
fi

if ! /usr/bin/xcrun clang --version >/dev/null; then
    fail 'Complete full Xcode first launch and review/accept its license locally with xcodebuild -license.'
fi

for tool in /usr/bin/codesign /usr/bin/productbuild /usr/sbin/pkgutil /usr/bin/security; do
    if [ ! -x "$tool" ]; then
        fail "Apple tool $tool is unavailable."
    fi
done

team="${APPLE_STORE_TEAM_ID:-}"
if [[ ! "$team" =~ ^[A-Z0-9]{10}$ ]]; then
    fail 'Set APPLE_STORE_TEAM_ID to the 10-character developer Team ID.'
else
    work=$(mktemp -d)
    trap 'rm -rf "$work"' EXIT
    # Check for identities with private keys. Do not print the keychain's
    # account names, certificate fingerprints, or provisioning-profile data.
    if /usr/bin/security find-identity -v -p codesigning > "$work/app-identities"; then
        if ! awk -v team="$team" '
            /"(Apple Distribution:|3rd Party Mac Developer Application:)/ &&
            index($0, "(" team ")") { found = 1 }
            END { exit !found }
        ' "$work/app-identities"; then
            fail 'Install a valid Mac App Store application signing identity for this Team ID.'
        fi
    else
        fail 'Unable to inspect application signing identities.'
    fi
    if /usr/bin/security find-identity -v > "$work/all-identities"; then
        if ! awk -v team="$team" '
            /"3rd Party Mac Developer Installer:/ &&
            index($0, "(" team ")") { found = 1 }
            END { exit !found }
        ' "$work/all-identities"; then
            fail 'Install a valid Mac Installer Distribution signing identity for this Team ID.'
        fi
    else
        fail 'Unable to inspect installer signing identities.'
    fi
fi

profile="${APPLE_STORE_PROFILE:-}"
if [ -z "$profile" ] && [ "${APPLE_STORE_TESTFLIGHT:-0}" != 1 ]; then
    printf 'No profile supplied; unrestricted Mac App Store entitlements may omit it. TestFlight always requires one.\n'
elif [ -z "$profile" ] || [ ! -f "$profile" ]; then
    fail 'Set APPLE_STORE_PROFILE to the Mac App Store Connect provisioning profile for io.github.frathe.picfetch.'
else
    printf 'Provisioning profile supplied; its signature, identity, expiry and entitlements still require artifact validation.\n'
fi

if [ "$failures" -ne 0 ]; then
    printf '%s preparation inputs remain unresolved.\n' "$failures" >&2
    exit 1
fi
printf 'Preparation inputs found. Signed sandbox qualification and final package validation are still required.\n'
