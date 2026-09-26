# Local inspection after disabling Qodana CI

Checked against JetBrains documentation on 2026-09-25. Ronin authorized disabling
Qodana CI because the trial subscription ended. GitHub reports workflow
`qodana_code_quality.yml` as `disabled_manually`; its YAML, `qodana.yaml` and
analysis build-tag configuration are retained. No credentials were changed.
The failed scan did not produce a usable analysis report. This is a waived CI
gate, not a clean Qodana result. CodeQL, tests and GoLand inspections remain.

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
because their plugin configurations differ. This repository's CI used
`qodana.starter` and `qodana.yaml` exclusions; GoLand's default profile is not
evidence of equivalent coverage. The documented local Qodana path has not yet
been exercised here after subscription expiry.
[IDE/standalone distinction](https://www.jetbrains.com/help/qodana/quick-start.html).

After deliberately restoring an eligible subscription/token, the workflow can
be re-enabled with `gh workflow enable qodana_code_quality.yml`. Review a fresh
post-suppression SARIF result before claiming that gate passes again.
