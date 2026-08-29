# Qodana CI evidence for run 33270269940

This document freezes what the CI Qodana artifact for GitHub Actions run
`33270269940` actually contains. Every number below was produced by running the
command shown for it against the downloaded artifact at
`$SCRATCH/qodana/x/` (SCRATCH =
`/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad`).
No number here is inferred or remembered; each is the direct output of a command
in the `## Reproduction` section.

## Run identity

Run `33270269940` was inspected with `gh run view 33270269940 --json headSha,conclusion,event`.
The command returned:

```json
{"conclusion":"success","event":"push","headSha":"210fee54929de03fc0316025834874f965df2cd0"}
```

`headSha` is `210fee54929de03fc0316025834874f965df2cd0`, `conclusion` is `success`, and
`event` is `push`. This SHA corresponds to the repository's short SHA `210fee5`
("ci: make Qodana reports inspectable" in `git log`). Every count in this document is
bound to this SHA and this run.

The artifact directory at `$SCRATCH/qodana/x` was already present and contained:
`log`, `open-in-ide.json`, `projectStructure`, `qodana-short.sarif.json`,
`qodana.sarif.json`, `report`. No re-download was needed.

## CI inventory at 210fee5

Two sources were read for run 210fee5: the SARIF results in `qodana.sarif.json`
(what the run reports as findings) and the per-rule totals in
`log/qodana_inspections_summary.csv` (what the run's inspection pass counted before
any filtering into the SARIF).

`qodana.sarif.json` contains 34 results in total: 33 with `ruleId` `DuplicatedCode`
and 1 with `ruleId` `GoTypeAssertionOnErrors`. No other `ruleId` appears among the
SARIF results.

`qodana_inspections_summary.csv` has exactly seven rows with a nonzero count in
column 8:

| Rule | CSV count | SARIF result count |
|---|---|---|
| GoMaybeNil | 2 | 0 |
| GoBoolExpressions | 3 | 0 |
| DuplicatedCode | 75 | 33 |
| GoRedundantConversion | 1 | 0 |
| GoTypeAssertionOnErrors | 1 | 1 |
| GoErrorStringFormat | 2 | 0 |
| GoVarAndConstTypeMayBeOmitted | 4 | 0 |
| **Total** | **88** | **34** |

## Suppression accounting

Of the 88 problems counted in the CSV inspection summary, only 34 appear as
results in the final SARIF file. The gap breaks down as follows.

For the five rule categories other than `DuplicatedCode` and
`GoTypeAssertionOnErrors` — `GoMaybeNil`, `GoBoolExpressions`,
`GoRedundantConversion`, `GoErrorStringFormat`, `GoVarAndConstTypeMayBeOmitted` —
the CSV counts sum to 2 + 3 + 1 + 2 + 4 = 12. None of these five `ruleId` values
appears anywhere in the SARIF results (the distinct `ruleId` values present in the
SARIF are exactly `DuplicatedCode` and `GoTypeAssertionOnErrors`). So all 12 of
the 12 problems counted in these categories are absent from the final SARIF:
12-of-12 suppressed.

For `GoTypeAssertionOnErrors`, the CSV count (1) equals the SARIF result count
(1) exactly. No suppression is observed for this rule.

For `DuplicatedCode`, the CSV count is 75. The SARIF has 33 `DuplicatedCode`
results (clusters). Summing the `locations` array length across those 33 results
gives 71 total fragment locations, and summing `relatedLocations` (defaulting
missing arrays to empty) across the same 33 results gives 0. `related` being 0
is what rules out "cluster members are hidden in `relatedLocations`" as an
alternative explanation for the gap between the CSV's 75 and the SARIF's 33
results: there is no separate pool of related locations to account for the
difference, only the 71 locations already counted directly on each result's
`locations` array.

Four numbers were observed for `DuplicatedCode`, each with its own unit: 33
cluster results, 71 location slots (the sum of each result's `locations`
array length), 63 distinct fragments (unique `uri:startLine:charOffset`
triples across all 71 location slots), and 75 CSV rows. The 71 location
slots are not disjoint: identifying each fragment by the triple
`uri:startLine:charOffset` shows that 8 fragments each appear in 2 different
clusters, which is why 71 slots reduce to only 63 distinct fragments (55
fragments appear in exactly 1 cluster, 8 fragments appear in exactly 2
clusters: 55 × 1 + 8 × 2 = 71). The 8 repeated fragments are:

```
internal/imaging/raw_test.go:348:10949
internal/imaging/raw_test.go:395:12195
internal/ui/exifwin/exifwin_test.go:751:20987
internal/ui/exifwin/exifwin_test.go:784:22072
internal/ui/grid_test.go:284:9102
internal/ui/grid_test.go:331:10785
internal/ui/slideshow_test.go:449:14310
internal/ui/slideshow_test.go:495:15662
```

Because 71 is a slot count, not a fragment count, it is not the right
quantity to compare against the CSV's 75. The CSV-to-SARIF gap for this rule
is therefore 75 − 63 = 12 fragments: 75 fragments counted in the CSV versus
63 distinct fragments appearing anywhere in the SARIF's `DuplicatedCode`
results. The commands run do not establish what those 12 fragments are or
why they are absent from the SARIF; no hypothesis is offered here.

> **Superseded 2026-08-29:** run `33274422606` at `ed3d4e6` excluded the 30
> test files and the CSV's `DuplicatedCode` count fell from 75 to 4, not to
> 0; 4 is the count of source-local `//goland:noinspection` suppressions on
> the orientation pixel loops. Therefore 75 CSV rows = 71 test-file
> fragments + 4 production fragments, and the 12-fragment gap = 8
> serialisation losses + 4 source suppressions, with nothing left
> unexplained.

## Duplication clusters

The SARIF contains 33 `DuplicatedCode` results, each treated here as one
cluster, numbered 1 through 33 in the order they appear in `qodana.sarif.json`.
Each cluster's fragments are shown as `file:startLine (N lines)`, where N is
the number of lines in that location's `snippet.text` field. These SARIF
region objects carry `charOffset` and `charLength` rather than an `endLine`
field.

Of the 33 clusters, 29 have every fragment in a single file, and 4 have
fragments spanning two different files (clusters 3, 5, 13, and 28 below).

