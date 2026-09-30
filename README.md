# Website

PicFetch's public site is authored and published from
[frathe/frathe.github.io](https://github.com/frathe/frathe.github.io).
Edit `website.md` there, run `make build` or `make update`, then push `main`.
GitHub Pages serves that repository's `docs/` directory at
https://frathe.github.io/.

This branch no longer contains the site generator. It remains until crawlers
stop requesting https://frathe.github.io/picfetch/. That address is served by
the `legacy-picfetch-redirect` branch, which redirects to the root site.

Do not merge this branch into `main`.
