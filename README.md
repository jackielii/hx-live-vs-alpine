# hx-live vs Alpine

Every Alpine.js core feature (18 directives, 9 magics, 3 globals) beside its
htmx 4 `hx-live` counterpart: the Alpine docs example on the left, a
like-for-like port on the right where one exists, and an honest note where
one does not. A matrix at the top of the page gives each feature a status:
`equivalent`, `workaround`, or `none`. Raw material for a blog post.

Every demo runs in its own iframe that loads exactly one library, because
Alpine and hx-live both claim the `:attr` shorthand and hx-live disables its
short form when it detects Alpine. The hx-live ports target htmx 4.0.0's typed
state bags (`data.*`, `aria.*`, `class.*`).

## Run

    npm install
    npm run dev          # gsx dev: Vite + Go with reload

## Test

    npm install && npm run build            # once, so dist/ exists for go:embed
    go tool gsx generate && go test ./...   # routes and manifest
    npm run e2e                             # Playwright parity, both libraries
    npm run e2e:static  # same suite against the export

## Deploy

The site is static. `npm run export` renders every page to `out/` and copies
the bundle to `out/static/`; `SITE_BASE=/hx-live-vs-alpine/ npm run export`
prefixes every URL for a GitHub Pages project site. `.github/workflows/pages.yml`
does that on each push to `main` and publishes `out/` with `actions/deploy-pages`
(repo setting: Pages → Source → GitHub Actions).

`npm run e2e:static` runs the same Playwright suite against the exported
directory served under the base path, the way Pages serves it.

## Layout

- `examples/<slug>/alpine.html`, `notes.md`, and `hxlive.html` for rows with a port: the content. `examples/examples.go` holds the manifest (group, features, status).
- `pages/frame.gsx`: the single-library iframe document.
- `pages/index.gsx`: the comparison page.
- `web/frame-alpine.js`, `web/frame-hxlive.js`: one Vite entry per library.
- `site/`: the page map that both serves and exports (candidate gsx pattern).

Pinned: `htmx.org@4.0.0`, `alpinejs@3.17.1`.

## If you like this, see also my other projects

- [gsx](https://github.com/gsxhq/gsx): the JSX-like templating language for Go this site is built with
- [gsxui](https://github.com/gsxhq/gsxui): shadcn-style components for gsx, used for the page chrome here
- [structpages](https://github.com/jackielii/structpages): struct-based routing for Go web apps with gsx/templ and htmx