| # | Fragments |
|---|---|
| 1 | internal/imaging/orientation_test.go:129 (13 lines), internal/imaging/orientation_test.go:149 (13 lines) |
| 2 | internal/ui/grid_test.go:284 (20 lines), internal/ui/grid_test.go:331 (20 lines) |
| 3 | internal/ui/grid_test.go:284 (8 lines), internal/ui/grid_test.go:331 (8 lines), internal/ui/step_test.go:229 (10 lines) |
| 4 | internal/update/tufroot_repo_test.go:76 (8 lines), internal/update/tufroot_repo_test.go:84 (8 lines) |
| 5 | internal/clipboard/clipboard_test.go:13 (16 lines), internal/clipboard/copyfiles_test.go:31 (16 lines) |
| 6 | internal/ui/grid/dupes_test.go:603 (13 lines), internal/ui/grid/dupes_test.go:639 (13 lines) |
| 7 | internal/ui/exifwin/exifwin_test.go:584 (13 lines), internal/ui/exifwin/exifwin_test.go:623 (13 lines), internal/ui/exifwin/exifwin_test.go:646 (13 lines) |
| 8 | internal/ui/imgcache_test.go:87 (7 lines), internal/ui/imgcache_test.go:107 (7 lines) |
| 9 | internal/ui/exifwin/exifwin_test.go:751 (10 lines), internal/ui/exifwin/exifwin_test.go:784 (10 lines) |
| 10 | internal/imaging/raw_test.go:318 (12 lines), internal/imaging/raw_test.go:374 (12 lines) |
| 11 | internal/update/apply_test.go:17 (13 lines), internal/update/apply_test.go:154 (13 lines) |
| 12 | internal/imaging/raw_test.go:275 (15 lines), internal/imaging/raw_test.go:348 (15 lines), internal/imaging/raw_test.go:395 (15 lines) |
| 13 | internal/filescan/filescan_test.go:73 (25 lines), internal/ui/drop_test.go:288 (16 lines) |
| 14 | internal/update/attest_test.go:245 (12 lines), internal/update/attest_test.go:266 (12 lines) |
| 15 | internal/imaging/gif_test.go:360 (7 lines), internal/imaging/gif_test.go:401 (6 lines) |
| 16 | internal/ui/autoupdate_test.go:357 (11 lines), internal/ui/autoupdate_test.go:433 (13 lines) |
| 17 | internal/filepicker/filepicker_test.go:76 (13 lines), internal/filepicker/filepicker_test.go:119 (13 lines) |
| 18 | internal/update/tufroot_test.go:54 (10 lines), internal/update/tufroot_test.go:141 (10 lines) |
| 19 | internal/ui/favthumbs_test.go:63 (10 lines), internal/ui/favthumbs_test.go:102 (10 lines) |
| 20 | internal/ui/favorites/confirm_test.go:35 (19 lines), internal/ui/favorites/confirm_test.go:60 (19 lines) |
| 21 | internal/imaging/gif_test.go:200 (8 lines), internal/imaging/gif_test.go:250 (8 lines) |
| 22 | internal/ui/filestate_test.go:38 (17 lines), internal/ui/filestate_test.go:71 (17 lines) |
| 23 | internal/imaging/save_test.go:252 (11 lines), internal/imaging/save_test.go:297 (11 lines) |
| 24 | internal/ui/menu_test.go:495 (16 lines), internal/ui/menu_test.go:571 (16 lines) |
| 25 | internal/imaging/orientation_test.go:98 (22 lines), internal/imaging/orientation_test.go:184 (22 lines) |
| 26 | internal/ui/favorites/favorites_test.go:663 (14 lines), internal/ui/favorites/favorites_test.go:699 (14 lines) |
| 27 | internal/ui/slideshow_test.go:449 (16 lines), internal/ui/slideshow_test.go:495 (13 lines) |
| 28 | internal/favthumbs/store_test.go:184 (16 lines), internal/favthumbs/sweep_test.go:14 (16 lines) |
| 29 | internal/imaging/raw_test.go:348 (17 lines), internal/imaging/raw_test.go:395 (17 lines) |
| 30 | internal/ui/exifwin/exifwin_test.go:751 (8 lines), internal/ui/exifwin/exifwin_test.go:784 (8 lines), internal/ui/exifwin/exifwin_test.go:1061 (9 lines) |
| 31 | internal/favthumbs/sync_test.go:285 (10 lines), internal/favthumbs/sync_test.go:315 (10 lines) |
| 32 | internal/ui/slideshow_test.go:449 (14 lines), internal/ui/slideshow_test.go:475 (11 lines), internal/ui/slideshow_test.go:495 (11 lines) |
| 33 | internal/update/extract_test.go:186 (18 lines), internal/update/extract_test.go:213 (18 lines) |

Every fragment's `artifactLocation.uri` across all 33 clusters was checked
against the pattern `_test\.go$`. The count of fragments that do **not** match
this pattern is 0 — every fragment in every cluster lives in a `_test.go` file.
No production (non-test) file is involved in any `DuplicatedCode` cluster in
this run.

## Effective profile

Running the `jq` filter over `$SCRATCH/qodana/x/log/qodana-config.json` returned
content (the filter did not return nothing, so the Step 8 fallback to
`qodana.yaml` was not needed). Piping that content through
`grep -A2 '^profile:'` produced:

```
profile:
  name: qodana.starter
#Enable inspections
```

The effective profile recorded by the run's own config dump is `qodana.starter`.

Separately, the repository's `qodana.yaml` was read directly (not as a
fallback, since the `jq` query above already succeeded, but as a cross-check).
Its `profile:` block matches the run's config dump exactly (`name:
qodana.starter`), and it also carries an active, uncommented section at the end
of the file:

```yaml
include:
  - name: DuplicatedCode
