# In-app updater Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user.

**Goal:** Opt-in, silent GitHub-release updater: at most one check per local calendar day, background download with SHA-256 + GitHub **immutable release attestation** verification, apply on quit (no relaunch), What's New markdown on the next launch.

**Architecture:** Viewer-independent logic lives in `internal/update` (HTTP, hashes, attestations, extract, apply). `internal/ui` owns the setting, the once-a-day trigger, `requestLifecycle` + `completion.Signal`, and OnStopped apply. What's New is a `widgets.Singleton` on `internal/ui/help`, same markdown path as the manual. OS binary swap is a dispatcher var (`update.Apply`) with unix/windows files, stubbed in tests like `wallpaper.Set`.

**Tech Stack:** Go 1.26 (see `go.mod`), stdlib `net/http` + `archive/{zip,tar}`, `golang.org/x/mod/semver`, GitHub Releases REST API + **release** Artifact Attestations (`predicate_type=release`), `github.com/sigstore/sigstore-go` behind a `Verifier` interface using **GitHub's** Fulcio/TSA trust root (not Sigstore Public Good). Fyne preferences / `lang.L` / `widget.NewRichTextFromMarkdown`. Do **not** add `go-selfupdate`, Sparkle, or WinSparkle.

## Pivot (2026-08-26)

The repo already uses **GitHub immutable releases**. Publishing a release mints a Sigstore bundle whose in-toto predicate is `https://in-toto.io/attestation/release/v0.2`, subjects are the six archives with SHA-256, and the Fulcio SAN is `https://dotcom.releases.github.com` (signer `O=GitHub, Inc., CN=Attester`, RFC3161 timestamps). Observed on `v0.2.5`.

**Do not** add `SHA256SUMS` or `actions/attest-build-provenance`. Those would attest workflow provenance and miss the attestation GitHub already issues. Task 1 as originally written is **superseded by Task 1b**. Task 2's `ParseChecksums` / `ChecksumsName` are leftover and Task 1b removes them; keep `VerifyHash`.

## Product behavior (normative)

