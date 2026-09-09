#!/bin/bash
set -euo pipefail

if [[ $(uname -s) != Darwin || $(uname -m) != arm64 ]]; then
    echo 'This experiment requires an Apple Silicon Mac.' >&2
    exit 1
fi

source_dir=$(cd "$(dirname "$0")" && pwd)
mkdir -p "${1:?pass the local asset directory}"
asset_dir=$(cd "$1" && pwd)
cd "$asset_dir"

if shasum -a 256 -c "$source_dir/../../internal/similarity/assets.sha256" >/dev/null 2>&1; then
    echo 'Pinned explorer assets already verified.'
    exit 0
fi

model_base='https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/resolve/ba1f3b0843f24bc5417d38e19c37b287d719b2f4'
curl -fL --retry 3 "$model_base/onnx/vision_model.onnx" -o vision_model.onnx.part
printf '%s\n' 'c0573e3f4140c3a7c4e9cc5912bd6b26a033b46a6a8e8af26cbea262b163bcad  vision_model.onnx.part' | shasum -a 256 -c -
mv vision_model.onnx.part vision_model.onnx
curl -fL --retry 3 "$model_base/preprocessor_config.json" -o preprocessor_config.json
curl -fL --retry 3 'https://github.com/microsoft/onnxruntime/releases/download/v1.29.0/onnxruntime-osx-arm64-1.29.0.tgz' -o onnxruntime.tgz.part
printf '%s\n' 'd0706fc34f315d8c88639d0a8c81f2e09e815f282cabed3493c06a054352cf92  onnxruntime.tgz.part' | shasum -a 256 -c -
tar -xzf onnxruntime.tgz.part
rm onnxruntime.tgz.part
shasum -a 256 -c "$source_dir/../../internal/similarity/assets.sha256"
