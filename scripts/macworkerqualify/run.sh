#!/bin/sh
# Qualify the native XPC process boundary with an ad-hoc signed fixture.
set -eu
repo=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
stage=$(mktemp -d /private/tmp/picfetch-worker.XXXXXX)
trap 'rm -rf "$stage"' EXIT HUP INT TERM
app="$stage/Qualification.app"
service="$app/Contents/XPCServices/io.github.frathe.picfetch.worker.xpc"
mkdir -p "$app/Contents/MacOS" "$service/Contents/MacOS"
cat > "$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0"?><plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>io.github.frathe.picfetch.workerqualification</string>
<key>CFBundleExecutable</key><string>qualification</string>
<key>CFBundlePackageType</key><string>APPL</string>
</dict></plist>
PLIST
cat > "$service/Contents/Info.plist" <<'PLIST'
<?xml version="1.0"?><plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>io.github.frathe.picfetch.worker</string>
<key>CFBundleExecutable</key><string>worker</string>
<key>CFBundlePackageType</key><string>XPC!</string>
<key>XPCService</key><dict><key>ServiceType</key><string>Application</string></dict>
</dict></plist>
PLIST
for kind in app service helper; do
  cat > "$stage/$kind.plist" <<'PLIST'
<?xml version="1.0"?><plist version="1.0"><dict>
<key>com.apple.security.app-sandbox</key><true/>
PLIST
  case "$kind" in
    app) echo '<key>com.apple.security.network.client</key><true/>' >> "$stage/$kind.plist" ;;
    helper) echo '<key>com.apple.security.inherit</key><true/>' >> "$stage/$kind.plist" ;;
  esac
  echo '</dict></plist>' >> "$stage/$kind.plist"
done
# Inject at the production signal-disposition boundary, without a runtime seam.
cat > "$stage/early-cancel.h" <<'HEADER'
#import <Foundation/Foundation.h>
#include <signal.h>
static inline void (*qualify_signal(int number, void (*handler)(int)))(int) {
    void (*previous)(int) = signal(number, handler);
    if (number == SIGTERM && previous != SIG_ERR) raise(SIGTERM);
    return previous;
}
#define signal qualify_signal
HEADER
xcrun clang -fobjc-arc -Wall -Wextra -Werror -framework Foundation "$repo/internal/macworker/native/client.m" -o "$app/Contents/MacOS/picfetch-worker-client"
xcrun clang -fobjc-arc -Wall -Wextra -Werror -framework Foundation -include "$stage/early-cancel.h" "$repo/internal/macworker/native/client.m" -o "$app/Contents/MacOS/picfetch-worker-client-early"
xcrun clang -fobjc-arc -Wall -Wextra -Werror -framework Foundation "$repo/internal/macworker/native/service.m" -o "$service/Contents/MacOS/worker"
xcrun clang -fobjc-arc -Wall -Wextra -Werror -framework Foundation "$repo/scripts/macworkerqualify/driver.m" -o "$app/Contents/MacOS/qualification"
xcrun clang -fobjc-arc -Wall -Wextra -Werror -framework Foundation "$repo/scripts/macworkerqualify/fixture.m" -o "$service/Contents/MacOS/picfetch-image-worker"
codesign --force --sign - --entitlements "$stage/helper.plist" "$app/Contents/MacOS/picfetch-worker-client"
codesign --force --sign - --entitlements "$stage/helper.plist" "$app/Contents/MacOS/picfetch-worker-client-early"
codesign --force --sign - --entitlements "$stage/helper.plist" "$service/Contents/MacOS/picfetch-image-worker"
codesign --force --sign - --entitlements "$stage/service.plist" "$service"
codesign --force --sign - --entitlements "$stage/app.plist" "$app"
codesign --verify --strict "$app"
python3 "$repo/scripts/macworkerqualify/check.py" "$app/Contents/MacOS/qualification"
