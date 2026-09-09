"""Prepare fixed English prompts with the pinned multilingual Gemma tokenizer.

Development only: requires sentencepiece==0.2.1. No user inputs or images.
"""

import hashlib
import json
import sys
from pathlib import Path

import sentencepiece

model, catalog = map(Path, sys.argv[1:3])
expected = "61a7b147390c64585d6c3543dd6fc636906c9af3865a5548f27f31aee1d4c8e2"
if hashlib.sha256(model.read_bytes()).hexdigest() != expected:
    raise SystemExit("tokenizer checksum mismatch")
sp = sentencepiece.SentencePieceProcessor(model_file=str(model))
encoded = []
for tag in json.loads(catalog.read_text())["tags"]:
    tokens = sp.encode(tag["prompt"].lower(), out_type=int)[:63] + [1]
    encoded.append(tokens + [0] * (64 - len(tokens)))
json.dump(encoded, sys.stdout)
print()
