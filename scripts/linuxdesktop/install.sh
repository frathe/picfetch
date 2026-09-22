#!/bin/sh
set -eu

# This script is distributed beside the binary, launcher template and notices.
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
data_home=${XDG_DATA_HOME:-"${HOME:?HOME must be set}/.local/share"}
case "$data_home" in
    /*) ;;
    *) printf '%s\n' 'XDG_DATA_HOME must be an absolute path.' >&2; exit 1 ;;
esac
if printf '%s' "$data_home" | LC_ALL=C grep '[[:cntrl:]=]' >/dev/null; then
    printf '%s\n' 'The installation path cannot contain control characters or an equals sign.' >&2
    exit 1
fi

install_dir="$data_home/@APP_ID@"
applications="$data_home/applications"
mkdir -p "$install_dir" "$applications"
binary_tmp=$(mktemp "$install_dir/.binary.XXXXXX")
desktop_tmp=$(mktemp "$applications/.@APP_ID@.XXXXXX")
trap 'rm -f "$binary_tmp" "$desktop_tmp"' 0
cp "$source_dir/@EXECUTABLE@" "$binary_tmp"
chmod 755 "$binary_tmp"
for name in LICENSE THIRD-PARTY-NOTICES.md PRIVACY.md '@APP_ID@.png'; do
    cp "$source_dir/$name" "$install_dir/$name"
done
mv -f "$binary_tmp" "$install_dir/@EXECUTABLE@"

# Desktop strings are unescaped before Exec argument quoting is interpreted.
# Percent signs are doubled so paths cannot become desktop field codes.
# env keeps desktop parsers from resolving a percent-escaped executable name
# before field expansion. It receives one absolute binary path, without a shell.
exec_path=$(printf '%s' "$install_dir/@EXECUTABLE@" | sed 's/\\/\\\\\\\\/g; s/"/\\\\"/g; s/\$/\\\\$/g; s/`/\\\\`/g; s/%/%%/g')
icon_path=$(printf '%s' "$install_dir/@APP_ID@.png" | sed 's/\\/\\\\/g')
while IFS= read -r line; do
    case "$line" in
        Exec=*) printf 'Exec=/usr/bin/env "%s" %%F\n' "$exec_path" ;;
        Icon=*) printf 'Icon=%s\n' "$icon_path" ;;
        *) printf '%s\n' "$line" ;;
    esac
done < "$source_dir/@APP_ID@.desktop" > "$desktop_tmp"
chmod 644 "$desktop_tmp"
mv -f "$desktop_tmp" "$applications/@APP_ID@.desktop"
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$applications"
fi
printf 'Installed PicFetch in %s\n' "$install_dir"