```

This confirms the plan's premise that `qodana.yaml` re-enables the
`DuplicatedCode` inspection via `include:`, on top of whatever the
`qodana.starter` profile does or does not enable by default. The dumped config
content from the run and the repository's `qodana.yaml` file were observed to
be textually identical.

## Reproduction

Every command below was run against the artifact for run `33270269940` at
SCRATCH = `/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad`.

Step 1 — confirm the artifact is present, or fetch it:

```bash
SCRATCH=/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad
test -f "$SCRATCH/qodana/x/qodana.sarif.json" || {
  gh run download 33270269940 -n qodana-report -D "$SCRATCH/qodana"
  unzip -o -q "$SCRATCH/qodana/qodana-report.zip" -d "$SCRATCH/qodana/x"
}
ls "$SCRATCH/qodana/x"
```

Output: `log  open-in-ide.json  projectStructure  qodana-short.sarif.json  qodana.sarif.json  report`
(no fetch was needed; the file was already present).

Step 2 — confirm the run's identity:

```bash
gh run view 33270269940 --json headSha,conclusion,event
```

Output: `{"conclusion":"success","event":"push","headSha":"210fee54929de03fc0316025834874f965df2cd0"}`

Step 3 — reproduce the per-rule SARIF counts:

```bash
SARIF="$SCRATCH/qodana/x/qodana.sarif.json"
jq '[.runs[].results[]] | length' "$SARIF"
jq -r '[.runs[].results[].ruleId] | group_by(.) | map({r:.[0],n:length}) | sort_by(-.n) | .[] | "\(.n)\t\(.r)"' "$SARIF"
```

Output:

```
34
33	DuplicatedCode
1	GoTypeAssertionOnErrors
```

Step 4 — reproduce the per-rule CSV counts:

```bash
awk -F';' 'NR>1 && $8+0>0 {print $8"\t"$1}' "$SCRATCH/qodana/x/log/qodana_inspections_summary.csv"
```

Output:

```
2	GoMaybeNil
3	GoBoolExpressions
75	DuplicatedCode
1	GoRedundantConversion
1	GoTypeAssertionOnErrors
2	GoErrorStringFormat
4	GoVarAndConstTypeMayBeOmitted
```

Step 5 — reproduce the cluster/fragment arithmetic:

```bash
jq -r '[.runs[].results[] | select(.ruleId=="DuplicatedCode")] | {results:length, locations:(map(.locations|length)|add), related:(map(.relatedLocations//[]|length)|add)}' "$SARIF"
```

Output: `{"results":33,"locations":71,"related":0}`

Step 6 — emit the full 33-cluster inventory:

```bash
jq -r '.runs[].results[]|select(.ruleId=="DuplicatedCode")|"| " + ([.locations[]|.physicalLocation|.artifactLocation.uri+":"+(.region.startLine|tostring)+" ("+((.region.snippet.text//"")|split("\n")|length|tostring)+" lines)"]|join(", ")) + " |"' "$SARIF"
```

Before using this command, its assumption about `snippet` was checked:

```bash
jq -r '[.runs[].results[]|select(.ruleId=="DuplicatedCode")|.locations[0].physicalLocation.region|keys]|add|group_by(.)|map("\(.[0]) x\(length)")|.[]' "$SARIF"
jq -r '.runs[].results[0].locations[0].physicalLocation.region.snippet | type' "$SARIF"
```

Output:

```
charLength x33
charOffset x33
snippet x33
sourceLanguage x33
startColumn x33
startLine x33
object
```

Every one of the 33 `DuplicatedCode` results' region objects carries exactly
the same six keys — `charLength`, `charOffset`, `snippet`, `sourceLanguage`,
`startColumn`, `startLine` — and no `endLine` key at all, and `snippet` is a
JSON object (not a plain string), so `.region.snippet.text` is a valid path
and the command above works as intended.

Raw output of the working command (one ragged row per cluster, fragments
comma-separated, kept here verbatim so the transformation into the fixed
two-column table above is reversible):

```
| internal/imaging/orientation_test.go:129 (13 lines), internal/imaging/orientation_test.go:149 (13 lines) |
| internal/ui/grid_test.go:284 (20 lines), internal/ui/grid_test.go:331 (20 lines) |
| internal/ui/grid_test.go:284 (8 lines), internal/ui/grid_test.go:331 (8 lines), internal/ui/step_test.go:229 (10 lines) |
| internal/update/tufroot_repo_test.go:76 (8 lines), internal/update/tufroot_repo_test.go:84 (8 lines) |
| internal/clipboard/clipboard_test.go:13 (16 lines), internal/clipboard/copyfiles_test.go:31 (16 lines) |
| internal/ui/grid/dupes_test.go:603 (13 lines), internal/ui/grid/dupes_test.go:639 (13 lines) |
| internal/ui/exifwin/exifwin_test.go:584 (13 lines), internal/ui/exifwin/exifwin_test.go:623 (13 lines), internal/ui/exifwin/exifwin_test.go:646 (13 lines) |
| internal/ui/imgcache_test.go:87 (7 lines), internal/ui/imgcache_test.go:107 (7 lines) |
| internal/ui/exifwin/exifwin_test.go:751 (10 lines), internal/ui/exifwin/exifwin_test.go:784 (10 lines) |
| internal/imaging/raw_test.go:318 (12 lines), internal/imaging/raw_test.go:374 (12 lines) |
| internal/update/apply_test.go:17 (13 lines), internal/update/apply_test.go:154 (13 lines) |
| internal/imaging/raw_test.go:275 (15 lines), internal/imaging/raw_test.go:348 (15 lines), internal/imaging/raw_test.go:395 (15 lines) |
| internal/filescan/filescan_test.go:73 (25 lines), internal/ui/drop_test.go:288 (16 lines) |
| internal/update/attest_test.go:245 (12 lines), internal/update/attest_test.go:266 (12 lines) |
| internal/imaging/gif_test.go:360 (7 lines), internal/imaging/gif_test.go:401 (6 lines) |
| internal/ui/autoupdate_test.go:357 (11 lines), internal/ui/autoupdate_test.go:433 (13 lines) |
| internal/filepicker/filepicker_test.go:76 (13 lines), internal/filepicker/filepicker_test.go:119 (13 lines) |
| internal/update/tufroot_test.go:54 (10 lines), internal/update/tufroot_test.go:141 (10 lines) |
| internal/ui/favthumbs_test.go:63 (10 lines), internal/ui/favthumbs_test.go:102 (10 lines) |
| internal/ui/favorites/confirm_test.go:35 (19 lines), internal/ui/favorites/confirm_test.go:60 (19 lines) |
| internal/imaging/gif_test.go:200 (8 lines), internal/imaging/gif_test.go:250 (8 lines) |
| internal/ui/filestate_test.go:38 (17 lines), internal/ui/filestate_test.go:71 (17 lines) |
| internal/imaging/save_test.go:252 (11 lines), internal/imaging/save_test.go:297 (11 lines) |
| internal/ui/menu_test.go:495 (16 lines), internal/ui/menu_test.go:571 (16 lines) |
| internal/imaging/orientation_test.go:98 (22 lines), internal/imaging/orientation_test.go:184 (22 lines) |
| internal/ui/favorites/favorites_test.go:663 (14 lines), internal/ui/favorites/favorites_test.go:699 (14 lines) |
| internal/ui/slideshow_test.go:449 (16 lines), internal/ui/slideshow_test.go:495 (13 lines) |
| internal/favthumbs/store_test.go:184 (16 lines), internal/favthumbs/sweep_test.go:14 (16 lines) |
| internal/imaging/raw_test.go:348 (17 lines), internal/imaging/raw_test.go:395 (17 lines) |
| internal/ui/exifwin/exifwin_test.go:751 (8 lines), internal/ui/exifwin/exifwin_test.go:784 (8 lines), internal/ui/exifwin/exifwin_test.go:1061 (9 lines) |
| internal/favthumbs/sync_test.go:285 (10 lines), internal/favthumbs/sync_test.go:315 (10 lines) |
| internal/ui/slideshow_test.go:449 (14 lines), internal/ui/slideshow_test.go:475 (11 lines), internal/ui/slideshow_test.go:495 (11 lines) |
| internal/update/extract_test.go:186 (18 lines), internal/update/extract_test.go:213 (18 lines) |
```

The command originally used for this step is kept below, labelled as
defective, along with the reason it produced `null`:

```bash
# DEFECTIVE — produces "-null" for every fragment; do not use.
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | "| " + ([.locations[] | .physicalLocation.artifactLocation.uri + ":" + (.physicalLocation.region.startLine|tostring) + "-" + (.physicalLocation.region.endLine|tostring)] | join(" | ")) + " |"' "$SARIF"
```

This command produced the literal string `null` in place of an end line for
every fragment because `DuplicatedCode` region objects have no `endLine`
key at all (see the key inventory above) — `.physicalLocation.region.endLine`
evaluates to JSON `null` for all 33 results, and `tostring` renders that as
the text `null`.

Raw output of the defective command (one ragged row per cluster,
pipe-separated), kept here verbatim for the record:

```
| internal/imaging/orientation_test.go:129-null | internal/imaging/orientation_test.go:149-null |
| internal/ui/grid_test.go:284-null | internal/ui/grid_test.go:331-null |
| internal/ui/grid_test.go:284-null | internal/ui/grid_test.go:331-null | internal/ui/step_test.go:229-null |
| internal/update/tufroot_repo_test.go:76-null | internal/update/tufroot_repo_test.go:84-null |
| internal/clipboard/clipboard_test.go:13-null | internal/clipboard/copyfiles_test.go:31-null |
| internal/ui/grid/dupes_test.go:603-null | internal/ui/grid/dupes_test.go:639-null |
| internal/ui/exifwin/exifwin_test.go:584-null | internal/ui/exifwin/exifwin_test.go:623-null | internal/ui/exifwin/exifwin_test.go:646-null |
| internal/ui/imgcache_test.go:87-null | internal/ui/imgcache_test.go:107-null |
| internal/ui/exifwin/exifwin_test.go:751-null | internal/ui/exifwin/exifwin_test.go:784-null |
| internal/imaging/raw_test.go:318-null | internal/imaging/raw_test.go:374-null |
| internal/update/apply_test.go:17-null | internal/update/apply_test.go:154-null |
| internal/imaging/raw_test.go:275-null | internal/imaging/raw_test.go:348-null | internal/imaging/raw_test.go:395-null |
| internal/filescan/filescan_test.go:73-null | internal/ui/drop_test.go:288-null |
| internal/update/attest_test.go:245-null | internal/update/attest_test.go:266-null |
| internal/imaging/gif_test.go:360-null | internal/imaging/gif_test.go:401-null |
| internal/ui/autoupdate_test.go:357-null | internal/ui/autoupdate_test.go:433-null |
| internal/filepicker/filepicker_test.go:76-null | internal/filepicker/filepicker_test.go:119-null |
| internal/update/tufroot_test.go:54-null | internal/update/tufroot_test.go:141-null |
| internal/ui/favthumbs_test.go:63-null | internal/ui/favthumbs_test.go:102-null |
| internal/ui/favorites/confirm_test.go:35-null | internal/ui/favorites/confirm_test.go:60-null |
| internal/imaging/gif_test.go:200-null | internal/imaging/gif_test.go:250-null |
| internal/ui/filestate_test.go:38-null | internal/ui/filestate_test.go:71-null |
| internal/imaging/save_test.go:252-null | internal/imaging/save_test.go:297-null |
| internal/ui/menu_test.go:495-null | internal/ui/menu_test.go:571-null |
| internal/imaging/orientation_test.go:98-null | internal/imaging/orientation_test.go:184-null |
| internal/ui/favorites/favorites_test.go:663-null | internal/ui/favorites/favorites_test.go:699-null |
| internal/ui/slideshow_test.go:449-null | internal/ui/slideshow_test.go:495-null |
| internal/favthumbs/store_test.go:184-null | internal/favthumbs/sweep_test.go:14-null |
| internal/imaging/raw_test.go:348-null | internal/imaging/raw_test.go:395-null |
| internal/ui/exifwin/exifwin_test.go:751-null | internal/ui/exifwin/exifwin_test.go:784-null | internal/ui/exifwin/exifwin_test.go:1061-null |
| internal/favthumbs/sync_test.go:285-null | internal/favthumbs/sync_test.go:315-null |
| internal/ui/slideshow_test.go:449-null | internal/ui/slideshow_test.go:475-null | internal/ui/slideshow_test.go:495-null |
| internal/update/extract_test.go:186-null | internal/update/extract_test.go:213-null |
```

To determine how many clusters span more than one file (used above), each
result's fragments were reduced to their distinct file URIs:

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | [.locations[].physicalLocation.artifactLocation.uri] | unique | length' "$SARIF" | sort | uniq -c
```

