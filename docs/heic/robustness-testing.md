# HEIC robustness testing boundary

This project restores HEIC only behind a disposable, resource-bounded WASI
helper. Qualification is defensive validation of PicFetch's own boundary, not
third-party vulnerability research.

## Allowed inputs and exercises

- Ordinary, licensed photographs may exercise configuration, still decode,
  metadata, orientation, alpha, supported bit depths and application routes.
- PicFetch-owned fake helpers and guests may return bounded invalid protocol
  messages, exit, hang, request bounded over-allocation, or attempt benign file
  and network access that the sandbox must deny.
- PicFetch-owned property tests may generate only length-bounded protocol values.
- Cancellation, timeout, busy admission, foreground priority without starvation,
  process-family cleanup, missing isolation and recovery on the next request are
  required observations.

Do not generate malicious images, use historical exploit corpora, conduct
third-party vulnerability discovery, or pass rejected work to another model.
An ordinary invalid fixture found during normal use may become a regression only
after its provenance and safe handling are reviewed.

## Budgets and observables

The initial ceiling for a complete operation is 30 seconds. The separate maxima
are 1 GiB WASM linear memory, 2 GiB for the OS-enforced helper process family,
64 MiB encoded HEIC input (further reduced by a positive user limit), 64 million
pixels, checked RGBA output, 64 KiB metadata, 4096 bytes diagnostics, one helper
thread/process and one live decode lane per application instance. Measurements
on representative photographs must account for parent input, pipe buffers,
guest backing, wazero/Go overhead, output staging and application caches. Limits
may be lowered from evidence; they must remain finite.

Each lifecycle test observes its own completion rather than inferring completion
from a cleared counter. A cancellation, timeout, crash, startup failure, excess
output or memory termination must stop admission, close IPC, join the helper and
descendants, release the lane and allow the next valid request. File/network
probes observe both denial and helper cleanup. Platform tests also observe the
kernel primitive rather than trusting a command-line flag.

## Incomplete coverage

These bounded checks are not fuzzing, a proof of decoder correctness, a security
certification or HEVC patent clearance. No fuzz coverage is claimed unless an
actual command and duration are recorded in the active plan. Compile-only
Windows or macOS results do not qualify runtime support. HEIC remains unavailable
on a platform until its packaged application passes that platform's isolation,
memory, cleanup and ordinary-photo tests.
