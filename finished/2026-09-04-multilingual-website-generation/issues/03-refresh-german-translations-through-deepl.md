# 03: Refresh German translations through DeepL

Type: task
Status: resolved
Blocked by: 01

**What to build:** Let maintainers deliberately refresh a safe, reviewable German translation cache from the English source while keeping ordinary builds offline, deterministic, and independent of live credentials.

- [x] Every language-sensitive source value, including metadata and accessibility text, is exposed as a stable translation unit.
- [x] URLs, commands, code, filenames, keyboard labels, IDs, dimensions, product names, and designated technical terms are protected from translation.
- [x] The committed German cache keys entries by stable semantic identity and the current English source hash.
- [x] `make translate` requests only missing or changed units, reuses current entries, and removes obsolete derived entries deterministically.
- [x] Translation requests use the supported DeepL API Developer contract, target German, and send extracted text or protected markup rather than raw Markdown.
- [x] Authentication comes from `DEEPL_API_KEY` or explicitly ignored local environment configuration and is never committed, rendered, logged, or placed in test fixtures.
- [x] Missing credentials and authentication, quota, rate-limit, network, malformed-response, and partial-response failures produce clear errors without partially replacing the prior cache.
- [x] A complete German cache exists for the current English source before the ticket is considered complete.
- [x] Offline generation rejects missing or stale German entries instead of silently substituting English.
- [x] Contract tests use a controlled local HTTP server to verify request protection, authentication placement, target language, response mapping, batching, and failure atomicity.
- [x] Routine tests make no external network calls, consume no translation allowance, and require no real API key.
- [x] Cache changes are readable in a normal Git diff so a maintainer can review translated output before publishing.

## Answer

Implemented a strict, incremental DeepL refresh that stores stable content IDs with
their current source hashes, protects opaque markup and technical values, prunes
obsolete entries, and replaces the cache atomically only after a complete validated
response. After explicit maintainer authorization, the production refresh completed
through the ignored local credential and produced 68 current German entries. The
credential is excluded from validator subprocesses, output, and diagnostics.