Output:

```
  29 1
   4 2
```

(29 clusters touch exactly 1 distinct file, 4 clusters touch exactly 2 distinct
files; none touch 3 or more.)

Step 7 — confirm every fragment is in a test file:

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" | grep -cv '_test\.go$'
```

Output: `0`

Step 8 — record the effective profile:

```bash
jq -r '.. | objects | select(has("content")) | .content' "$SCRATCH/qodana/x/log/qodana-config.json" | grep -A2 '^profile:'
```

Output:

```
profile:
  name: qodana.starter
#Enable inspections
```

Cross-check against the repository's `qodana.yaml`:

```bash
cat /Users/ronin/Projects/picfetch/qodana.yaml
```

Relevant lines observed: the `profile:` / `name: qodana.starter` block, and,
uncommented at the end of the file:

```yaml
include:
  - name: DuplicatedCode
```

Supporting arithmetic for the `## Suppression accounting` section — the sum of
CSV counts for the five rule categories other than `DuplicatedCode` and
`GoTypeAssertionOnErrors`, and the set of distinct `ruleId` values present in
the SARIF results:

```bash
jq -r '[.runs[].results[].ruleId] | unique | .[]' "$SARIF"
awk -F';' 'NR>1 && $8+0>0 && $1!="DuplicatedCode" && $1!="GoTypeAssertionOnErrors" {s+=$8} END{print s}' "$SCRATCH/qodana/x/log/qodana_inspections_summary.csv"
```

Output:

```
DuplicatedCode
GoTypeAssertionOnErrors
12
```

## Fragment identity

A fragment is identified in this document by the triple
`uri:startLine:charOffset`, not by `uri:startLine` alone. Two different
fragments in the same file can share a `startLine` while differing in
`charOffset` (for example when the tool anchors a region on the line
containing an enclosing statement rather than the exact duplicated text);
including `charOffset` makes the identity exact — an exact match on all
three fields — rather than merely probable on a match of file and line
alone.

Commands used to establish the 71-slot / 63-fragment / 8-repeated finding
that underlies the rewritten `## Suppression accounting` paragraph:

```bash
jq -r '.runs[].results[]|select(.ruleId=="DuplicatedCode")|.locations[]|.physicalLocation|.artifactLocation.uri+":"+(.region.startLine|tostring)+":"+(.region.charOffset|tostring)' "$SARIF" | sort > /tmp/frag.txt
wc -l < /tmp/frag.txt
sort -u /tmp/frag.txt | wc -l
sort /tmp/frag.txt | uniq -d
```

Output:

```
71
63
internal/imaging/raw_test.go:348:10949
internal/imaging/raw_test.go:395:12195
internal/ui/exifwin/exifwin_test.go:751:20987
internal/ui/exifwin/exifwin_test.go:784:22072
internal/ui/grid_test.go:284:9102
internal/ui/grid_test.go:331:10785
internal/ui/slideshow_test.go:449:14310
internal/ui/slideshow_test.go:495:15662
```

This confirms 71 total location slots, 63 distinct fragments, and exactly 8
fragments each repeating. A further check (`sort /tmp/frag.txt | uniq -c |
sort -rn`) confirmed each of these 8 fragments appears exactly twice, never
more: 55 fragments appear once and 8 appear twice, and 55 × 1 + 8 × 2 = 71.

## GoLand comparison at 210fee5

This section compares the CI numbers above against a GoLand run over the same
commit. It was produced on 2026-08-29 through the `goland` MCP server against
the open project at `/Users/ronin/Projects/picfetch`.

### What was compared, and on what tree

`git rev-parse HEAD` returned `210fee54929de03fc0316025834874f965df2cd0`, the
same SHA as CI run `33270269940`. `git status --porcelain --untracked-files=no`
returned nothing, so no tracked file differed from that commit. The only
`git status --porcelain` entries were two untracked files, `plans/2026-08-29-qodana-duplication-close.md`
and `plans/2026-08-29-qodana-evidence.md` (both since archived to
`finished_refactorings/` under the same names by Task 6), both written by
this investigation and neither of them Go source. The IDE reported the same state independently:
`mcp__goland__git_status` returned branch `main`, 0 staged, 0 unstaged, and the
same 2 untracked files. So both sides analysed the same Go source tree.

`mcp__goland__get_project_modules` returned one module, `imagedrop`, and
`mcp__goland__get_repositories` returned one Git root at the project root. The
IDE module is named `imagedrop` while the directory is named `picfetch`; the
`git_status` agreement above is what establishes that they are the same tree.

