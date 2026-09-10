#!/bin/bash
set -euo pipefail

source_dir=$(cd "$(dirname "$0")" && pwd)
mkdir -p "${1:?pass the local asset directory}"
asset_dir=$(cd "$1" && pwd)
cd "$source_dir/../.."
# Share the application's verified, cancellable installer on every supported OS.
exec go run ./scripts/explorereval -install -assets "$asset_dir"
