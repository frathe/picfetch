# Find more like this: local evaluation

Status: technical experiment complete; relevance judgments and proceed/revise
decision pending. This does not complete FML-001 or admit FML-002.

## Retained run

On 2026-09-14, the approved local demo produced a successful run with 446
represented files and no failures (422 JPEG, 24 HEIC). The manifest chooses
20 content references from distinct prior Explorer cohorts and adds four
appearance queries. This selection supplies a review starting point; it does
not establish that the references cover all required semantic cases.

```sh
make explorer-evaluate TRIAL=search EXPLORER_EVIDENCE=.scratch/find-more-like-this/evidence
```

Evidence: [local review](../../.scratch/find-more-like-this/evidence/search-JyDNM0/result/search-review.html),
[retained results](../../.scratch/find-more-like-this/evidence/search-JyDNM0/result/search-result.json),
[Favorite baseline](../../.scratch/find-more-like-this/evidence/search-JyDNM0/result/favorite-profile.json).
The run's parent directory retains the console log and exit status `0`.
Private previews, paths, vectors and judgments stay in ignored local evidence.
No private image pixels were sent to a remote model for this evaluation.

Machine: Apple M5 Max, 48 GiB RAM, macOS 26.6.2; darwin/arm64, Go 1.27.1.
CPU provider with six inference threads; canonical full oriented decoding,
bilinear 224 preprocessing, exact cosine in the original 768 dimensions.
Existing pinned model revision:
`ba1f3b0843f24bc5417d38e19c37b287d719b2f4`.
Representation suffix: `/oriented-bilinear-224-v1`.
The installed assets were reused; no dependency or asset version changed.

Corpus SHA-256:
`1a76610a0a183a4896b396adc33563dc3d85eacb6b04dd646053fbf721d45e00`.
Source identity/version/content SHA-256:
`4850774efb2f500851c14f14f6d3a67e4b4fdf38b0589cb2c493d9a92be9015e`.

## Observed measurements

| Observation | Result | Boundary |
| --- | ---: | --- |
| First ranked result | 20.631 s | First reference prepared first; publication after 100 processed sources. |
| Complete preparation | 85.175 s | All 446 represented. |
| Warm query p50 / p95 | 0.599 / 0.625 ms | Exact ranking on prepared corpus; no inference. |
| Synthetic 10,000-vector p50 / p95 | 13.875 / 14.192 ms | 30 timed queries after warm-up; deterministic vectors, 30,720,000 vector bytes. |
| Worker maximum RSS | 1,166,065,664 bytes | After ranking/benchmark, before report serialization; excludes parent and separate Favorite workers. |
| Retained real-source vector storage | 1,370,112 bytes | 446 x 768 float32 values; distinct from RSS and disk. |
| Favorite cold worker | 86.639 s; 446 inference attempts | Separate production Explorer baseline including map work. |
| Favorite warm worker | 1.360 s; 446 reused; zero inference | New production worker, same temporary Favorite cache. |
| Temporary Favorite disk usage | 7,660,194 bytes | Whole temporary baseline directory, including its file list; removed after measurement. |

The first-result observation is below the initial 30-second target and the
synthetic warm p95 is below 200 ms. Development checks ran concurrently, so
these are exploratory local measurements, not controlled final qualification.
They do not measure Grid paint/interaction, general-cache reuse or the final
production search session. Those remain for FML-008. The trial uses full-sort
ranking; FML-002 will supply the bounded production top-k interface.

## Quality decision still required

Content judgments: **0 of 20**. Appearance judgments: **0 of 4**. Neither median
precision is measured. The 0.6 median content P@10 target cannot yet be assessed.
Do not infer usefulness from timing, cohort diversity, or cosine values.

The local report supports marking relevance, marking each reference reviewed,
and downloading the updated corpus without rerunning inference. Preserve that
file and record the human **proceed** or **revise** decision here. Confirm the
review covers duplicates, hard negatives and weak matches, adding references
if needed. Appearance judgments stay separate from content acceptance.

Synthetic command/report regressions cover self-exclusion, equal-score path
ordering, distinct copies, unreadable inputs, failed/malformed representations,
weak negative-score matches, small scopes and missing-slot precision. They
prove accounting and failure behavior, not semantic relevance. macOS native
regressions also prove the network-denied worker and cold/warm Favorite reuse.
Other-platform search UI qualification remains unverified.