### Which profile each side ran

CI ran the `qodana.starter` profile with `DuplicatedCode` re-enabled through
`include:` in `qodana.yaml`, as recorded in `## Effective profile` above. Its
dumped `log/effective.profile.xml` carries
`<inspection_tool class="DuplicatedCode" enabled="true" level="WEAK WARNING" enabled_by_default="true">`
with no `<option>` elements inside it, so the inspection ran at its default
settings (no changed minimum fragment size).

The IDE side ran through `mcp__goland__lint_files`, which uses the project's own
inspection profile. That profile is `.idea/inspectionProfiles/Project_Default.xml`,
whose entire body is:

```xml
<component name="InspectionProjectProfileManager">
  <profile version="1.0">
    <option name="myName" value="Project Default" />
    <inspection_tool class="GoStructLayout" enabled="true" level="WEAK WARNING" enabled_by_default="true" />
  </profile>
</component>
```

It names exactly one inspection, `GoStructLayout`, and says nothing about
`DuplicatedCode`; `DuplicatedCode` therefore ran at the IDE default, which is
enabled. This is confirmed by the run itself, which returned both
`Duplicated code fragment (...)` problems and `Struct '...' might be suboptimal`
(`GoStructLayout`) problems.

The two profiles are therefore not identical as inspection *sets* — the IDE side
returned findings from inspections CI did not report, such as `GoStructLayout`
and "HTTP links are not secure" — but for the one inspection under comparison,
`DuplicatedCode`, both sides ran it enabled and at default settings. Only
`DuplicatedCode` findings are counted below; every other inspection returned by
the IDE was discarded before counting.

### The IDE run

`mcp__goland__lint_files` was called with `min_severity: "warning"` over all 326
tracked `.go` files, in 9 batches of 40 (the last batch held 6). No response
carried `more: true`, and no file entry carried `timedOut: true` or a
`notAnalyzedReason`, so all 326 files were analysed. Although `min_severity` was
`warning`, the tool returned `WEAK WARNING` problems as well, which is what
`DuplicatedCode` is reported at.

Duplication detection is project-wide rather than batch-local. This was checked
before batching by calling `lint_files` with the single file
`internal/filescan/filescan_test.go`; it returned
`Duplicated code fragment (25 lines long)` at line 73, whose only partner in the
CI SARIF is `internal/ui/drop_test.go:288` (cluster 13 above), a file that was
not in that call. So a fragment is reported even when the other half of its
cluster is outside the batch.

Each returned problem carries the fields `severity`, `description`, `lineText`,
`line` and `column`. There is no `endLine` field and no cluster or group
identifier. The IDE surface therefore counts in fragments, not clusters, and no
IDE cluster count is obtainable from it. The raw results are kept at
`$SCRATCH/goland-dup-210fee5.txt`.

### Both counts, with their units

- **IDE: 71 `DuplicatedCode` problems**, unit = fragments. One problem per
  duplicated fragment. The 71 problems sit at 71 distinct `file:startLine`
  positions — no position is reported twice — so the IDE's fragment count and its
  distinct-position count are both 71.
- **CI: 63 distinct fragments**, unit = fragments, from the 33 SARIF cluster
  results occupying 71 location slots, against 75 rows in the CSV inspection
  summary. These four numbers are the ones established earlier in this document.

The IDE's 71 fragments and CI's 71 location slots are equal but are not the same
quantity. CI's 71 is a slot count in which 8 fragments are counted twice because
they belong to 2 clusters each. The IDE reports each such position only once: for
example `internal/ui/grid_test.go:284` appears in CI clusters 2 and 3, and the IDE
returns a single problem there, `Duplicated code fragment (20 lines long)`. The
IDE's 71 is therefore comparable to CI's 63, not to CI's 71, and the equality is
a coincidence.

### The fragment-set difference

Both sides were reduced to the key `uri:startLine` and compared with `comm`. The
result:

- fragments in CI but not in the IDE: **0**
- fragments in the IDE but not in CI: **8**
- fragments in both: **63**

CI's 63 fragments are a strict subset of the IDE's 71. CI reports nothing the IDE
does not. The 8 IDE-only fragments are, in full:

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

They live in exactly 2 files: `internal/imaging/loader_test.go` (7 fragments) and
`internal/update/tufroot_test.go` (1 fragment).

A caveat on the key. `uri:startLine` assumes both tools anchor a fragment on the
same line, and the log evidence below shows two cases where CI anchored one line
away from where the IDE anchored. For all 63 fragments that both sides did emit,
the keys matched exactly, so the anchoring agreed everywhere both tools produced
output; the disagreement appears only among fragments CI failed to emit.

### Why the 8 are missing from CI: the CI run logged the failure itself

`internal/imaging/loader_test.go` does not appear anywhere in
`qodana.sarif.json` (`grep -c 'loader_test' "$SARIF"` returned 0).
`internal/update/tufroot_test.go` appears twice, at lines 54 and 141, which is
cluster 18 above; line 173 is absent.

The run's own `log/idea.log` contains exactly 3 warnings from the
`DuplicatesProblem` component, and no other warning from that component:

```
2026-08-29 19:14:27,202 WARN - #o.j.q.s.i.r.g.DuplicatesProblem - Can't find duplicate problem in db, file://$PROJECT_DIR$/internal/imaging/loader_test.go:405:11963
2026-08-29 19:14:27,202 WARN - #o.j.q.s.i.r.g.DuplicatesProblem - Can't find duplicate problem in db, file://$PROJECT_DIR$/internal/imaging/loader_test.go:334:9850
2026-08-29 19:14:27,286 WARN - #o.j.q.s.i.r.g.DuplicatesProblem - Can't find duplicate problem in db, file://$PROJECT_DIR$/internal/update/tufroot_test.go:143:3919
```

They name exactly the 2 files that hold the 8 IDE-only fragments, and no other
file. The trailing numbers are a 1-based line number and a 0-based character
offset, which was checked by counting characters in the files: the characters
before line 334 of `loader_test.go` number 9848, and 9848 + 2 leading tabs = 9850;
the characters before line 405 number 11961, and 11961 + 2 = 11963. The lines
themselves are `loader_test.go:334` = `if err != nil {`,
`loader_test.go:405` = `loaded, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes)`,
and `tufroot_test.go:143` = `srv := repo.serve()`. The IDE anchors the same
duplicated regions at `loader_test.go:333` (`loaded, err := LoadImage(...)`),
`loader_test.go:406` (`if err != nil {`) and `tufroot_test.go:173`
(`srv := repo.serve()`) — the same repeating two-line pattern, anchored one line
apart.

The warnings are emitted at 19:14:27, immediately after the line
`The Project analysis stage completed in 41s`, that is during result
serialisation and after detection. So CI detected these duplicates and then
failed to write them out.

The loss is upstream of the SARIF, not introduced by it. The HTML report's own
problem list, `report/results/result-allProblems.json`, holds 34 problems: 33 of
type `Duplicated code fragment` and 1 of type
`Type assertion on errors fails on wrapped errors`. Its 33 duplication problems
carry 71 `sources` entries reducing to 63 distinct `path:line` fragments — the
same 33 / 71 / 63 as the SARIF. Nothing is dropped between the report and the
SARIF; the drop happens where the log warns.

