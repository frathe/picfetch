"""Temporary native smoke: hold renderer-owned tile work until cancellation."""
import json
from pathlib import Path

root = Path.cwd()
work = Path('/private/tmp/picfetch-comparison-smoke')
source_path = root / 'internal/ui/compare/shader.go'
source = source_path.read_text().replace('import (', 'import (\n "fmt"\n "os"', 1)
needle = 'generateTile: generateRenderTile,'
replacement = '''generateTile: func(ctx context.Context, _ *renderSource, _ tileKey) (*renderTile, error) {
 fmt.Fprintln(os.Stderr, "[DEBUG-26-held] tile started")
 <-ctx.Done()
 fmt.Fprintln(os.Stderr, "[DEBUG-26-held] tile cancelled")
 return nil, ctx.Err()
},'''
assert source.count(needle) == 1
probe = work / 'shader-held.go'
probe.write_text(source.replace(needle, replacement, 1))
overlay = json.loads((work / 'overlay.json').read_text())
overlay['Replace'][str(source_path)] = str(probe)
(work / 'held-overlay.json').write_text(json.dumps(overlay))
