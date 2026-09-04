# Website maintenance

The public site is generated from [`../website.md`](../website.md). Edit that
file for English metadata, labels, links, media, repeated content, and prose. Do
not edit the generated HTML or German cache by hand.

Generated deployment artifacts are:

- `docs/index.html` — English regular page
- `docs/de/index.html` — German regular page
- `docs/amp/index.html` — English AMP page
- `docs/de/amp/index.html` — German AMP page
- `site/translations/de.json` — derived German translations

## One-time setup

Install Go 1.27 or newer, Node.js 20 or newer, Chrome, and the pinned Node tools:

```sh
npm ci
```

Only translation refreshes need a DeepL API key. Supply it in the process
environment:

```sh
export DEEPL_API_KEY='your-key'
```

Alternatively, create the ignored `.env.local` file:

```dotenv
DEEPL_API_KEY=your-key
```

Never commit either credential file, put the key in a Makefile argument, or paste
it into logs or review output.

## Edit and publish

After changing `website.md`, use the full transactional workflow:

```sh
make update
```

`make update` refreshes only missing or changed German units, removes obsolete
cache entries, generates all four pages in a staging directory, checks local
links, and validates both AMP documents. The cache and pages are published only
after every stage succeeds. A failed translation, build, or validation leaves the
previous derived files in place.

The narrower commands are useful while editing:

```sh
make translate       # networked; refresh German cache only
make build           # offline; generate all four pages from current inputs
make validate-amp    # offline; validate both committed AMP pages
make check           # offline; run tests and reject stale generated output
```

`make build` and `make check` deliberately fail if any German entry is missing or
stale; they never substitute English on a German page. Routine tests use a local
fake service and do not call DeepL.

Review the authored, translated, and generated changes together before pushing:

```sh
git diff -- website.md site/translations/de.json \
  docs/index.html docs/de/index.html \
  docs/amp/index.html docs/de/amp/index.html
make check
```

GitHub Pages continues to publish the committed files beneath `docs/` after the
branch is pushed; there is no separate deployment step here.

## Branch safety

Any branch whose name contains `website` or `webpage`, case-insensitively, is
publish-only relative to `main`. Push and publish it independently. Never merge
such a branch into `main`.
