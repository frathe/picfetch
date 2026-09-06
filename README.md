# Website maintenance

The public site is generated from [`../website.md`](../website.md). Edit that
file for English metadata, labels, links, media, repeated content, and prose. Do
not edit the generated HTML or German cache by hand.

Generated deployment artifacts are:

- `docs/index.html` — English regular page
- `docs/de/index.html` — German regular page
- `docs/amp/index.html` — English AMP page
- `docs/de/amp/index.html` — German AMP page
- `docs/sitemap.xml` — preferred English and German regular URLs
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

Every generated page must have exactly one canonical link in its HTML head,
pointing to its language's regular URL. Builds reject missing, duplicate, or
incorrect canonical links before replacing any published files. The sitemap is
generated from `site.base_url` alongside the pages, and `make check` rejects a
missing or stale sitemap. Template or generator changes that do not change
translated content can use `make build` followed by `make check` offline.

Review the authored, translated, and generated changes together before pushing:

```sh
git diff -- website.md site/translations/de.json \
  docs/index.html docs/de/index.html \
  docs/amp/index.html docs/de/amp/index.html docs/sitemap.xml
make check
```

GitHub Pages continues to publish the committed files beneath `docs/` after the
branch is pushed; there is no separate deployment step here.

## Search indexing

The preferred indexed pages are `https://frathe.github.io/picfetch/` and
`https://frathe.github.io/picfetch/de/`. AMP pages and `index.html` aliases declare
the corresponding regular URL as canonical. The sitemap contains only the two
regular URLs; AMP discovery and language alternates remain in the HTML.

After publishing, submit `https://frathe.github.io/picfetch/sitemap.xml` in the
Search Console property `https://frathe.github.io/picfetch/`. Inspect the regular
URLs, run the live test, and request indexing. For a duplicate warning, compare
the last crawl date, user-declared canonical, and Google-selected canonical.
Google may still report an older canonical choice while it reprocesses the site.
Confirm the preferred URLs become indexed; duplicate aliases need not be indexed.

The separate `frathe/frathe.github.io` repository owns the host homepage and
redirects it to `/picfetch/`. Preserve its matching canonical and immediate
redirect. If Google selects the host homepage instead, inspect that page too.
A `docs/robots.txt` here would be served at `/picfetch/robots.txt`, whereas crawlers
read robots rules only at `https://frathe.github.io/robots.txt`. Sitemap submission
works without adding a robots file to this project. `.gitignore` does not control
search indexing.

See Google's [canonical URL guidance](https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls)
and [Page indexing report documentation](https://support.google.com/webmasters/answer/7440203).

## Branch safety

Any branch whose name contains `website` or `webpage`, case-insensitively, is
publish-only relative to `main`. Push and publish it independently. Never merge
such a branch into `main`.
