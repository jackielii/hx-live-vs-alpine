# Static export and GitHub Pages deployment

**Date:** 2026-09-07
**Extends:** `2026-09-04-feature-catalogue-design.md` (merged as PR #1)
**Status:** approved design, pre-implementation

## Goal

Render the whole site to static files and deploy it to GitHub Pages as a
project site at `https://jackielii.github.io/hx-live-vs-alpine/`, from a
GitHub Actions workflow on every push to `main`. The Go server and `gsx dev`
keep working unchanged. The mechanism is built as a small, app-agnostic `site`
package so it can be contributed to gsx as a "static export" pattern, and so it
can grow into a static-site generator later.

Non-goals now: incremental builds, content collections, markdown pages, a CLI.
Those are the SSG future; this spec only makes sure nothing blocks them.

## Key facts

- The app has no dynamic requests: every page is a pure function of the
  manifest. The route set is finite and known at startup.
- GitHub Pages project sites live under `/<repo>/`. Every page URL, iframe
  `src`, and asset URL needs that prefix. The prefix must be known at Vite
  build time too, because Vite writes absolute asset URLs into CSS (fonts).
- Today's prod build already has a latent bug: Vite builds with `base: "/"`
  and emits `url(/assets/font.woff2)` while the Go server mounts the bundle at
  `/static/`, so the gsxui fonts 404 in prod. Aligning Vite's `base` with the
  vite package's `StaticURL` fixes it for both the server and the export.

## Architecture

**A site is a list of pages.** A page is a path and a `gsx.Node`. The server
is a generic handler over that list; the exporter walks the same list to
disk. One rendering path, two sinks.

```
examples manifest ──► site.Map ──► Handler()  ──► net/http (dev/prod server)
                             └───► Export()   ──► out/**/index.html (+ static assets)
```

### Package `site` (`site/site.go`, this repo, contribution candidate)

```go
package site

// Page is one URL of the site and the node that renders it.
type Page struct {
    Path string   // absolute, starts with "/", ends with "/" (directory-style)
    Node gsx.Node
}

// Map is the whole site in render order.
type Map []Page

// Handler serves every page at exactly its path (GET only, 404 otherwise).
func (m Map) Handler() http.Handler

// Export renders every page to dir/<path>/index.html. ctx must already
// carry whatever the nodes need (vite, base); Export adds nothing.
func (m Map) Export(ctx context.Context, dir string) error

// Base-path plumbing: the prefix the site is mounted under ("/" or "/repo/").
func NewContext(ctx context.Context, base string) context.Context
func Base(ctx context.Context) string             // "" when unset → "/"
func URL(ctx context.Context, path string) string  // Base + path, no double slash
func Middleware(base string, next http.Handler) http.Handler // NewContext per request

// CopyFS copies fsys into dir, skipping entries for which skip returns true.
func CopyFS(dir string, fsys fs.FS, skip func(path string) bool) error
```

Rules: `Path` values are validated in `Handler`/`Export` (leading and trailing
slash, no `..`); duplicates are an error. `Export` refuses a `dir` that is a
file, creates it if missing, and overwrites files inside it (it does not
delete unrelated files; the workflow always starts from an empty checkout).
Any render error aborts the export with a non-zero exit.

### App wiring (`main.go`)

- `buildMap(exs) site.Map`: `/` → `pages.Index(exs, feats)`; for every demo row, `/frame/alpine/<slug>/` and, when `HasDemo()`, `/frame/hxlive/<slug>/` → `pages.Frame(lib, ex)`. Paths gain a trailing slash in both modes so directory-index hosting needs no redirects.
- Flags: `-base` (default `/`), `-export DIR`. Serving: `vite.New` with `StaticURL: base + "static/"`, handler = `vite.Middleware(site.Middleware(base, mux))` where `mux` mounts `/static/` (prod), `/healthz`, and `siteMap.Handler()`. Exporting: `ctx = site.NewContext(vite.NewContext(ctx, v), base)`; `siteMap.Export(ctx, dir)`; `site.CopyFS(dir+"/static", distFS sub "dist", skip .vite/ and .gitkeep)`; write `dir/.nojekyll`.
- `pages/index.gsx`: the iframe `src` becomes `site.URL(ctx, "/frame/"+lib+"/"+slug+"/")`. Nothing else links internally (anchors are `#slug`; asset URLs already come from `vite.Entry`, which prefixes `StaticURL`).
- Remove `/public/` and its embed (unused scaffold residue; the export would otherwise need to copy it too).

### Vite (`vite.config.ts`, `package.json`)

- `const siteBase = withSlashes(process.env.SITE_BASE || "/")`; `base: command === "serve" ? "/__vite/" : siteBase + "static/"`. Manifest keys stay source paths; `file` values stay relative; only URLs written into CSS/JS change.
- Scripts: `"export": "vite build && go tool gsx generate && go run . -export out -base ${SITE_BASE:-/}"`, plus `"e2e:static": "playwright test -c e2e/static.config.ts"`. `out/` is gitignored.

### Deployment (`.github/workflows/pages.yml`)

On `push` to `main` and `workflow_dispatch`: checkout; setup-go from `go.mod`; setup-node 22 with npm cache; `npm ci`; `SITE_BASE=/hx-live-vs-alpine/ npm run export`; `actions/configure-pages`; `actions/upload-pages-artifact` with `path: out`; `actions/deploy-pages` in a `github-pages` environment with `pages: write` and `id-token: write`. One-time repo setting: Pages → Source → GitHub Actions.

### Output layout

```
out/
  .nojekyll
  index.html
  frame/alpine/<slug>/index.html      (28)
  frame/hxlive/<slug>/index.html      (23)
  static/assets/*                      (hashed JS, CSS, fonts)
```

## Verification

- **Go tests (`site/site_test.go`)**: `URL` joins for base `/`, `/repo/`, `/repo` (normalised); `Handler` serves each page, 404s a non-page, and redirects a page path missing its trailing slash; `Export` into a temp dir writes `index.html` and `a/b/index.html` with the rendered bodies; validation rejects a path without slashes, `..`, and duplicates; `CopyFS` honours `skip`.
- **Go tests (`main_test.go`)**: `buildMap` yields 1 + 28 + 23 paths, none for `hxlive` on `none` rows; existing route tests move to trailing-slash paths; exporting the real map with base `/repo/` to a temp dir produces the layout above, the index's iframe srcs start with `/repo/frame/`, and asset URLs start with `/repo/static/assets/`.
- **Playwright**: all specs switch to base-relative navigation (`page.goto("./")`, `./frame/${lib}/counter/`) so one spec set runs under two configs: `e2e/playwright.config.ts` against the Go server on 8899 as today, and `e2e/static.config.ts` whose webServer runs `SITE_BASE=/hx-live-vs-alpine/ npm run export` then `go run ./cmd/servestatic -dir out -base /hx-live-vs-alpine/ -port 8898` (a 25-line `http.StripPrefix` file server) with `baseURL: http://localhost:8898/hx-live-vs-alpine/`. Both must pass 48/48. Fonts are checked by the static index spec: the main stylesheet's first `url(` font resolves to 200.
- **Deployed check**: after the first workflow run, open the Pages URL and the counter frame under it.

## Contribution back to gsx

Deliverable in this repo: `docs/pattern/static-export.md`, written in the style of `gsx/docs/guide/patterns/render-once.md` ("Copy the helper", then the wiring, then the workflow), with the `site` package as the helper and a short "Why the base path must reach Vite" section covering the font bug. The `site` package stays generic (no imports from this app) so it can be lifted into `github.com/gsxhq/gsx/site` or a gsx-examples entry when the pattern is accepted. Follow-ups noted, not done here: a fix to the `gsx init` template's Vite `base` so fonts work under `/static/` out of the box, and the SSG direction (content collections, a `gsx build` command over a `site.Map`).

## Error handling

Export: any render error, unwritable output, or missing `dist/.vite/manifest.json` exits non-zero with the path that failed. Server: unchanged (404 for unknown paths, 500 on render error).
