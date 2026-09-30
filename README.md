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

`frathe/frathe.github.io` publishes the live site. Its Pages source is the
`main` branch `docs/` directory, which GitHub serves at `https://frathe.github.io/`.
Pushing this `website` branch does not publish the site. After `make build` and
`make check`, copy the generated `docs/` tree into that repository's `docs/`
directory, keep `docs/robots.txt` there, and push `main`.

The old project URL `https://frathe.github.io/picfetch/` stays on a separate
`legacy-picfetch-redirect` branch of `frathe/picfetch`. That branch is the
picfetch Pages source and only redirects the former pages, AMP pages, and sitemap
to the root site. Do not point picfetch Pages back at this `website` branch.

## Search indexing

The preferred indexed pages are `https://frathe.github.io/` and
`https://frathe.github.io/de/`. AMP pages and `index.html` aliases declare
the corresponding regular URL as canonical. The sitemap contains only the two
regular URLs; AMP discovery and language alternates remain in the HTML.

After publishing, submit `https://frathe.github.io/sitemap.xml` in the
Search Console property `https://frathe.github.io/`. Inspect the regular
URLs, run the live test, and request indexing. For a duplicate warning, compare
the last crawl date, user-declared canonical, and Google-selected canonical.
Google may still report an older canonical choice while it reprocesses the site.
Confirm the preferred URLs become indexed; duplicate aliases need not be indexed.

Crawlers read robots rules only at `https://frathe.github.io/robots.txt`. That
file lives in the user-site repository and points at the root sitemap. The
legacy `/picfetch/` URLs redirect to these canonical pages. `.gitignore` does not
control search indexing.

See Google's [canonical URL guidance](https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls)
and [Page indexing report documentation](https://support.google.com/webmasters/answer/7440203).

## Branch safety

Any branch whose name contains `website` or `webpage`, case-insensitively, is
publish-only relative to `main`. Push and publish it independently. Never merge
such a branch into `main`.
