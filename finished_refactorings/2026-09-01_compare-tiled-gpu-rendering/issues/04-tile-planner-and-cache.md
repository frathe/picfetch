# 04 - Implement the virtual tile planner and bounded cache

Status: resolved

## Contract

Plan visible guttered 1022-pixel-interior tiles from source coordinates and
physical display density. Coarsen until no more than seven visible tiles are
needed, then prefetch nearest neighbors. Cache generated tiles per source with a
64 MiB byte budget.

## Acceptance

Table tests cover fit/fill/zoom/pan/edge/HiDPI cases, deterministic slot order,
mixed levels, gutters, cache hits/eviction, and the seven-detail invariant.

## Comments

The planner/cache tests failed first on the absent plan, key, generator, and
budget. They now cover fit, HiDPI, zoom, sampler-forced coarsening, deterministic
nearest prefetch, exact level-zero gutters, hits, and 64 MiB eviction. The full
compare suite passes. The shader audit then exposed odd-dimension coarse-edge
gutter drift, prefetch-biased LRU promotion, unnecessary detail work below
overview density, and full-pane planning behind a swipe clip. Each received a
focused regression and now passes.
