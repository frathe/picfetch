## Title

DuplicatedCode findings are detected but silently dropped from the SARIF ("Can't find duplicate problem in db")

## Environment

- Linter image: `jetbrains/qodana-go:2026.2`
- CI runner: `JetBrains/qodana-action@v2026.2` on GitHub Actions, `ubuntu-latest`
- Inspection profile: `qodana.starter`, with the `DuplicatedCode` inspection re-enabled on top of that profile via an `include:` block in the repository's `qodana.yaml`:
  ```yaml
  include:
    - name: DuplicatedCode
  ```
- Scan type: a full scan (not `pr-mode`)
- Repository: a public Go repository, `https://github.com/frathe/picfetch`

## Summary

In a full-scan CI run of `jetbrains/qodana-go:2026.2` with `DuplicatedCode` enabled, the analysis detects a set of duplicate code fragments, and its own log records that result-writing then failed to find some of those fragments again when serialising them. The final SARIF output is missing exactly the fragments named in those log warnings. The tool's own JSON problem report (`result-allProblems.json`) and the SARIF file agree with each other and are both missing the same fragments, so the loss happens upstream of both output formats, somewhere between detection and whichever shared step writes both of them out.

## What we observed

We compared two independent measurements of the same Go source tree at commit `210fee5` (full SHA `210fee54929de03fc0316025834874f965df2cd0`) in the public repository `https://github.com/frathe/picfetch`: the CI run's own SARIF output, and an interactive lint pass over every tracked `.go` file run through the GoLand IDE's inspection engine, which uses the same `DuplicatedCode` inspection implementation.

The IDE pass reported 71 `DuplicatedCode` problems, one per duplicated code fragment, at 71 distinct `file:line` positions with no duplicate positions among them. The CI run's SARIF file (`qodana.sarif.json`) contained 33 `DuplicatedCode` results (each result is a cluster of two or more fragments that duplicate each other), whose `locations` arrays together held 71 location entries, but only 63 of those 71 location entries were distinct, identified by the triple `uri:startLine:charOffset` (8 fragments in the SARIF each appeared inside two different clusters, so they were each counted twice, reducing 71 location entries to 63 distinct fragments).

Comparing the CI run's 63 distinct fragments against the IDE's 71 fragments, using the key `file:startLine`: every one of CI's 63 fragments also appeared in the IDE's 71, and the CI set was a strict subset of the IDE set. The 8 fragments present in the IDE result but absent from the CI SARIF were, in full:

```
internal/imaging/loader_test.go:333
internal/imaging/loader_test.go:351
internal/imaging/loader_test.go:387
internal/imaging/loader_test.go:406
internal/imaging/loader_test.go:423
internal/imaging/loader_test.go:441
internal/imaging/loader_test.go:459
internal/update/tufroot_test.go:173
```

These 8 fragments live in exactly two files: 7 in `internal/imaging/loader_test.go` and 1 in `internal/update/tufroot_test.go`. The string `loader_test` does not appear anywhere in the CI run's `qodana.sarif.json` at all — that file is entirely absent from the SARIF, not merely under-represented.

The CI run's own log file, `log/idea.log`, contains exactly 3 warning-level log lines from a component named `DuplicatesProblem`, and no other warning from that component anywhere in the log. Immediately before them, the log records the line:

```
2026-08-29 19:14:27,077 [  69096]   INFO - #o.j.q.s.i.r.ConsoleLog - The Project analysis stage completed in 41s
```

The 3 warning lines that follow, quoted verbatim including their original timestamps and thread-id brackets, are:

```
2026-08-29 19:14:27,202 [  69221]   WARN - #o.j.q.s.i.r.g.DuplicatesProblem - Can't find duplicate problem in db, file://$PROJECT_DIR$/internal/imaging/loader_test.go:405:11963
2026-08-29 19:14:27,202 [  69221]   WARN - #o.j.q.s.i.r.g.DuplicatesProblem - Can't find duplicate problem in db, file://$PROJECT_DIR$/internal/imaging/loader_test.go:334:9850
2026-08-29 19:14:27,286 [  69305]   WARN - #o.j.q.s.i.r.g.DuplicatesProblem - Can't find duplicate problem in db, file://$PROJECT_DIR$/internal/update/tufroot_test.go:143:3919
```

Each warning names a file, a 1-based line number, and a 0-based character offset. These 3 warnings name exactly the 2 files that hold the 8 IDE-only fragments (`internal/imaging/loader_test.go` and `internal/update/tufroot_test.go`), and no other file in the repository.

This was not a one-off. A second, later CI run over a different commit (run `33274030031` at commit `f481c4a`, full SHA `f481c4a7847f1484d148fc4353fae80019d4c404`) logged the identical 3 `Can't find duplicate problem in db` warnings, naming the same 2 files at the same lines and character offsets, in its own `log/idea.log`.

## Why we believe detection succeeded

The 3 warning lines are timestamped at 19:14:27,202 and 19:14:27,286, which is after the log line `The Project analysis stage completed in 41s` (timestamped 19:14:27,077). That ordering places the warnings during the result-serialisation step that runs after the analysis stage has already finished, not during detection itself.

The line numbers and character offsets in the warnings, once decoded, land inside the same duplicated regions that the IDE independently reports. For example, the warning naming `loader_test.go:334` (0-based character offset 9850) sits one line away from the IDE's reported fragment at `loader_test.go:333`; the warning naming `loader_test.go:405` (character offset 11963) sits one line away from the IDE's fragment at `loader_test.go:406`; and the warning naming `tufroot_test.go:143` (character offset 3919) falls inside the region the IDE reports starting at `tufroot_test.go:141`, two lines past that region's start, whose duplicate partner is the IDE's other reported region at line 173 — the fragment absent from the SARIF — and lines 143 and 173 hold identical text (`srv := repo.serve()`). In every case the warned position lies within a region the IDE independently reports as duplicated, rather than at an unrelated location. This is consistent with detection having found the correct duplicated fragments and a lookup keyed on file/line/offset then failing at write time, rather than with detection having missed anything.