1. **Off by default.** Settings checkbox `Check for updates`. Unchecked on a fresh install (`p.Bool`, same as `MergeMode`).
2. **When turned on:** check immediately. After that, check on the **first launch of each local calendar day** (`YYYY-MM-DD` in `time.Local`). No midnight timer while the app stays open. No "Check now" menu item. No toast, no progress UI, no modal "update available".
3. **Skip the network** when the setting is off, `app.Metadata().Version` is empty/invalid (plain `go test` / un-packaged `go build`), or this `GOOS/GOARCH` has no release asset.
4. **Newer only.** Compare `v`+`Metadata.Version` to the latest *non-draft, non-prerelease* tag with `semver.Compare`. Never downgrade. `/releases/latest` already skips drafts and prereleases.
5. **Download in the background.** Cancel in-flight work on setting-off, shutdown, or a newer check. A partial download is discarded.
6. **Verify or refuse.** SHA-256 the archive. If the Releases API `digest` field is present, it must match. Then fetch `GET /repos/frathe/picfetch/attestations/sha256:{hex}?predicate_type=release` and Sigstore-verify the bundle against GitHub's trust root with SAN `https://dotcom.releases.github.com`. The in-toto statement must name this asset, this digest, `predicate.repository == "frathe/picfetch"`, and `predicate.tag` equal to the release version. Fail closed: mismatch, missing attestation, or failed signature → log + delete staging, do not apply.
7. **Apply on the next close.** `Lifecycle().SetOnStopped` in `registerShutdown`, **after** session/preferences save. Do **not** relaunch. Unix: rename-swap the running binary (and macOS `Info.plist` when present). Windows: detached `cmd` helper that waits for this PID, then copies; use `CREATE_NO_WINDOW` (`0x08000000`), same as `internal/clipboard/windows.go`.
8. **What's New on the next launch** of the *new* version: `help.ShowWhatsNew` renders the GitHub release `body` as markdown. Notes are cached at staging time so the window works offline. Show once, then delete the cache. If apply failed, current version ≠ cached version → do not show.
9. **Failures are silent in the UI.** `fyne.LogError` at the UI boundary; `internal/update` returns errors. Do not strip quarantine, disable Gatekeeper, or otherwise fight OS trust. First browser install of an unsigned build is unchanged.

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Every user-visible string is `lang.L("English text")` with the same key in `translations/en.json` and `translations/de.json`. English is an identity map. `main_test.go` enforces locale parity.
- `internal/update` is viewer-independent and returns errors. UI glue uses `fyne.LogError`.
- Background work uses `requestLifecycle` + `completion.Signal`; add the signal to `drain` in `internal/ui/harness_test.go`. Marshal UI (What's New) through `fyne.Do` if it is not already on the UI goroutine. `SetOnStarted` is already on the UI goroutine.
- No mutable package-level test seams except dispatcher vars (`update.Apply`) and injectable `Client` fields (`HTTP`, `Verify`, `Now`, `LookPath`/`run` as needed). Do not add package-level `var checkForUpdates`.
- Tests never hit live GitHub. Use `httptest.Server`. `newTestUI` must not start a check: wire the client in `startViewerRuntime` (production `Run` only), not `registerFeatures`.
- Update `ARCHITECTURE.md` in the same change that adds `internal/update`.
- Preserve `gofmt` / `goimports -local github.com/frathe/picfetch`. Tabs, not spaces.
- Subagents must not start Task N+1 themselves. They stop after their task's verification and report.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `cursor-grok-4.5-high-fast` | YAML + README; superseded. |
| Task 1b implementer | `cursor-grok-4.5-high-fast` | Revert provenance CI; drop unused checksum parser. |
| Tasks 2–5 implementer | `cursor-grok-4.6-xhigh` | New package, TDD, OS apply, Sigstore wrapper. |
| Tasks 6–8 implementer | `cursor-grok-4.6-xhigh` | Settings/preferences wiring, Fyne lifecycle, help window, docs. |
| Task reviewer (each task) | `cursor-grok-4.5-high-fast` | Mid-tier floor. |
| Parent review / fix after each task | this session (do not dispatch) | Review and fix after every step. |
| Final whole-branch review | `cursor-grok-4.6-xhigh` | Cross-task type consistency + silent-UX check. |

Do not use `claude-opus-5-thinking-high` unless a task is blocked on Sigstore API mismatch that the implementer cannot resolve from `sigstore-go` docs.

## File structure

- Create: `internal/update/update.go` — constants, `Due`, `NormalizeVersion`, `AssetName`, `VerifyHash`
- Create: `internal/update/github.go` — latest release + release-attestation bundle fetch
- Create: `internal/update/download.go` — download, hash, verify, extract, write `Stage`
- Create: `internal/update/extract.go` — zip/tar.gz, zip-slip guard, payload paths
- Create: `internal/update/attest.go` — `Verifier` interface + GitHub Fulcio Sigstore impl
- Create: `internal/update/embed/tuf-repo.github.com/root.json` — GitHub TUF root (Task 4)
- Create: `internal/update/apply.go` — `var Apply` dispatcher
- Create: `internal/update/apply_unix.go` — `//go:build !windows` rename-swap
- Create: `internal/update/apply_windows.go` — `//go:build windows` detached helper
- Create: matching `*_test.go` files next to the above
- Keep: `internal/update/checksums.go` with `VerifyHash` only (`ParseChecksums` / `ChecksumsName` removed in Task 1b)
- Modify: `.github/workflows/release.yml` — **no** SHA256SUMS, **no** `attest-build-provenance`; `permissions: contents: write` only
- Modify: `internal/preferences/preferences.go` (+ test) — `CheckForUpdates`, `LastUpdateCheckDay`
- Modify: `internal/ui/settingswin/settingswin.go` (+ test, `fakeHost`) — checkbox
- Modify: `internal/ui/memlimits.go` — `settings.checkForUpdates` / `lastUpdateCheckDay`
- Modify: `internal/ui/features.go`, `run.go`, `harness_test.go`, `preferences_wiring_test.go`
- Create: `internal/ui/autoupdate.go` (+ test) — glue
- Create: `internal/ui/help/whatsnew.go` (+ test)
- Modify: `internal/ui/help/help.go` — `whatsNewWin` field
- Modify: `translations/en.json`, `translations/de.json`
- Modify: `internal/ui/help/manual.md`, `manual_de.md`
- Modify: `ARCHITECTURE.md`, `README.md` (immutable release attestations)
- Modify: `todos.md` — already points at this plan; move to Done when the branch is complete
- Modify: `go.mod` / `go.sum` — `golang.org/x/mod`, `github.com/sigstore/sigstore-go` (Task 4)

## Locked constants

```go
const (
    RepoOwner             = "frathe"
    RepoName              = "picfetch"
    APIHost               = "https://api.github.com" // Client.BaseURL in tests
    ReleaseAttestationSAN = "https://dotcom.releases.github.com"
    GitHubTUFMirror       = "https://tuf-repo.github.com"
)
```

`AssetName(goos, goarch)` maps runtime onto the names `release.yml` already publishes. darwin/amd64 is `x86_64` in the filename (`uname -m`), not `amd64`.

| goos/goarch | asset |
|-------------|--------|
| darwin/arm64 | `picfetch-macos-arm64.zip` |
| darwin/amd64 | `picfetch-macos-x86_64.zip` |
| windows/amd64 | `picfetch-windows-amd64.zip` |
| windows/arm64 | `picfetch-windows-arm64.zip` |
| linux/amd64 | `picfetch-linux-amd64.tar.gz` |
| linux/arm64 | `picfetch-linux-arm64.tar.gz` |
| anything else | `ok=false` — no check |

GitHub JSON (only the fields we use):

```go
type ghRelease struct {
    TagName    string    `json:"tag_name"`
    Body       string    `json:"body"`
    Draft      bool      `json:"draft"`
    Prerelease bool      `json:"prerelease"`
    Assets     []ghAsset `json:"assets"`
}
type ghAsset struct {
    Name               string `json:"name"`
    BrowserDownloadURL string `json:"browser_download_url"`
    Digest             string `json:"digest"` // optional "sha256:hex"; if set, must match computed hash too
}
```

User-Agent: `picfetch/<version>` (GitHub rejects empty UA). Timeout: 30s on the production `http.Client`.

---

### Task 1: Publish SHA256SUMS and attest release artifacts

> **SUPERSEDED by Task 1b.** Already executed against the pre-pivot plan. Do not re-run. Task 1b reverts the CI extras and README.

**Model:** `cursor-grok-4.5-high-fast` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Modify: `.github/workflows/release.yml`
- Modify: `README.md` — one short paragraph under Download: releases include `SHA256SUMS` and GitHub Artifact Attestations

**Interfaces:**
- Consumes: existing artifact names in `release.yml` (do not rename zips)
- Produces: release asset `SHA256SUMS`; Sigstore attestations for every file listed in it, signed by this workflow. Task 3's parser and Task 4's verifier treat that as the contract.

TDD does not apply (YAML). Do not add Go code in this task.

- [ ] **Step 1: Widen workflow permissions**

Replace the top-level `permissions` block in `.github/workflows/release.yml` with:

```yaml
permissions:
  contents: write
  id-token: write
  attestations: write
```

`id-token` and `attestations` are required for `actions/attest-build-provenance`. Keep `contents: write` so `softprops/action-gh-release` can still attach files.

- [ ] **Step 2: Hash and attest in the `release` job, then attach SHA256SUMS**

In the `release` job, after `Download artifacts` and **before** `Create GitHub release`, add:

```yaml
      - name: Checksums and attestations
        working-directory: dist
        run: |
          sha256sum \
            picfetch-macos-arm64.zip \
            picfetch-macos-x86_64.zip \
            picfetch-windows-amd64.zip \
            picfetch-windows-arm64.zip \
            picfetch-linux-amd64.tar.gz \
            picfetch-linux-arm64.tar.gz \
            > SHA256SUMS
          cat SHA256SUMS

      - name: Attest release artifacts
        uses: actions/attest-build-provenance@v3
        with:
          subject-checksums: dist/SHA256SUMS
```

`softprops/action-gh-release` already uploads `dist/*`; after this step `dist/SHA256SUMS` is included. Do not change artifact names.

If `actions/attest-build-provenance@v3` is unpublished when this is implemented, use the current major documented on https://github.com/actions/attest-build-provenance (still `subject-checksums`).

- [ ] **Step 3: Document the files on the Downloads section**

In `README.md` under `## Download`, after the existing artifact-name sentence, add:

```
Each GitHub release also attaches a `SHA256SUMS` file and GitHub Artifact
Attestations (Sigstore) for those archives. The in-app updater, when enabled,
refuses to install a build that fails either check.
```

- [ ] **Step 4: Verify the YAML still lists every existing asset**

Read the `Create GitHub release` step: `files: dist/*` must remain. `build-macos` / `build-cross` artifact names must be unchanged. Run no Go tests.

---

### Task 1b: Use immutable release attestations (revert SHA256SUMS / provenance)

**Model:** `cursor-grok-4.5-high-fast` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Modify: `.github/workflows/release.yml` — restore `permissions: contents: write` only; delete the Checksums and Attest steps added in Task 1
- Modify: `README.md` — replace the Task 1 Download paragraph with the immutable-release text below
- Modify: `internal/update/update.go` — drop `ChecksumsName`; package comment talks about release attestations, not checksum files
- Modify: `internal/update/checksums.go` / `checksums_test.go` — delete `ParseChecksums` and its tests; keep `VerifyHash` and `TestVerifyHash*`
- Test: `go test -count=1 ./internal/update/`

**Interfaces:**
- Consumes: Task 2 `VerifyHash`, `AssetName`, `Newer`, `Due`
- Produces: CI publishes the same six archives only; README documents GitHub immutable release attestations; no `SHA256SUMS` asset; no `ParseChecksums` / `ChecksumsName`. Task 3 Check uses API `digest`, not a sums file.

TDD: existing `TestParseChecksums*` must fail to compile after deletion, then be removed so `go test ./internal/update/` is green. `TestVerifyHash` stays.

- [ ] **Step 1: Revert release.yml extras**

Top-level permissions must be exactly:

```yaml
permissions:
  contents: write
```

Delete the `Checksums and attestations` and `Attest release artifacts` steps. `Download artifacts` remains immediately followed by `Create GitHub release`. `files: dist/*` unchanged. Artifact names unchanged.

- [ ] **Step 2: README Download paragraph**

Replace the Task 1 paragraph under `## Download` with:

```
Releases are immutable. GitHub issues a Sigstore release attestation that
binds each archive's SHA-256 to the tag. The in-app updater, when enabled,
refuses to install a build that fails that check.
```

- [ ] **Step 3: Drop unused checksum-file parser**

Remove `ChecksumsName` and `ParseChecksums`. Keep `VerifyHash` (Task 3 compares the downloaded bytes to the API digest). Update the package comment to: verifies GitHub immutable release attestations, and replaces the running binary.

- [ ] **Step 4: `go test -count=1 ./internal/update/` PASS**

Do not add sigstore. Do not start Task 3.

---

### Task 2: Version, asset names, daily due, SHA256SUMS parser

> **Complete.** `ParseChecksums` / `ChecksumsName` were leftover from the pre-pivot plan; Task 1b removes them. Keep `VerifyHash`, `AssetName`, `Newer`, `Due`. Do not re-run.

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Create: `internal/update/update.go`
- Create: `internal/update/checksums.go`
- Create: `internal/update/update_test.go`
- Create: `internal/update/checksums_test.go`
- Modify: `go.mod` / `go.sum` — add `golang.org/x/mod` only (`go get golang.org/x/mod@latest` is fine; do not add sigstore yet)

**Interfaces:**
- Consumes: SHA256SUMS format from Task 1
- Produces: `NormalizeVersion`, `Newer`, `AssetName`, `Due`, `ParseChecksums`, `VerifyHash` as specified below. Later tasks import these names exactly.

- [ ] **Step 1: Write the failing tests**

`internal/update/update_test.go`:

```go
package update

import (
    "testing"
    "time"
)

func TestAssetName(t *testing.T) {
    tests := []struct {
        goos, goarch, want string
        ok                 bool
    }{
        {"darwin", "arm64", "picfetch-macos-arm64.zip", true},
        {"darwin", "amd64", "picfetch-macos-x86_64.zip", true},
        {"windows", "amd64", "picfetch-windows-amd64.zip", true},
        {"windows", "arm64", "picfetch-windows-arm64.zip", true},
        {"linux", "amd64", "picfetch-linux-amd64.tar.gz", true},
        {"linux", "arm64", "picfetch-linux-arm64.tar.gz", true},
        {"freebsd", "amd64", "", false},
        {"linux", "386", "", false},
    }
    for _, tc := range tests {
        got, ok := AssetName(tc.goos, tc.goarch)
        if got != tc.want || ok != tc.ok {
            t.Errorf("AssetName(%q, %q) = %q, %v; want %q, %v",
                tc.goos, tc.goarch, got, ok, tc.want, tc.ok)
        }
    }
}

func TestNewer(t *testing.T) {
    tests := []struct {
        current, latest string
        want            bool
    }{
        {"0.2.5", "v0.2.6", true},
        {"v0.2.5", "v0.2.5", false},
        {"0.2.6", "v0.2.5", false},
        {"", "v0.2.6", false},
        {"0.2.5", "not-a-version", false},
        {"0.2.5", "v0.2.6-rc.1", false}, // prerelease latest is not newer-stable
    }
    for _, tc := range tests {
        if got := Newer(tc.current, tc.latest); got != tc.want {
            t.Errorf("Newer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
        }
    }
}

func TestDue(t *testing.T) {
    now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.Local)
    if !Due("", now) {
        t.Error("empty last day must be due (first check after enabling)")
    }
    if Due("2026-08-26", now) {
        t.Error("same local calendar day must not be due")
    }
    if !Due("2026-08-25", now) {
        t.Error("previous local calendar day must be due")
    }
}

func TestDayStringUsesLocalDate(t *testing.T) {
    loc := time.FixedZone("test", 2*60*60)
    ts := time.Date(2026, 8, 26, 1, 0, 0, 0, loc)
    if got := DayString(ts); got != "2026-08-26" {
        t.Errorf("DayString = %q, want 2026-08-26 in the timestamp's location", got)
    }
}
```

`internal/update/checksums_test.go`:

```go
package update

import (
    "crypto/sha256"
    "encoding/hex"
    "testing"
)

func TestParseChecksums_GNUAndBinaryStar(t *testing.T) {
    const body = `# comment
deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  picfetch-macos-arm64.zip
cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe *picfetch-windows-amd64.zip

`
    m, err := ParseChecksums([]byte(body))
    if err != nil {
        t.Fatal(err)
    }
    if m["picfetch-macos-arm64.zip"] != "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef" {
        t.Errorf("gnu line: %+v", m)
    }
    if m["picfetch-windows-amd64.zip"] != "cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe" {
        t.Errorf("star line: %+v", m)
    }
}

func TestVerifyHash(t *testing.T) {
    sum := sha256.Sum256([]byte("hello"))
    hexSum := hex.EncodeToString(sum[:])
    if err := VerifyHash([]byte("hello"), hexSum); err != nil {
        t.Fatal(err)
    }
    if err := VerifyHash([]byte("hello"), "00"+hexSum[2:]); err == nil {
        t.Fatal("want mismatch error")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./internal/update/`

Expected: FAIL, package not found or undefined names.

- [ ] **Step 3: Implement**

`internal/update/update.go`:

```go
package update

import (
    "strings"
    "time"

    "golang.org/x/mod/semver"
)

const (
    RepoOwner     = "frathe"
    RepoName      = "picfetch"
    ChecksumsName = "SHA256SUMS"
)

func AssetName(goos, goarch string) (string, bool) {
    switch goos + "/" + goarch {
    case "darwin/arm64":
        return "picfetch-macos-arm64.zip", true
    case "darwin/amd64":
        return "picfetch-macos-x86_64.zip", true
    case "windows/amd64":
        return "picfetch-windows-amd64.zip", true
    case "windows/arm64":
        return "picfetch-windows-arm64.zip", true
    case "linux/amd64":
        return "picfetch-linux-amd64.tar.gz", true
    case "linux/arm64":
        return "picfetch-linux-arm64.tar.gz", true
    default:
        return "", false
    }
}

// NormalizeVersion returns a canonical semver (leading v) or "".
func NormalizeVersion(s string) string {
    s = strings.TrimSpace(s)
    if s == "" {
        return ""
    }
    if s[0] != 'v' {
        s = "v" + s
    }
    if !semver.IsValid(s) {
        return ""
    }
    if semver.Prerelease(s) != "" {
        return ""
    }
    return semver.Canonical(s)
}

// Newer reports whether latest is a stable semver strictly greater than current.
func Newer(current, latest string) bool {
    c := NormalizeVersion(current)
    l := NormalizeVersion(latest)
    if c == "" || l == "" {
        return false
    }
    return semver.Compare(l, c) > 0
}

func DayString(t time.Time) string {
    // Format uses t's Location. Production passes time.Now() (local).
    // Tests pass a time constructed in a fixed zone so the day is deterministic.
    return t.Format("2006-01-02")
}

// Due reports whether a check should run: never checked, or last check was
// a previous local calendar day. lastDay is DayString's format or empty.
func Due(lastDay string, now time.Time) bool {
    today := DayString(now)
    return lastDay == "" || lastDay != today
}
```

`internal/update/checksums.go`: parse lines with `bufio.Scanner`; skip empty/`#`; split on first whitespace; trim a leading `*` from the filename; require `len(hex)==64` and hex-only; `strings.ToLower` the hex. `VerifyHash` sha256s `data` and compares with `subtle.ConstantTimeCompare` against the decoded expected hex.

Package comment on `update.go`: this package talks to GitHub Releases, verifies checksums and attestations, and replaces the running binary. It has no Fyne types.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/update/`

Expected: PASS. Then `go tool goimports -local github.com/frathe/picfetch -w internal/update/`

---

### Task 3: Fetch, download, hash-verify, extract, stage

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Create: `internal/update/github.go`
- Create: `internal/update/download.go`
- Create: `internal/update/extract.go`
- Create: `internal/update/attest.go` — `Verifier` + `VerifyPolicy` only; no sigstore import
- Create: `internal/update/github_test.go`, `download_test.go`, `extract_test.go`
- Modify: `internal/update/update.go` — add `APIHost`, `Client`, `Config`, `Stage`, `Release`

**Interfaces:**
- Consumes: Task 2 `Newer`, `AssetName`, `VerifyHash`, `NormalizeVersion`. No checksum file. No `ParseChecksums`.
- Produces:

```go
type Config struct {
    BaseURL  string // default APIHost, httptest in tests
    HTTP     Doer
    Now      func() time.Time
    Verify   Verifier // required in production (Task 4). Task 3 Download does not call it.
    StageDir string
    GOOS     string // default runtime.GOOS
    GOARCH   string
}

type Doer interface {
    Do(*http.Request) (*http.Response, error)
}

type Release struct {
    Version     string // canonical vX.Y.Z
    Notes       string // GitHub body, may be empty
    AssetName   string
    AssetURL    string
    AssetDigest string // hex from API digest without "sha256:" prefix; empty if omitted
}

type Stage struct {
    Version    string
    Notes      string
    BinaryPath string
    PlistPath  string // darwin extracted Info.plist; empty otherwise
}

func NewClient(cfg Config) *Client

// Check reports a Release when an update should be downloaded, or (nil, nil)
// when none. Errors are network/API/parse failures.
func (c *Client) Check(ctx context.Context, currentVersion string) (*Release, error)

// Download fetches rel's archive, SHA-256s it, compares to AssetDigest when
// set, extracts, and writes Stage under StageDir. Attestation verify is Task 4.
func (c *Client) Download(ctx context.Context, rel Release) (Stage, error)

func LoadStage(dir string) (Stage, error) // missing file → (Stage{}, os.ErrNotExist)
func SaveStage(dir string, s Stage) error
func RemoveStage(dir string) error
```

Declare this in `attest.go` now (no Sigstore import). Task 4 fills in `NewSigstoreVerifier` and makes `Download` call `Verify`. Do not change the signature later.

```go
type VerifyPolicy struct {
    Tag       string // canonical vX.Y.Z
    AssetName string
}

type Verifier interface {
    Verify(ctx context.Context, digest, bundle []byte, policy VerifyPolicy) error
}
```

`digest` is the raw 32-byte SHA-256 of the archive. Task 3 `Download` compares `AssetDigest` when non-empty and does **not** call `Verify`. Production must not ship between Task 3 and Task 4.

- [ ] **Step 1: Write failing httptest for Check**

Serve `GET /repos/frathe/picfetch/releases/latest` → JSON with `tag_name: v0.2.6`, `body: "## Fixes\n\n- toast"`, one asset `picfetch-linux-amd64.tar.gz` with a `browser_download_url` on the same server and `digest: "sha256:"` + 64 hex chars.

Client `GOOS=linux, GOARCH=amd64`, `currentVersion=0.2.5`. Assert `Release.Version == "v0.2.6"`, `Notes` contains `toast`, `AssetName` matches, `AssetDigest` is the hex (no `sha256:` prefix).

Also: same tag as current → `Check` returns `nil, nil`. Unsupported GOOS → `nil, nil`. Draft/prerelease true → `nil, nil`. Missing platform asset → error. Missing / empty `digest` → still a Release with `AssetDigest == ""` (not an error). Non-empty digest that is not `sha256:` + 64 hex → error.

Do **not** require a `SHA256SUMS` asset. Check does not download the archive.

- [ ] **Step 2: Run `go test -count=1 -run TestCheck ./internal/update/` — FAIL**

- [ ] **Step 3: Implement Check**

`GET {BaseURL}/repos/{owner}/{repo}/releases/latest` with headers:

- `Accept: application/vnd.github+json`
- `User-Agent: picfetch/<currentVersion>`
- `X-GitHub-Api-Version: 2022-11-28`

Default `BaseURL` is `APIHost`. Decode `ghRelease`. If `Draft` or `Prerelease` or `!Newer(current, TagName)` → `(nil, nil)`. `AssetName(GOOS,GOARCH)`; find that asset; fill `Release`. Strip a leading `sha256:` from `ghAsset.Digest` (`strings.TrimPrefix`, lowercase hex). Do not use a GitHub token. Do not fetch attestations here.

- [ ] **Step 4: Extract tests (zip slip, payload paths)**

Build a zip in the test with `PicFetch.app/Contents/MacOS/picfetch` bytes `newbin` and `PicFetch.app/Contents/Info.plist` bytes `plist`. `extract(ctx, zipPath, destDir)` must produce those two files. A zip entry named `../escape` or `/tmp/x` must error and not write outside destDir.

Linux tarball: `picfetch-linux-amd64` file in a `.tar.gz` → `BinaryPath` that file, `PlistPath` empty.

Windows zip: `picfetch.exe` → `BinaryPath` that file.

Payload picker (exact):

1. If `…/Contents/MacOS/picfetch` exists (any parent `*.app`) → that binary; plist is `…/Contents/Info.plist` if present.
2. Else if `picfetch.exe` exists → that.
3. Else if exactly one extracted file whose name has prefix `picfetch-linux-` or equals `picfetch` → that.
4. Else error.

- [ ] **Step 5: Download test**

httptest serves the archive bytes. `Release.AssetDigest` is that file's sha256 hex. `Download` writes `StageDir/stage.json` + extracted files. `LoadStage` round-trips Version/Notes/BinaryPath. Wrong `AssetDigest` → error and StageDir has no usable `stage.json`. Empty `AssetDigest` → still succeeds (hash is computed; attestation bind is Task 4).

- [ ] **Step 6: Implement Download + extract + stage JSON**

`stage.json`:

```go
type stageFile struct {
    Version    string `json:"version"`
    Notes      string `json:"notes"`
    BinaryPath string `json:"binaryPath"`
    PlistPath  string `json:"plistPath,omitempty"`
}
```

Paths in the JSON are absolute. `RemoveStage` deletes the whole StageDir contents (or the dir). Download starts by `RemoveStage` so a failed retry cannot apply a mix of two versions.

Limit archive size: 200 MiB (`io.LimitReader`); oversize is an error.

Zip-slip: `filepath.Join(dest, entry)` then `filepath.Rel(dest, cleaned)` must not start with `..`.

If `rel.AssetDigest != ""`, `VerifyHash(archiveBytes, rel.AssetDigest)` before extract.

- [ ] **Step 7: `go test -count=1 ./internal/update/` PASS**

---

### Task 4: Immutable release attestation verification

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Modify: `internal/update/attest.go` — GitHub Fulcio Sigstore impl + in-toto release policy
- Create: `internal/update/attest_test.go`
- Create: `internal/update/embed/tuf-repo.github.com/root.json` — GitHub TUF root (not Sigstore Public Good)
- Modify: `internal/update/download.go` — after hash verify, fetch and verify attestation; required
- Modify: `go.mod` / `go.sum` — `github.com/sigstore/sigstore-go`

**Interfaces:**
- Consumes: Task 3 `Verifier` / `VerifyPolicy`; archive digest from Download
- Produces: `NewSigstoreVerifier() (Verifier, error)`; Download fails closed without a verified **release** attestation. This is GitHub's immutable-release attestation (`predicate_type=release`), not Actions build provenance.

Pinned identity (same policy `gh release verify-asset` uses in `cli/cli` `pkg/cmd/release/shared/attestation.go`):

- Trust root: GitHub Sigstore TUF (`GitHubTUFMirror`), **not** `root.FetchTrustedRoot()` / Public Good
- Verifier options: `verify.WithSignedTimestamps(1)` only — **not** SCT + Rekor + `WithObserverTimestamps`
- SAN regex: `^https://dotcom\.releases\.github\.com$` (`ReleaseAttestationSAN`)
- OIDC issuer: any (`verify.NewIssuerMatcher("", ".*")`) — GitHub's attester is not the Actions OIDC identity
- Artifact digest policy: archive SHA-256 (`verify.WithArtifactDigest("sha256", digest)`)

Then, on the verified in-toto statement (fail closed):

- `predicateType` is a release predicate: prefix `https://in-toto.io/attestation/release/`
- Some subject has `name == policy.AssetName` and `digest.sha256` equal to the archive hex
- `predicate.repository == "frathe/picfetch"`
- `predicate.tag == policy.Tag` (canonical `vX.Y.Z`)
- `predicate.purl == "pkg:github/frathe/picfetch@" + policy.Tag`

This does **not** prove `release.yml` built the zip. A published draft still gets a release attestation. Accepted.

- [ ] **Step 1: Write tests that do not need a live Sigstore bundle or live GitHub**

1. SAN regex / identity helpers: exact `^https://dotcom\.releases\.github\.com$`. Do not pin `release.yml@refs/tags/`.
2. Download httptest: after optional `AssetDigest` passes, `GET /repos/frathe/picfetch/attestations/sha256:{hex}?predicate_type=release` returns `{"attestations":[{"bundle":{}}]}`. A fake `Verifier` records `(digest, bundleJSON, policy)` and returns nil. Assert raw SHA-256 bytes, `policy.Tag == rel.Version`, `policy.AssetName == rel.AssetName`. HTTP 404 → Download error. Fake `Verifier` error → Download error, no `stage.json`. Empty `attestations` array → error. Missing query `predicate_type=release` in the request → test fails.
3. `checkReleaseStatement` (unexported) with a small JSON fixture: happy path; wrong repository; wrong tag; digest present under a different subject name; missing asset name; non-release `predicateType`. No network.

Fake (same `Verifier` signature as Task 3):

```go
type fakeVerifier struct {
    digest []byte
    bundle []byte
    policy VerifyPolicy
    err    error
}

func (f *fakeVerifier) Verify(_ context.Context, digest, bundle []byte, policy VerifyPolicy) error {
    f.digest = append([]byte(nil), digest...)
    f.bundle = append([]byte(nil), bundle...)
    f.policy = policy
    return f.err
}
```

GitHub response shape (parse only what we need):

```go
type ghAttestations struct {
    Attestations []struct {
        Bundle json.RawMessage `json:"bundle"`
    } `json:"attestations"`
}
```

Use the first attestation's `bundle` raw JSON. Pass `VerifyPolicy{Tag: rel.Version, AssetName: rel.AssetName}`.

- [ ] **Step 2: Run tests — FAIL on missing attest fetch / missing statement checks**

- [ ] **Step 3: Implement fetch + production verifier**

Fetch: `GET {BaseURL}/repos/frathe/picfetch/attestations/sha256:{hex}?predicate_type=release` with the same Accept/UA headers as Check. The `release` sentinel is version-agnostic (v0.1 / v0.2). Do not fetch by git SHA. Do not call `root.FetchTrustedRoot()`.

Embed GitHub's TUF root (download `https://tuf-repo.github.com/root.json` once at implement time into `internal/update/embed/tuf-repo.github.com/root.json`). `NewSigstoreVerifier`:

```go
opts := tuf.DefaultOptions()
opts.Root = githubTUFRoot // //go:embed
opts.RepositoryBaseURL = GitHubTUFMirror
opts.CachePath = filepath.Join(userCache, "picfetch", "tuf") // UserCacheDir, else TempDir
client, err := tuf.New(opts)
trusted, err := root.GetTrustedRoot(client)
inner, err := verify.NewVerifier(trusted, verify.WithSignedTimestamps(1))
```

`(*sigstoreVerifier).Verify` loads the bundle from JSON (`bundle.LoadJSON` or whatever this module version exports), then:

```go
sanMatcher, err := verify.NewSANMatcher("", `^https://dotcom\.releases\.github\.com$`)
issuerMatcher, err := verify.NewIssuerMatcher("", ".*")
certID, err := verify.NewCertificateIdentity(sanMatcher, issuerMatcher, certificate.Extensions{})
_, err = inner.Verify(entity, verify.NewPolicy(
    verify.WithArtifactDigest("sha256", digest),
    verify.WithCertificateIdentity(certID),
))
```

Then `checkReleaseStatement` on the DSSE/in-toto payload. If the module still names the constructor `NewSignedEntityVerifier`, use that name; keep SAN, TSA-only options, and statement policy identical.

`Client.Download` after hashes: `if c.Verify == nil { return Stage{}, errors.New("update: missing attestation verifier") }`. Fetch bundle, then `c.Verify.Verify(ctx, digest[:], bundleJSON, VerifyPolicy{Tag: rel.Version, AssetName: rel.AssetName})`.

Do not add a live integration test against github.com or tuf-repo.github.com.

- [ ] **Step 4: `go test -count=1 ./internal/update/` PASS** including a `go vet` of the new import.

If `sigstore-go` pulls something incompatible with `go 1.26`, stop and report; do not downgrade the Go directive.

---

### Task 5: Apply on unix and Windows

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Create: `internal/update/apply.go`
- Create: `internal/update/apply_unix.go` (`//go:build !windows`)
- Create: `internal/update/apply_windows.go` (`//go:build windows`)
- Create: `internal/update/apply_test.go` (unix path; runs on macOS/Linux CI)
- Create: `internal/update/apply_script_test.go` — Windows helper *text* tested on all OSes
- Create: `internal/update/apply_windows_test.go` (`//go:build windows`) only if a real spawn test is cheap; otherwise skip — script test is the portable coverage

**Interfaces:**
- Consumes: `Stage`
- Produces:

```go
// Apply replaces the running executable at dest with stage.BinaryPath.
// dest is os.Executable() from UI glue. A var so tests stub it.
var Apply = func(stage Stage, dest string) error {
    switch runtime.GOOS {
    case "windows":
        return applyWindows(stage, dest)
    default:
        return applyUnix(stage, dest)
    }
}

func windowsApplyScript(dest, staged string, pid int) string // tested on all GOOS
```

- [ ] **Step 1: Unix apply test**

Temp dir: `dest` file contains `old`, `stage.BinaryPath` contains `new`, mode 0755. `applyUnix(stage, dest)`. Read dest == `new`. Optional: `stage.PlistPath` with bytes `PLIST` and dest `.../PicFetch.app/Contents/MacOS/picfetch` — after apply, `.../Contents/Info.plist` equals `PLIST`.

Unwritable dest (chmod 0555 dir on unix) → error, dest still `old` if rename was rolled back.

- [ ] **Step 2: Implement applyUnix**

```go
func applyUnix(stage Stage, dest string) error {
    dest, err := filepath.EvalSymlinks(dest)
    // rename dest → dest+".old"
    // copy stage.BinaryPath → dest (io.Copy), chmod 0755
    // if copy fails, rename .old back
    // os.Remove(dest+".old") — ignore error (busy inode)
    // if stage.PlistPath != "" { copy over filepath.Join(filepath.Dir(dest), "..", "Info.plist") }
    // do not os.RemoveAll the stage dir here; UI glue calls RemoveStage after successful Apply
}
```

Copy, do not `os.Rename` from StageDir onto dest across filesystems.

- [ ] **Step 3: Windows script test (all platforms)**

`windowsApplyScript("C:\\App\\picfetch.exe", "C:\\cache\\picfetch.exe", 4242)` must:

- Wait until PID 4242 is gone (`tasklist` / `tasklist /FI "PID eq 4242"` loop) or equivalent `timeout` loop
- `copy /Y` staged onto dest
- `del` the staged file
- `del "%~f0"` (self-delete the `.cmd`)
- **Not** start `picfetch.exe` again

- [ ] **Step 4: Implement applyWindows**

Write `dest+".apply.cmd"` with that script. `exec.Command("cmd", "/C", scriptPath)` with `SysProcAttr.CreationFlags = 0x08000000` (CREATE_NO_WINDOW). `cmd.Start()` not `Run()`. Extra: `DETACHED_PROCESS` (0x00000008) if CREATE_NO_WINDOW alone still ties the helper to the dying parent — if the helper dies with the app on a smoke check, add it.

Do not relaunch.

- [ ] **Step 5: `go test -count=1 ./internal/update/` PASS on the current OS.** `GOOS=windows go test -c -o /dev/null ./internal/update/` must compile.

---

### Task 6: Preference + Settings checkbox (default off)

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Modify: `internal/preferences/preferences.go`
- Modify: `internal/preferences/preferences_test.go`
- Modify: `internal/ui/memlimits.go` — `settings` fields
- Modify: `internal/ui/settingswin/settingswin.go` + `settingswin_test.go` (`fakeHost`)
- Modify: `internal/ui/features.go` — restore
- Modify: `internal/ui/run.go` — `currentPreferences`
- Modify: `internal/ui/preferences_wiring_test.go`
- Create: getter/setter next to `favthumbs.go` pattern, in `internal/ui/autoupdate.go` if that file exists; otherwise `internal/ui/updatesetting.go` until Task 7 merges it. Prefer creating `autoupdate.go` with **only** the Host methods in this task:

```go
func (v *viewer) CheckForUpdates() bool
func (v *viewer) SetCheckForUpdates(on bool) // field only; do not start network in this task
func (v *viewer) LastUpdateCheckDay() string
func (v *viewer) SetLastUpdateCheckDay(day string)
```

**Interfaces:**
- Consumes: Host checkbox pattern from `FavoritePreviewCache`
- Produces: `preferences.State.CheckForUpdates bool`, `LastUpdateCheckDay string`; settings Host methods; default **false** / empty day

Keys:

```go
keyCheckForUpdates    = "checkForUpdates"
keyLastUpdateCheckDay = "lastUpdateCheckDay"
```

Load: `p.Bool(keyCheckForUpdates)`, `p.String(keyLastUpdateCheckDay)`. Save: always `SetBool` / `SetString` (empty string is a valid "never").

- [ ] **Step 1: Failing preference tests**

Extend `TestLoadPreferences_NothingSavedReturnsDefaults`: `CheckForUpdates` false, `LastUpdateCheckDay` empty. Extend `TestSavePreferences_RoundTrip` with `CheckForUpdates: true`, `LastUpdateCheckDay: "2026-08-26"`.

- [ ] **Step 2: Failing settings tests**

Mirror `TestFavPreviewCheck_*`: seed `updateCheck: false` without firing `SetCheckForUpdates`; tapping the check to true records one `true` call. `fakeHost` gains `updateCheck bool` and `updateCheckCalls []bool`.

In `build()`, after `favPreviewCheck`:

```go
w.updateCheck = widget.NewCheck(lang.L("Check for updates"), w.host.SetCheckForUpdates)
w.updateCheck.Checked = w.host.CheckForUpdates()
```

Add `updateCheck *widget.Check` on `Window`; nil it in Show's onClosed alongside `favPreviewCheck`. Append to the VBox.

Host interface: `CheckForUpdates() bool`, `SetCheckForUpdates(bool)`.

- [ ] **Step 3: Translations (required for the checkbox to not show the raw key in de)**

`translations/en.json`: `"Check for updates": "Check for updates"`

`translations/de.json`: `"Check for updates": "Nach Updates suchen"`

- [ ] **Step 4: Wiring tests**

`TestCheckForUpdates_DefaultsToFalseOnStartup` — `newTestViewer`, `!v.CheckForUpdates()`. `TestSetCheckForUpdates_UpdatesGetterAndCurrentPreferences` — set true, both getter and `currentPreferences()` true. Restore in `registerFeatures`: `view.SetCheckForUpdates(prefs.CheckForUpdates)` and `view.SetLastUpdateCheckDay(prefs.LastUpdateCheckDay)` **without** starting HTTP (Task 7).

- [ ] **Step 5: Implement until `go test -count=1 ./internal/preferences/ ./internal/ui/settingswin/` and `go test -count=1 -run 'TestCheckForUpdates|TestSetCheckForUpdates|TestLoadPreferences|TestSavePreferences' ./internal/ui/ ./internal/preferences/` PASS.** Also `go test -count=1 -run 'TestTranslations_' .` (`TestTranslations_EveryLocaleCoversEnglish` and `TestTranslations_EnglishMapsEachKeyToItself` in `main_test.go`).

Update `memlimits.go` `settings` comment to mention the updates checkbox. `currentPreferences` must copy both new fields (the FavoritePreviewCache test exists specifically to catch a missing literal — add the analogous assert).

---

### Task 7: Glue — daily check, background download, apply on stop, drain

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Modify: `internal/ui/autoupdate.go` — full orchestration
- Create: `internal/ui/autoupdate_test.go`
- Modify: `internal/ui/viewer.go` — fields `update *update.Client`, `updateOp` lifecycle+done signal, `updateDir string`
- Modify: `internal/ui/run.go` — `startViewerRuntime`, `registerShutdown`
- Modify: `internal/ui/harness_test.go` — `drain` waits `v.updateDone`; `newTestUI` sets `v.updateDir = t.TempDir()` and does **not** assign a Client
- Modify: `internal/ui/build.go` only if `updateDir` must be constructed with the viewer (prefer `defaultUpdateDir()` in `buildViewer` like `defaultWallpaperDir`)

**Interfaces:**
- Consumes: `update.Client`, `Due`, `Apply`, preferences fields from Task 6
- Produces: production path in `startViewerRuntime` only

```go
func defaultUpdateDir() string {
    base, err := os.UserCacheDir()
    if err != nil {
        base = os.TempDir()
    }
    return filepath.Join(base, "picfetch", "updates")
}

func (v *viewer) maybeStartUpdateCheck()
func (v *viewer) applyStagedUpdate()
```

`maybeStartUpdateCheck` rules (all must hold):

1. `v.CheckForUpdates()`
2. `v.update != nil`
3. `NormalizeVersion(v.currentUpdateVersion()) != ""`
4. `Due(v.LastUpdateCheckDay(), v.update.Now())` — tests set `Client.Now`; production uses `time.Now`
5. `AssetName` ok for this GOOS/GOARCH (Client.Check already returns nil,nil)

Then `token := v.updateOp.lifecycle.begin()`; `done := v.updateDone.Begin()`; goroutine:

```go
defer done()
rel, err := v.update.Check(token.ctx, v.currentUpdateVersion())
if err != nil {
    fyne.LogError("update check failed", err)
    return
}
if !token.alive() { return }
v.SetLastUpdateCheckDay(update.DayString(v.update.Now()))
// persist immediately so a crash still skips a second check today:
v.app.Preferences().SetString("lastUpdateCheckDay", v.LastUpdateCheckDay())
if rel == nil { return }
st, err := v.update.Download(token.ctx, *rel)
if err != nil {
    fyne.LogError("update download failed", err)
    return
}
if !token.alive() {
    _ = update.RemoveStage(v.updateDir)
    return
}
_ = st // Download already SaveStage'd
```

Use the same preference key string as `internal/preferences` — **do not duplicate the key literal**. Add `preferences.SetLastUpdateCheckDay(app, day)` or export the key. Cleanest: `preferences.Save` is heavy; add:

```go
func SaveLastUpdateCheckDay(app fyne.App, day string) {
    app.Preferences().SetString(keyLastUpdateCheckDay, day)
}
```

in `internal/preferences` (same package, unexported key). Call that from the glue. Task 6 can have missed this — add it here with a three-line test.

`SetCheckForUpdates(true)` calls `maybeStartUpdateCheck()`. `SetCheckForUpdates(false)` calls `v.updateOp.lifecycle.invalidate()` and does **not** delete an already-complete stage (apply still happens; the bits are already on disk).

`startViewerRuntime`:

```go
if view.updateDir == "" {
    view.updateDir = defaultUpdateDir()
}
ver, err := update.NewSigstoreVerifier()
if err != nil {
    fyne.LogError("update verifier unavailable", err)
} else {
    view.update = update.NewClient(update.Config{
        HTTP:     &http.Client{Timeout: 30 * time.Second},
        Now:      time.Now,
        Verify:   ver,
        StageDir: view.updateDir,
    })
    view.maybeStartUpdateCheck()
}
```

`registerShutdown` **after** `preferences.Save`:

```go
view.updateOp.lifecycle.invalidate()
view.applyStagedUpdate()
```

Do **not** `Wait` the download in OnStopped (quit must stay fast). `applyStagedUpdate`:

```go
func (v *viewer) applyStagedUpdate() {
    st, err := update.LoadStage(v.updateDir)
    if err != nil {
        return
    }
    dest, err := os.Executable()
    if err != nil {
        fyne.LogError("update apply skipped", err)
        return
    }
    if err := update.Apply(st, dest); err != nil {
        fyne.LogError("failed to apply update", err)
        return
    }
    if err := saveWhatsNew(v.app, st.Version, st.Notes); err != nil {
        fyne.LogError("failed to store release notes", err)
    }
    _ = update.RemoveStage(v.updateDir)
}
```

`saveWhatsNew` is Task 8. In this task, write notes to the Fyne cache so Task 8 can read the same key: implement `saveWhatsNew` / `loadWhatsNew` / `clearWhatsNew` here using `app.Cache()` key `whatsnew.json` (same pattern as `internal/session`). Task 8 only adds the window.

```go
type whatsNewCache struct {
    Version string `json:"version"`
    Body    string `json:"body"`
}
```

Write cache **before** Apply on Windows? If Apply only *schedules* a helper, the process still runs `saveWhatsNew` in OnStopped — good, notes land even if the copy happens after exit. Order: save notes, then Apply, then RemoveStage. On Windows RemoveStage must **not** delete `stage.BinaryPath` before the helper copies — `applyWindows` copies from BinaryPath after PID exit, so **do not RemoveStage on Windows in this process**. Unix can RemoveStage after successful rename-swap.

```go
if runtime.GOOS != "windows" {
    _ = update.RemoveStage(v.updateDir)
}
```

Windows helper already `del`s the staged exe; leftover `stage.json` on next launch: `LoadStage` + `NormalizeVersion(current)==st.Version` → `RemoveStage` leftover, do not Apply again. Add that guard at the start of `applyStagedUpdate` and at startup (`maybeStartUpdateCheck` or `startViewerRuntime`): if staged version ≤ current, `RemoveStage`.

- [ ] **Step 1: Tests with httptest Client on a viewer that did go through `newTestViewer`**

Assign `v.updateDir`, `v.update = update.NewClient(...)`, `v.SetCheckForUpdates(true)`, stub `v.app` metadata if empty: Fyne test apps often have empty Version — **set the Client check version by passing it into Check from a test-only field** `v.updateCurrentVersion string` with fallback to Metadata. In tests:

```go
v.updateCurrentVersion = "0.2.5"
```

```go
func (v *viewer) currentUpdateVersion() string {
    if v.updateCurrentVersion != "" {
        return v.updateCurrentVersion
    }
    return v.app.Metadata().Version
}
```

This is per-viewer state, not a package seam.

Cases:

1. Setting off → Check is never called (fake Doer that `t.Error`s on Do).
2. Setting on, Due, httptest latest = current → LastUpdateCheckDay becomes today, no stage.
3. Setting on, httptest newer + matching `digest` + fake Verifier → stage.json exists after `waitFor` on `v.updateDone`.
4. `drain` includes `&v.updateDone` after wallpaper/clipboard in the table, and `v.updateOp.lifecycle.invalidate()` next to the other invalidates.

`waitFor` on never-begun Signal returns immediately — production tests that never start a check stay fast.

Stub `update.Apply` in the apply-on-stop test to record `(stage, dest)` and return nil. Call `v.applyStagedUpdate()` directly (do not need a real Fyne OnStopped).

- [ ] **Step 2: Implement until `go test -count=1 -run 'TestUpdate|TestDrain' ./internal/ui/` and `go test -count=1 ./internal/update/` PASS.** Then `go test -count=1 ./internal/ui/` is the real gate if the run is not too long; at minimum `go test -race -count=1 -run 'TestUpdate|TestCheckForUpdates' ./internal/ui/`.

`SetCheckForUpdates` in Task 6 was field-only; this task adds the maybeStart/invalidate calls.

---

### Task 8: What's New window, manuals, ARCHITECTURE

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.5-high-fast` (reviewer)

**Files:**
- Create: `internal/ui/help/whatsnew.go`
- Create: `internal/ui/help/whatsnew_test.go`
- Modify: `internal/ui/help/help.go` — `whatsNewWin widgets.Singleton`
- Modify: `internal/ui/run.go` — `SetOnStarted` also `view.maybeShowWhatsNew()`
- Modify: `internal/ui/autoupdate.go` — `maybeShowWhatsNew`
- Modify: `translations/*.json` — `"What's New"`, `"What's New in %s"`
- Modify: `internal/ui/help/manual.md` + `manual_de.md` — Settings bullet
- Modify: `ARCHITECTURE.md` — package row + Where-to-look
- Modify: `todos.md` — leave the updater under ACTIVE DEVELOPMENT until the user ships; do not mark Done from this task

**Interfaces:**
- Consumes: `whatsnew.json` from Task 7; `help.Help`
- Produces:

```go
func (h *Help) ShowWhatsNew(version, body string)
```

Window title: `fmt.Sprintf(lang.L("What's New in %s"), strings.TrimPrefix(version, "v"))`. Size: 640×480. Content: `container.NewScroll(widget.NewRichTextFromMarkdown(body))` with `Wrapping = fyne.TextWrapWord`. Resize **before** SetContent (Singleton.Show already does). No tables in our own strings; GitHub bodies with tables may render poorly — accept Fyne's limitation (same as the manual). Escape closes. Do not `KeepOnTop`. Do not add a Help menu item (unintrusive; About is enough). Empty `body` → still show the window with a single translated fallback line `lang.L("This release has no notes.")`.

`maybeShowWhatsNew`:

```go
func (v *viewer) maybeShowWhatsNew() {
    wn, err := loadWhatsNew(v.app)
    if err != nil || wn.Version == "" {
        return
    }
    cur := update.NormalizeVersion(v.currentUpdateVersion())
    if cur == "" || update.NormalizeVersion(wn.Version) != cur {
        return
    }
    clearWhatsNew(v.app)
    v.help.ShowWhatsNew(wn.Version, wn.Body)
}
```

Call from `SetOnStarted` **after** `syncNativeMenuBar`, same callback as CLI drop, so the main window already exists. `newTestUI` does not call `SetOnStarted` — What's New tests call `maybeShowWhatsNew` directly.

- [ ] **Step 1: help package test**

`ShowWhatsNew("v0.2.6", "# Hi\n\n- item")` → window open, title uses the translated format, RichText has segments. Second Show raises rather than stacking (Singleton).

- [ ] **Step 2: UI test**

`newTestViewer`, write `whatsnew.json` via `saveWhatsNew` for `v0.2.6`, set `v.updateCurrentVersion = "0.2.6"`, `maybeShowWhatsNew()`, assert help window open (`ManualOpen`-style reporter: add `WhatsNewOpen() bool` on `Help`). Cache cleared so a second call does not reopen. Version `0.2.5` vs cache `v0.2.6` → do not show.

- [ ] **Step 3: Manual copy**

`manual.md` Settings bullet, append:

```
, and **Check for updates** (off by default). When enabled, PicFetch
  asks GitHub for a newer release at most once per day, downloads it in
  the background, verifies GitHub's immutable release attestation, and
  installs it the next time you quit. The following launch shows a What's
  New window with that release's notes. There is no prompt and no
  auto-restart
```

`manual_de.md` matching German at the Einstellungen bullet (same structure, no tables).

- [ ] **Step 4: ARCHITECTURE.md**

Add under OS integrations / after `internal/session`:

```
### `internal/update`

GitHub-release check, SHA-256 + immutable release attestation verify, stage, apply.

| File | Responsibility |
|------|----------------|
| `update.go` | `Client`, `AssetName`, `Newer`, `Due`. |
| `github.go` | Releases + release-attestation HTTP. |
| `checksums.go` | `VerifyHash` (optional API digest). |
| `download.go` / `extract.go` | Fetch, hash, unzip/tar, `Stage`. |
| `attest.go` | GitHub Fulcio Sigstore `Verifier` + in-toto release policy. |
| `apply.go` / `apply_unix.go` / `apply_windows.go` | `Apply` dispatcher. |
```

Where to look:

```
- "How do in-app updates work?" → `internal/update` + `internal/ui/autoupdate.go` + `help/whatsnew.go`. Off by default (`preferences.CheckForUpdates`). Apply is OnStopped, not a relaunch.
```

Extend the preferences Where-to-look sentence to mention the updates checkbox.

- [ ] **Step 5: Translations**

en identity map:

- `"What's New": "What's New"`
- `"What's New in %s": "What's New in %s"`
- `"This release has no notes.": "This release has no notes."`

de:

- `"What's New": "Neuigkeiten"`
- `"What's New in %s": "Neuigkeiten in %s"`
- `"This release has no notes.": "Diese Version hat keine Hinweise."`

- [ ] **Step 6: Verify**

Run from repo root:

```
make fmt-check
go vet ./...
go build ./...
go test -count=1 ./internal/update/ ./internal/preferences/ ./internal/ui/help/ ./internal/ui/settingswin/
go test -count=1 -run 'TestUpdate|TestCheckForUpdates|TestWhatsNew|TestTranslation|TestLocale' ./internal/ui/ .
```

Before handoff, the implementer (or parent) runs `go test -race ./...` as the full gate (`AGENTS.md`).

---

## Spec coverage (self-review)

| Requirement | Task |
|-------------|------|
| Config option, default off | 6 |
| Check once a day on first launch of that day; immediate check when enabled | 2 (`Due`), 7 |
| GitHub hashes (file SHA-256 + optional API `digest` + attestation subjects) | 3, 4 |
| GitHub signatures (immutable release attestation / GitHub Fulcio) | 4 |
| Background download | 7 |
| Apply on next close, no relaunch | 5, 7 |
| What's New with GitHub notes on next launch | 7 cache, 8 window |
| Unintrusive (no toast, no prompt, no menu spam) | 7–8 (explicit absences) |
| Three OSes | 2 asset map, 5 apply |
| ARCHITECTURE / lang / drain / no commit | 7 drain, 8 docs, Global Constraints |

## Out of scope

- Apple notarization / Authenticode (does not remove first-run Gatekeeper/SmartScreen).
- Homebrew / winget / Flathub publication.
- Auto-relaunch, update progress UI, "Check now", skipping a staged update in the UI.
- Updating `go run` / unpackaged builds (empty or invalid `Metadata.Version`).
- Applying over a non-writable install prefix: log and leave the stage for a later writable run; do not sudo.
- Proving the archive was built by `.github/workflows/release.yml` (a published zip still gets a GitHub release attestation).
