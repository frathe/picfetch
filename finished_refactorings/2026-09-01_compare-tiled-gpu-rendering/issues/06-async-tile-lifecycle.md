# 06 - Add asynchronous tile delivery and lifecycle control

Status: resolved

## Contract

Run at most one cancellable tile worker per pane. Check source/view tokens
between bounded generations, publish through `UIQueue`, ignore stale results,
reuse cached tiles across Swap, and make `Settle` wait for load, vector, tile,
and queued follow-on work.

## Acceptance

Tests cover supersession, close, reopen, resize, vector replacement, swap/cache
reuse, queue ordering, worker count, and race-safe settlement.

## Comments

The initial async tests proved single-generator execution, stale-source
rejection, cache reuse, and settlement. Lifecycle audit then reproduced a
vector-replacement deadlock caused by waiting for obsolete tiles before
draining the UI completion that cancels them. `Settle` now drains causal work
first, reusable channel epochs replace cancellable WaitGroup helper goroutines,
tile publication is coalesced, and focused tests cover active clear/reopen,
stale views, vector replacement, active Swap, shutdown, and race settlement.

The first native memory run exposed one more lifecycle cost: every same-source
view revision cancelled a tile after its destination buffer could already be
allocated. With a large decoded-image cache raising Go's collection goal,
discarded four-megabyte tile buffers accumulated between collections. A guard
was observed failing with an allocated tile discarded on view change. The
worker now finishes and caches that one immutable tile, then jumps directly to
the latest same-source plan; clear and source replacement still cancel.