### The 75 − 63 = 12 gap

The 8 IDE-only fragments account for 8 of the 12. 63 + 8 = 71, and the CSV counts
75, so 4 of the 12 are not accounted for by anything measured here. The IDE run
does not supply a 9th, 10th, 11th or 12th fragment: its total is 71 problems, and
all 71 are listed or matched above. No hypothesis is offered for the remaining 4.

> **Superseded 2026-08-29:** see the note under `## Suppression accounting`
> above — the remaining 4 are now explained.

Two facts bear on that residual without settling it. First, the CSV column is
headed "Problems Count", and it is not established here whether it counts distinct
fragments or fragment-in-cluster slots; under CI's own slot accounting the 33
emitted clusters already occupy 71 slots rather than 63. Second, the IDE reports
one problem per anchor position even when that position belongs to 2 clusters, so
the IDE cannot distinguish a fragment in 1 cluster from the same fragment in 2.
Neither observation was pushed to a conclusion.

### Verdict

At commit 210fee5 the IDE finds 71 `DuplicatedCode` fragments and CI finds 63,
CI's 63 are a strict subset of the IDE's 71, and the 8 fragments CI is missing —
7 in `internal/imaging/loader_test.go` and 1 in `internal/update/tufroot_test.go` —
are accounted for by a genuine Qodana serialisation failure that the CI run
recorded itself as 3 `DuplicatesProblem` "Can't find duplicate problem in db"
warnings naming exactly those 2 files and no others; the counting unit and the
`qodana.starter`-versus-Project-Default profile difference explain the shape of
the other numbers but explain none of the 8, and 4 of the 12-fragment CSV-to-SARIF
gap remain unexplained by any measurement taken here.

