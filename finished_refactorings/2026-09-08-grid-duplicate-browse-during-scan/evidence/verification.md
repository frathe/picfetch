# Grid duplicate browsing: verification

Date: 2026-09-09
Status: complete; every acceptance criterion passes

## Red/green

- [Original regression](red.log): with three held sources, `known_group`
  fails for hide off and on because all eight file identities appear instead
  of the highlighted source's two copies.
- [Next vertical slice](updates-red.log): entry is corrected, but the newly
  decoded matching member does not join while other source reads remain held.
- [Every exact AC1–AC7 test command](final-ac.log) passes on the restored final
  implementation. Verbose output confirms all five required root subtests run.
- [Complete grid package](grid-full-green.log) passes.
- [Focused race regression](race-green.log) passes every behavioral case,
  including both Return and click opening of a hidden copy (15.500s native).

The fixture constructs the viewer/pools inside synctest, admits hashing before
opening Grid, and holds three ReaderURI sources independently. It drains queued
callbacks after synctest quiescence without settling held work. Fake-clock
advancement models the existing hash-notification throttle; callback delivery
is observed separately. Cleanup releases every reader before stopping and
settling the feature. Opening a copy uses the viewer's normal neighbor preload
to avoid the Fyne test driver's inline completion racing GridWrap's deferred
unselect. No new production test interface was introduced.

## Negative guard checks

Each temporary mutation was restored before proceeding:

| Deliberate violation | Observed guard | Evidence |
| --- | --- | --- |
| Restore suppression of partial hash notifications during browse | `group_updates` reports no queued grouping result | [log](negative-hash_progress.log) |
| Disable browse cancellation | Exit cases report late delivery retaining browse | [log](negative-cancel.log) |
| Omit source identity remapping | Reorder results contain the wrong URI identities | [log](negative-identity.log) |
| Wait for scan completion before partial group dissolution | Sensitivity change leaves browse active | [log](negative-dissolution.log) |
| Omit inspect admission when opening a variant | Return and click fail to retain the selected hidden copy in inspect | [log](negative-inspect.log) |

## Manuals and repository checks

The English/German diffs both describe immediate known-group entry, continuing
analysis and membership updates, preservation while an update is pending, and
the unchanged unknown-group wait. Existing title/badge and exit documentation
is retained. [Embedding and rendering guards](manual-ac.log) pass.

[Linux shard validation](shards.log) passes: 679 runnable root tests in three
shards, with `TestGridBrowseDuringAnalysis` added to ui-1. The existing
`grid_test.go` Qodana exclusion already applies.

Canonical command (exit 0, one full run):

```sh
make verify 'TEST_CAPTURE=/work/.scratch/grid-duplicate-browse-during-scan/evidence/race-$(TEST_PARTITION).json'
```

The standard four race partitions, memory budget, timeout, and race flags
remained unchanged. The attempted capture override was not forwarded into
Docker's inner Make invocation: the console log shows its default
`/tmp/picfetch-test-{non-ui,ui-1,ui-2,ui-3}.json` paths. Those raw JSON files
were removed with the container. The complete console output is retained;
raw JSON is unavailable. The gate was not repeated solely for logging.

[Complete make verify output](verify.log) confirms formatting, TUF/Qodana,
vet, build and the four race partitions pass. The new Linux race regression
passes in 18.980s. Root UI shards 1, 2, and 3 finish successfully in 318.452s,
281.198s, and 274.134s respectively. No golden was regenerated.