## Reproduction

A minimal pair that reproduces the discrepancy: the two fragments at `internal/imaging/loader_test.go:333` and `internal/imaging/loader_test.go:351`. In an IDE-based lint pass using the same `DuplicatedCode` inspection, both are reported as a `Duplicated code fragment (13 lines long)` pair, anchored on the line:

```
		loaded, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes)
```

In the corresponding CI run's `qodana.sarif.json`, neither line appears anywhere, and the string `loader_test` does not occur in the file at all.

To reproduce the IDE side of this comparison, run the `DuplicatedCode` inspection (as shipped in `jetbrains/qodana-go:2026.2`, the same inspection implementation an IDE-based lint pass uses) over a Go project containing two or more files with a repeated code pattern of at least a few lines, where the repeated pattern appears at least 8 times across at least 2 files. Reported fragment count should match the number of repeated occurrences.

To reproduce the CI side, run `jetbrains/qodana-go:2026.2` as a full scan (not `pr-mode`) over the same project with a profile that has `DuplicatedCode` enabled, and inspect the resulting `qodana.sarif.json` and `log/idea.log`. If the defect reproduces, `idea.log` will contain one or more warning lines of the form `Can't find duplicate problem in db, file://<path>:<line>:<charOffset>` emitted shortly after a log line reading `The Project analysis stage completed in <N>s`, and the fragments named in those warnings — and only those fragments — will be absent from `qodana.sarif.json`, even though other fragments from the same duplication analysis are present and correctly reported.

## Impact

A `DuplicatedCode` count read from `qodana.sarif.json`, or from any tool that consumes it, silently understates the true number of duplicate fragments the analysis actually found. In the run we examined, the true count was 71 fragments and the reported count was 63 — an 8-fragment undercount, with no indication in the SARIF, the HTML report, or any exit code that anything was dropped. The only signal that a problem occurred is a `WARN`-level line in `idea.log`, a file that is not normally read by CI gating logic and that carries no summary count of how many such warnings occurred. A CI pipeline that gates on the SARIF's `DuplicatedCode` result count, or on a report generated from it, will pass or fail based on an incomplete result set without any visible warning at the level most consumers look at.

## What we would expect

Either the fragments named in the `Can't find duplicate problem in db` warnings should still appear in the final SARIF output (a correctness fix to the lookup that is failing at serialisation time), or, if the analysis engine genuinely cannot recover a detected duplicate's data at write time, the run should fail loudly — a non-zero exit code, an ERROR-level log entry, or a note in the run summary — rather than silently emitting a SARIF file that is missing entries relative to what was actually detected.

## A second, separate issue: exclude globs compile but do not suppress

This is a distinct defect from the serialisation issue described above; the maintainers may want to track it as a separate ticket.

We added an `exclude:` block to `qodana.yaml` intended to suppress `DuplicatedCode` findings in test files, using a glob pattern in the `paths:` list:

```yaml
exclude:
  - name: DuplicatedCode
    paths:
      - "**/*_test.go"
```

In CI run `33273666731` at commit `d38f6e8` (full SHA `d38f6e84ae81f2d3ec36db54e02fd2442446317b`), this produced no change: the SARIF still contained 33 `DuplicatedCode` results, all still located in `_test.go` files. We then tried a second glob dialect, `"**_test.go"` in place of `"**/*_test.go"`, on the theory that Qodana's own built-in exclusions use a dialect where `**` already crosses path separators (the same file's `effective.profile.xml` lists built-in scopes such as `glob:**.md` and `glob:**.test.ts`, neither of which has a leading `**/`). In CI run `33274030031` at commit `f481c4a` (full SHA `f481c4a7847f1484d148fc4353fae80019d4c404`), this also produced no change: the SARIF again contained exactly 33 `DuplicatedCode` results, all still in `_test.go` files.

In both of these runs, `log/effective.profile.xml` contained the following element nested inside the `DuplicatedCode` inspection's own `<inspection_tool>` element:

```
<scope name="qodana.yaml.exclude.DuplicatedCode" level="INFORMATION" enabled="false" />
```

This shows that the `exclude:` entry was read, matched to the `DuplicatedCode` inspection by name, and compiled into a real scope object attached to that inspection — the exclusion mechanism engaged — yet the count and the set of reported fragments were completely unchanged by it in either run, meaning the compiled scope matched zero of the excluded files' findings.

We ruled out two ordinary explanations for this before concluding the exclusion itself was ineffective. First, delivery: each run's `log/qodana-config.json` contains an embedded copy of the exact `qodana.yaml` content that run was given, and in both runs that embedded copy showed the correct, intended `paths:` pattern for that run — the configuration reaching the container was not stale or wrong. Second, caching: neither run's `idea.log` showed any cache-hit or cache-restore message that could explain a stale, pre-exclusion result set being reused instead of a fresh analysis.

As a further control, we replaced the glob entirely with an explicit, literal list of the 30 file paths that contain the flagged test-only duplication, one path per line under the same `paths:` key. In CI run `33274422606` at commit `ed3d4e6` (full SHA `ed3d4e6cd40645dea8b97a1c95c4baf5e7dd1800`), this produced 0 `DuplicatedCode` results in the SARIF — full suppression. So the same `exclude: / name: DuplicatedCode / paths:` mechanism, under the same profile and inspection, does correctly suppress findings when the files are named as literal paths; it only fails to suppress anything when the `paths:` entry is a glob pattern, regardless of which of the two glob dialects we tried.