The historical figure of 90 from `todos.md` is not reproduced and was not
reproducible here: it was taken at `e9cfe7b`, which is 27 commits behind
`210fee5` (`git rev-list --count e9cfe7b..210fee5` returned 27), and that range
includes commits whose stated purpose was removing duplication — `795aa80`
("deduplicate shared image test fixtures"), `32de323` ("suppress honest
duplication in orientation pixel loops"), `fda5f54` ("extract JPEG
header-segment walker from the four duplicated loops") and `4f3b5b7`/`cd793ae`
(the settings numeric-entry extraction). Re-measuring the 90 would require
checking out `e9cfe7b`, which was out of scope here.

## Minimal reproduction

A specific fragment pair the IDE reports and CI does not:
`internal/imaging/loader_test.go:333` and `internal/imaging/loader_test.go:351`.
Both are reported by the IDE as `Duplicated code fragment (13 lines long)`,
anchored on the line
`		loaded, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes)`.
Neither line appears anywhere in `qodana.sarif.json`; the string `loader_test`
does not occur in that file at all.

To reproduce the IDE side:

```
mcp__goland__lint_files(
  projectPath = "/Users/ronin/Projects/picfetch",
  files       = ["internal/imaging/loader_test.go", "internal/update/tufroot_test.go"],
  min_severity= "warning")
```

which returns 7 `Duplicated code fragment` problems in `loader_test.go` at lines
333, 351, 387, 406, 423, 441 and 459, and 3 in `tufroot_test.go` at lines 54, 141
and 173.

To reproduce the CI side:

```bash
SARIF="$SCRATCH/qodana/x/qodana.sarif.json"
grep -c 'loader_test' "$SARIF"                       # 0
jq -r '.runs[].results[]|select(.ruleId=="DuplicatedCode")|.locations[]
       |.physicalLocation.artifactLocation.uri+":"
        +(.physicalLocation.region.startLine|tostring)' "$SARIF" \
  | sort -u | grep 'tufroot_test'                    # only :54 and :141, never :173
grep "Can't find duplicate problem in db" "$SCRATCH/qodana/x/log/idea.log"
```

## Reproduction of the GoLand comparison

Step 1 — confirm the tree:

```bash
git rev-parse HEAD                         # 210fee54929de03fc0316025834874f965df2cd0
git status --porcelain --untracked-files=no  # empty
```

Step 2 — confirm the MCP server sees the project: `mcp__goland__get_project_modules`
returned `{"modules":[{"name":"imagedrop","type":"WEB_MODULE"}]}` and
`mcp__goland__get_repositories` returned `{"roots":[{"pathRelativeToProject":"","vcsName":"Git"}]}`.

Step 3 — file counts:

```bash
git ls-files '*.go' | wc -l                  # 326
git ls-files '*.go' | grep -c '_test\.go$'   # 155
git ls-files '*.go' | grep -vc '_test\.go$'  # 171
git ls-files | wc -l                         # 477
```

The CSV's "Times Inspection was Performed" column relates to these as follows,
and does not divide evenly into any of them. Grouping the CSV's 95 inspection
rows by language and by that column:

```bash
awk -F';' 'NR>1 {print $4"\t"$9}' "$SCRATCH/qodana/x/log/qodana_inspections_summary.csv" \
  | sort | uniq -c | sort -rn
```

```
  76 go	306
  10 RegExp	7
   5 	413
   1 go	361
   1 RegExp	3
   1 	46
   1 	1
```

So 413 is not a figure specific to `DuplicatedCode`: it is shared by all 5
inspections that declare no language (`HardcodedPasswords`, `DuplicatedCode`,
`VulnerableLibrariesLocal`, `MaliciousLibrariesLocal`, `CustomRegExpInspection`),
and 306 is shared by 76 of the 77 Go-language inspections (the 77th,
`GoCoverageInspection`, ran 361 times).

Against the file counts: 306 is 20 fewer than the 326 tracked `.go` files, and is
neither the 171 non-test nor the 155 test count. 413 is 87 more than the 326 `.go`
files and 64 fewer than the 477 tracked files of all types; 413 / 326 = 1.267, not
an integer. None of these numbers divides evenly into another, and no command run
here identifies which files make up the 306 or the 413. They are recorded as
observed and not reconciled.

Step 4 — the IDE run, in 9 batches:

```
mcp__goland__lint_files(projectPath="/Users/ronin/Projects/picfetch",
                        files=<40 paths from `git ls-files '*.go' | sort`>,
                        min_severity="warning", timeout=300000)
```

Raw results kept at `$SCRATCH/goland-dup-210fee5.txt` (71 rows, one per
`Duplicated code fragment` problem, as `file:startLine<TAB>lines-long`).

Step 5 — count in both units:

```bash
grep -vc '^#' "$SCRATCH/goland-dup-210fee5.txt"                 # 71 problems (fragments)
grep -v '^#' "$SCRATCH/goland-dup-210fee5.txt" | cut -f1 | sort -u | wc -l   # 71 distinct file:startLine
grep -v '^#' "$SCRATCH/goland-dup-210fee5.txt" | cut -f1 | sort | uniq -d    # empty
```

No cluster count is available from this surface: the returned problems carry no
group identifier.

Step 6 — diff the fragment sets:

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | .locations[]
       | .physicalLocation.artifactLocation.uri + ":"
         + (.physicalLocation.region.startLine|tostring)' "$SARIF" \
  | sort -u > "$SCRATCH/ci-frag.txt"                            # 63 lines
grep -v '^#' "$SCRATCH/goland-dup-210fee5.txt" | cut -f1 | sort -u > "$SCRATCH/ide-frag-u.txt"  # 71 lines
comm -23 "$SCRATCH/ci-frag.txt" "$SCRATCH/ide-frag-u.txt" | wc -l   # 0
comm -13 "$SCRATCH/ci-frag.txt" "$SCRATCH/ide-frag-u.txt" | wc -l   # 8
comm -12 "$SCRATCH/ci-frag.txt" "$SCRATCH/ide-frag-u.txt" | wc -l   # 63
```

Step 7 — corroborate against the run's own report and log:

```bash
AP="$SCRATCH/qodana/x/report/results/result-allProblems.json"
jq -r '.listProblem | length' "$AP"                                              # 34
jq -r '[.listProblem[].type] | group_by(.) | map("\(length)\t\(.[0])") | .[]' "$AP"
# 33	Duplicated code fragment
# 1	Type assertion on errors fails on wrapped errors
jq -r '[.listProblem[] | select(.type=="Duplicated code fragment") | .sources | length] | add' "$AP"  # 71
jq -r '.listProblem[] | select(.type=="Duplicated code fragment") | .sources[]
       | .path + ":" + (.line|tostring)' "$AP" | sort -u | wc -l                 # 63
grep -c "Can't find duplicate problem in db" "$SCRATCH/qodana/x/log/idea.log"    # 3
```


## Exclusion fallback

Task 4 added an `exclude:` block to `qodana.yaml`:

```yaml
exclude:
  - name: DuplicatedCode
    paths:
      - "**/*_test.go"
```

Whether the `2026.2` Go linter honors a per-file glob (`**/*_test.go`) inside
`exclude: paths:` was not established by anything checked before Task 5 ran.
JetBrains' own `qodana.yaml` reference documents that key with directory and
literal-file examples only:

```yaml
exclude:
  - name: All
    paths:
      - asm-test/src/main/java/org
      - asm/Visitor.java
      - benchmarks
```

(https://www.jetbrains.com/help/qodana/qodana-yaml.html) — no glob appears in
that example. A JetBrains maintainer (`brichbash`) stated in the project's own
GitHub discussions that the sibling `patterns:` glob key under `exclude:` is
"a known issue," and directed users to YAML profiles (`ignore:` under a custom
profile document) instead:
https://github.com/JetBrains/Qodana/discussions/259#discussioncomment-9105835.
That confirmed report is about `patterns:`, not `paths:`, so it does not prove
`paths:` globs fail — but it corroborates the brief's caution rather than
weighing against it. Two CI runs, below, tried two different glob dialects.
Both compiled into a real, disabled scope and matched nothing. The current
`qodana.yaml` no longer uses a glob at all — see "Explicit file list applied"
below.

### CI run 33273666731 at d38f6e8: the glob was parsed but matched nothing

Run `33273666731` at commit `d38f6e8` (confirmed via the artifact's own
`revisionId`, `d38f6e84ae81f2d3ec36db54e02fd2442446317b`) completed
successfully. Its SARIF still held **33 `DuplicatedCode` results**, and every
fragment in them was still inside a `_test.go` file — **0** of their
locations fall outside `_test.go`:

```bash
SARIF="$SCRATCH/q2/x/qodana.sarif.json"
jq -r '[.runs[].results[] | select(.ruleId=="DuplicatedCode")] | length' "$SARIF"          # 33
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode")
       | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" \
  | sort -u | grep -cv '_test\.go$'                                                         # 0
```

So `"**/*_test.go"` did not exclude anything: the count is unchanged from the
pre-exclusion baseline and the fragment set is still exactly the 63-location,
33-result, all-test-file shape established earlier in this document. Read on
its own, that would be consistent with "the exclusion was never applied at
all" — but the run's artifact also shows that is not what happened. The
mechanism engaged; only the pattern failed to match any file. In
`log/effective.profile.xml`, nested inside the `DuplicatedCode` inspection
element itself, sits a compiled scope built from the `qodana.yaml` exclusion:

```
<scope name="qodana.yaml.exclude.DuplicatedCode" level="INFORMATION" enabled="false" />
```

That scope's existence — named after the `exclude:` entry, sitting inside the
`DuplicatedCode` inspection's own XML element — is direct evidence that the
`exclude:` key, its `name:`, and its `paths:` list all reached the Qodana
engine and were compiled into a real (if inert) scope. The `exclude:`
mechanism is sound. Only the glob pattern inside it failed to match any file.

### Why the first glob didn't match: wrong glob dialect

The same `effective.profile.xml` also lists the built-in scopes Qodana ships
with, alongside the compiled `qodana.yaml.exclude.*` ones:

```bash
grep -o 'scope name="glob:[^"]*"' "$SCRATCH/q2/x/log/effective.profile.xml" | sort -u
```
```
scope name="glob:**.md"
scope name="glob:**.test.ts"
scope name="glob:**/node_modules/**"
scope name="glob:.qodana/**"
scope name="glob:build/**"
scope name="glob:buildSrc/**"
scope name="glob:builds/**"
scope name="glob:dist/**"
scope name="glob:scope#test:*..*"
scope name="glob:tests/**"
scope name="glob:tools/**"
scope name="glob:vendor/**"
```

JetBrains' own built-in exclusion for TypeScript spec files is written
`glob:**.test.ts`, not `glob:**/*.test.ts`. In this dialect `**` already
crosses path separators on its own, so a leading `**/` before the final
segment is redundant at best and, per this run, actually prevents a match at
the repository root or single-segment paths — consistent with `"**/*_test.go"`
matching zero of the 29+ `_test.go` files in this repository despite every
one of them being a real target. The Go analogue of `glob:**.test.ts` is
`**_test.go`, not `**/*_test.go`. For that reason, `qodana.yaml`'s `paths:`
entry was changed from `"**/*_test.go"` to `"**_test.go"` in the next round —
see immediately below for what that run showed.

### CI run 33274030031 at f481c4a: the corrected dialect also matched nothing

Run `33274030031` at commit `f481c4a` (confirmed via the artifact's own
`revisionId`, `f481c4a7847f1484d148fc4353fae80019d4c404`) also completed
successfully, with `qodana.yaml`'s `paths:` entry reading `"**_test.go"`.
Its SARIF again held **33 `DuplicatedCode` results**, again with **0** of
their locations outside `_test.go`:

```bash
SARIF="$SCRATCH/q3/x/qodana.sarif.json"
jq -r '[.runs[].results[] | select(.ruleId=="DuplicatedCode")] | length' "$SARIF"          # 33
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode")
       | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" \
  | sort -u | grep -cv '_test\.go$'                                                         # 0
```

Three pieces of evidence from this run's own artifact confirm the corrected
config genuinely reached CI — this is not a "the config did not arrive"
failure:

1. `log/qodana-config.json` echoes the live `qodana.yaml` content verbatim.
   Its embedded config text ends:
   ```
   exclude:
     - name: DuplicatedCode
       paths:
         - "**_test.go"
   ```
   (extracted with
   `jq -r '.. | objects | select(has("content")) | .content' log/qodana-config.json`)
   — the exact pattern that was set for this run, not a stale one.
2. `log/effective.profile.xml` again carries the compiled
   `<scope name="qodana.yaml.exclude.DuplicatedCode" level="INFORMATION"
   enabled="false" />` nested inside the `DuplicatedCode` inspection element,
   exactly as it did for the first glob attempt — proof the `exclude:` entry
   was parsed and compiled a second time, with the new pattern.
3. `grep -icE "cache (hit|restored)" log/idea.log` returns **0** — no stale
   cache was reused for this run, so a cached pre-exclusion result set
   cannot explain the unchanged count.

Two glob dialects have now each compiled into a disabled scope and
suppressed nothing. Guessing at a third glob spelling is not the next step.

### Rejected approach: directory-level exclusion

The brief's original fallback was to replace the glob with the directories
that contain flagged test files, derived from Task 1's cluster table with:

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode")
       | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" \
  | xargs -n1 dirname | sort -u
```

Run against the SARIF captured earlier in this file
(`$SCRATCH/qodana/x/qodana.sarif.json`, i.e.
`/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad/qodana/x/qodana.sarif.json`)
this is a **measured observation, not something applied**: it returns 10
distinct directories, not the 22 the brief named (that figure was invented,
not measured, and is withdrawn):

```
internal/clipboard
internal/favthumbs
internal/filepicker
internal/filescan
internal/imaging
internal/ui
internal/ui/exifwin
internal/ui/favorites
internal/ui/grid
internal/update
```

This approach is rejected outright, not merely deprioritized. All 10 of these
directories also contain production (non-`_test.go`) Go source — confirmed
with `ls <dir>/*.go | grep -v _test`, e.g. `internal/imaging` alone holds 17
non-test files including `loader.go` and `orientation.go`, and `internal/ui`
holds 38. Excluding a directory to silence one flagged test file inside it
would also suppress `DuplicatedCode` reporting for every production file in
that same directory — the exact failure mode this task exists to prevent
(see the brief's "Binding global constraints": deleting the `include:` block
would silently disable duplication reporting everywhere; a directory-level
`exclude:` would silently disable it in the 10 directories that most need
it, while looking like a narrow, reasonable fix at the moment a glob had just
failed). A directory-level fallback is therefore not used under any
circumstance, regardless of which glob dialect is at fault.

### Explicit file list: applied this round

With two glob dialects each ruled out by a CI run rather than by inspection,
`qodana.yaml`'s `paths:` entry was replaced with the explicit list of
**files** (not directories) that contain flagged test files — the form
JetBrains actually documents (see the directory-path example quoted at the
top of this section). The list was derived, in an earlier round of this
document, with:

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode")
       | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" | sort -u
```

against the 210fee5-commit SARIF used throughout this document, which
returns 29 distinct files, to which `internal/imaging/loader_test.go` was
added explicitly (see below for why), for **30 files** total. That 30-file
list is now the live content of `qodana.yaml`'s `paths:`, taken from this
section rather than re-derived, and confirmed by counting the entries
written:

```bash
ruby -ryaml -e 'puts YAML.load_file("qodana.yaml")["exclude"][0]["paths"].length'   # 30
```

The 30 files:

```
internal/clipboard/clipboard_test.go
internal/clipboard/copyfiles_test.go
internal/favthumbs/store_test.go
internal/favthumbs/sweep_test.go
internal/favthumbs/sync_test.go
internal/filepicker/filepicker_test.go
internal/filescan/filescan_test.go
internal/imaging/gif_test.go
internal/imaging/loader_test.go
internal/imaging/orientation_test.go
internal/imaging/raw_test.go
internal/imaging/save_test.go
internal/ui/autoupdate_test.go
internal/ui/drop_test.go
internal/ui/exifwin/exifwin_test.go
internal/ui/favorites/confirm_test.go
internal/ui/favorites/favorites_test.go
internal/ui/favthumbs_test.go
internal/ui/filestate_test.go
internal/ui/grid/dupes_test.go
internal/ui/grid_test.go
internal/ui/imgcache_test.go
internal/ui/menu_test.go
internal/ui/slideshow_test.go
internal/ui/step_test.go
internal/update/apply_test.go
internal/update/attest_test.go
internal/update/extract_test.go
internal/update/tufroot_repo_test.go
internal/update/tufroot_test.go
```

Why `internal/imaging/loader_test.go` is on the list despite never appearing
in any SARIF produced so far: `## GoLand comparison at 210fee5` above
established that the IDE detects 71 `DuplicatedCode` fragments at commit
210fee5 while CI's `qodana.sarif.json` holds only 63, and that the run's own
`log/idea.log` logs 3 `Can't find duplicate problem in db` warnings — a
serialisation failure, not a detection failure — naming exactly the 2 files
that hold the missing 8 fragments: `internal/imaging/loader_test.go` (7
fragments; `grep -c 'loader_test' "$SARIF"` returns 0 against that SARIF —
the file never appears there at all) and `internal/update/tufroot_test.go`
(1 fragment, at line 173; that file is on the list anyway because it also
has 2 other fragments CI did write out, cluster 18). So `loader_test.go`'s
absence from every SARIF seen so far is not evidence it is duplication-free;
it is on the list because a future Qodana release that fixes that
serialisation defect could otherwise start reporting exactly those 7
fragments unopposed.

### Stopping condition

If the next CI run still reports `DuplicatedCode` results with this 30-file
list in place, the conclusion is not "try a fourth configuration attempt."
Two different `paths:` spellings have each been independently confirmed, by
that run's own artifact, to reach the engine and compile into a real,
disabled `qodana.yaml.exclude.DuplicatedCode` scope nested in the
`DuplicatedCode` inspection — and each still left the count and the fragment
set completely unchanged. A third failure under those same conditions —
config confirmed present in `qodana-config.json`, scope confirmed compiled
in `effective.profile.xml`, no cache reuse in `idea.log` — would mean
`exclude:` does not apply to the `DuplicatedCode` inspection at all in this
linter version, regardless of what `paths:` contains. That would become a
second finding for the upstream bug report (alongside the `Can't find
duplicate problem in db` serialisation defect already being reported per
Task 2), not a fourth glob or list variant to try. This is written here as
the stopping condition to apply if that happens, not as a prediction that it
will.

## Confirmation run

The `### Stopping condition` above was written before this run and was not
reached.

Run `33274422606` at commit `ed3d4e6` (full SHA
`ed3d4e6cd40645dea8b97a1c95c4baf5e7dd1800`, confirmed via
`gh run view 33274422606 --json headSha,conclusion,event`, which returned
`{"conclusion":"success","event":"push","headSha":"ed3d4e6cd40645dea8b97a1c95c4baf5e7dd1800"}`)
completed with `conclusion: success` and returned **0** SARIF results:

```bash
SP=/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad
jq '[.runs[].results[]]|length' "$SP/q4/x/qodana.sarif.json"
```

Output: `0`.

The summary CSV's `DuplicatedCode` count fell from 75 to **4**, not to 0:

```bash
awk -F';' '$1=="DuplicatedCode" {print $8}' "$SP/q4/x/log/qodana_inspections_summary.csv"
```

Output: `4`.

That the CSV count fell to 4 rather than to 0 is the proof the exclusion
narrowed the inspection's scope rather than disabling the inspection
outright — a disabled inspection would have counted 0 in the CSV as well as
in the SARIF. The remaining 4 are the production fragments in the
orientation pixel loops (`internal/imaging/orientation.go`) that carry
source-local `//goland:noinspection DuplicatedCode` suppressions, which is
why they are counted in the pre-suppression CSV but never reach the SARIF.
