# Local inspection and Qodana CI

## Current status: CI restored on 2026-09-26

Ronin renewed the trial and authorized re-enabling Qodana CI on 2026-09-26.
The workflow is active; PR 63 references the renewed GitHub secret while
preserving the analysis profile, build tags, exact exclusions and permissions.
The earlier paused-CI waiver is withdrawn. Require a fresh completed scan and
review its post-suppression `qodana.sarif.json` before calling this gate clean.
A licensing failure or incomplete scan remains unverified, not waived or passed.
The local options below are fallback inspection tools, not a replacement for
the restored CI gate. CodeQL and tests remain required too.

## Historical pause: 2026-09-25

The local options were checked against JetBrains documentation on 2026-09-25.
On that date, Ronin authorized disabling Qodana CI because the trial had ended.
GitHub then reported `qodana_code_quality.yml` as `disabled_manually`; its YAML,
`qodana.yaml` and analysis build-tag configuration were retained. No credentials
were changed as part of that pause. The failed scan produced no usable report.
The gate was explicitly waived for that pause only, not counted as a clean
result. This paragraph records history, not the current verification policy.

## Local options

Qodana inside GoLand is distinct from the standalone scanner: JetBrains says
running it inside the IDE needs no separate Qodana license. Open **Problems ->
Qodana -> Try locally**, or **Tools -> Qodana -> Try Code Analysis with Qodana**.
Keep **Send analysis results to Qodana Cloud** unchecked for a local-only run.
Keep the existing project configuration rather than overwriting it in the dialog.
[Qodana quick start](https://www.jetbrains.com/help/qodana/quick-start.html),
[IDE integration](https://www.jetbrains.com/help/qodana/qodana-ide-plugin.html).

The standalone Go linter, whether in local Docker/native mode or CI, requires a
Qodana project token for license verification. Go is not included in Community.
Moving the failed CI command onto a laptop does not remove that requirement.
[Qodana for Go](https://www.jetbrains.com/help/qodana/golang.html),
[edition matrix](https://www.jetbrains.com/help/qodana/pricing.html).

GoLand's own **Code -> Inspect Code...** also runs the enabled inspections for
a chosen scope and profile, including batch-only checks that editor highlighting
does not run. It is separate from the Qodana tool window.
[GoLand inspections](https://www.jetbrains.com/help/go/running-inspections.html).

For automation, GoLand documents the macOS command:

```text
GoLand.app/Contents/bin/inspect.sh <project> <inspection-profile.xml> <output> -format json
```

Use the actual installed application path and an exported XML profile. Project
profiles normally live in `.idea/inspectionProfiles`. This starts GoLand in the
background and cannot run alongside an already-running GoLand instance; use
the current IDE session's inspections while developing. XML, JSON and plain-text
reports are supported. No headless scan was attempted during this session.
[GoLand command-line inspector](https://www.jetbrains.com/help/go/command-line-code-inspector.html).

## Limits and restoration

Historical checks support keeping the JetBrains inspection engine rather than
building a second analyzer. The [August comparison](../finished_refactorings/2026-08-29-qodana-evidence.md)
records wrapped-error assertions, nil safety, redundant boolean/type expressions,
error-string style and duplicate fragments. The post-suppression findings were
one wrapped-error assertion and 33 test-only duplicate clusters; the other
categories were already intentionally suppressed, not outstanding defects.
The successful full scan in [run 36020763631](https://github.com/frathe/picfetch/actions/runs/36020763631)
has zero post-suppression SARIF findings. Its pre-suppression CSV still lists data
flow, boolean simplification, redundant conversion, duplicate code, error-string
style and omitted-type advice. Preserve those distinctions and existing justified
suppressions when selecting a local profile.

JetBrains explicitly warns that IDE and standalone Qodana results can differ
because their plugin configurations differ. This repository's CI uses
`qodana.starter` and `qodana.yaml` exclusions; GoLand's default profile is not
evidence of equivalent coverage. The documented local Qodana path has not yet
been exercised here after subscription expiry.
[IDE/standalone distinction](https://www.jetbrains.com/help/qodana/quick-start.html).

For a future owner-authorized pause, retain the configuration and use the local
options above. After the owner authorizes restoration with an eligible
subscription/token, re-enable with `gh workflow enable qodana_code_quality.yml`.
This restoration already happened on 2026-09-26; do not repeat it merely because
you are following this guide. Always review a fresh post-suppression SARIF result
before claiming that the restored gate passes.
