# Issue tracker: Local Markdown

Issues and specs for this repository live as Markdown files in `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`
- The specification is `.scratch/<feature-slug>/spec.md`
- Implementation issues are separate files under
  `.scratch/<feature-slug>/issues/<NN>-<slug>.md`
- Issue numbering starts at `01`
- Triage state is recorded as a `Status:` line near the top
- Comments are appended under a `## Comments` heading

## Publishing and fetching

When a skill says to publish to the issue tracker, create the corresponding
file under `.scratch/<feature-slug>/`.

When a skill says to fetch a ticket, read the referenced local Markdown file.

## Wayfinding

- The effort map is `.scratch/<effort>/map.md`
- Child tickets live under `.scratch/<effort>/issues/`
- `Type:` records `research`, `prototype`, `grilling`, or `task`
- `Status:` records `claimed` or `resolved`
- `Blocked by:` lists prerequisite ticket numbers
- Claim work by setting `Status: claimed` before starting
- Resolve work by adding an `## Answer`, setting `Status: resolved`, and
  recording the outcome in the effort map
