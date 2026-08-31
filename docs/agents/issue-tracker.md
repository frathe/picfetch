# Issue tracker: Local Markdown

Issues and specs for this repo live as Markdown files in `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`
- The spec is `.scratch/<feature-slug>/spec.md`
- Implementation issues are one file per ticket at `.scratch/<feature-slug>/issues/<NN>-<slug>.md`, numbered from `01`
- Triage state is recorded as a `Status:` line near the top of each issue file
- Comments and conversation history are appended under `## Comments`

## Publishing and fetching

When a skill says “publish to the issue tracker”:

- Write a spec to `.scratch/<feature-slug>/spec.md`
- Write each implementation ticket to `.scratch/<feature-slug>/issues/<NN>-<slug>.md`

When a skill says “fetch the relevant ticket,” read the referenced file. Ticket numbers are local to a feature, so use a file path or `<feature-slug>/<NN>` when the feature is not otherwise clear.

## Wayfinding operations

Used by `/wayfinder`. The map is one file with one child file per ticket.

- **Map**: `.scratch/<effort>/map.md`
- **Child ticket**: `.scratch/<effort>/issues/NN-<slug>.md`
- **Type**: a `Type:` line containing `research`, `prototype`, `grilling`, or `task`
- **Status**: a `Status:` line containing `claimed` or `resolved`
- **Blocking**: a `Blocked by: NN, NN` line; a ticket is unblocked when every listed ticket is resolved
- **Frontier**: first numbered ticket that is open, unblocked, and unclaimed
- **Claim**: set `Status: claimed` before beginning work
- **Resolve**: append the result under `## Answer`, set `Status: resolved`, and add a context pointer to the map’s Decisions-so-far section
