# Static export

A gsx site whose pages are pure functions of build-time data can be served by
`net/http` in development and written to disk for static hosting, from one
list of pages. The pattern below is the `site` package from
[jackielii/hx-live-vs-alpine](https://github.com/jackielii/hx-live-vs-alpine):
a `Pages` list of paths to nodes that becomes an `http.Handler` for local development
and a directory tree for a static host, with one base-path helper that keeps
internal links correct in both places.

## Copy the helper

Copy this into the package that owns the page list:

```go
// Package site turns a list of pages into both an HTTP handler and a static
// export. A site is a Pages list of paths to gsx nodes; Handler serves it,
// Export writes it to disk as directory indexes. The same nodes render
// either way.
//
// Base-path plumbing lets one build serve from "/" locally and from
// "/<repo>/" on a project site: put the base in the context (Middleware or
// NewContext) and build every internal link with URL.
package site

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gsxhq/gsx"
)

// Page is one URL of the site and the node that renders it.
type Page struct {
	Path string // absolute, starts and ends with "/"
	Node gsx.Node
}

// Pages is the whole site in render order: a list, not a Go map. Handler
// renders each node concurrently across requests, and Export renders every
// node once per Export call while Handler may render the same node again at
// any time, so every node must be safe for concurrent, repeated Render
// calls.
type Pages []Page

func (m Pages) validate() error {
	seen := map[string]bool{}
	for _, p := range m {
		switch {
		case !strings.HasPrefix(p.Path, "/") || !strings.HasSuffix(p.Path, "/"):
			return fmt.Errorf("site: path %q must start and end with /", p.Path)
		case strings.Contains(p.Path, ".."):
			return fmt.Errorf("site: path %q must not contain ..", p.Path)
		case strings.Contains(p.Path, "//"):
			return fmt.Errorf("site: path %q must not contain //", p.Path)
		case strings.ContainsAny(p.Path, "{}"):
			return fmt.Errorf("site: path %q must not contain { or }", p.Path)
		case p.Node == nil:
			return fmt.Errorf("site: path %q has no node", p.Path)
		case seen[p.Path]:
			return fmt.Errorf("site: duplicate path %q", p.Path)
		}
		seen[p.Path] = true
	}
	return nil
}

// Handler serves every page at exactly its path with GET. A request for a
// page's path without its trailing slash is redirected to the slash form by
// the mux; any other path is a 404. Each page renders into a buffer first,
// so a render error produces a real 500 instead of a truncated 200 with a
// partial body already sent.
func (m Pages) Handler() (http.Handler, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	for _, p := range m {
		node := p.Node
		mux.HandleFunc("GET "+p.Path+"{$}", func(w http.ResponseWriter, r *http.Request) {
			var buf bytes.Buffer
			if err := node.Render(r.Context(), &buf); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(buf.Bytes())
		})
	}
	return mux, nil
}

// Export renders every page to dir/<path>/index.html. ctx must already carry
// whatever the nodes read (asset resolution, the base path); Export adds
// nothing. Existing files are overwritten; unrelated files are left alone.
// An empty list writes nothing, so dir is never created. A render error
// aborts the export mid-way, leaving the pages written so far in place.
func (m Pages) Export(ctx context.Context, dir string) error {
	if err := m.validate(); err != nil {
		return err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return fmt.Errorf("site: %q is not a directory", dir)
	}
	for _, p := range m {
		target := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(p.Path, "/")), "index.html")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("site: %w", err)
		}
		f, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("site: %w", err)
		}
		renderErr := p.Node.Render(ctx, f)
		closeErr := f.Close()
		if renderErr != nil {
			return fmt.Errorf("site: render %s: %w", p.Path, renderErr)
		}
		if closeErr != nil {
			return fmt.Errorf("site: write %s: %w", target, closeErr)
		}
	}
	return nil
}

type baseKey struct{}

// normalize gives base a leading and trailing slash; "" and "/" both mean "/".
func normalize(base string) string {
	base = strings.Trim(base, "/")
	if base == "" {
		return "/"
	}
	return "/" + base + "/"
}

// NewContext records the base path the site is mounted under.
func NewContext(ctx context.Context, base string) context.Context {
	return context.WithValue(ctx, baseKey{}, normalize(base))
}

// Base returns the mounted base path, "/" when none was set.
func Base(ctx context.Context) string {
	if b, ok := ctx.Value(baseKey{}).(string); ok {
		return b
	}
	return "/"
}

// URL prefixes path with the base path. path need not have a leading slash.
func URL(ctx context.Context, path string) string {
	return strings.TrimSuffix(Base(ctx), "/") + "/" + strings.TrimPrefix(path, "/")
}

// Middleware puts base into every request's context.
func Middleware(base string, next http.Handler) http.Handler {
	base = normalize(base)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), base)))
	})
}

// CopyFS copies fsys into dir, creating directories as needed and skipping
// any entry (and its subtree) for which skip returns true. Existing files
// are overwritten, and unlike os.CopyFS it does not preserve file modes:
// everything is written with the process's default permissions.
func CopyFS(dir string, fsys fs.FS, skip func(path string) bool) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return os.MkdirAll(dir, 0o755)
		}
		if skip != nil && skip(p) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		src, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			return err
		}
		return dst.Close()
	})
}
```

## Declare the site

List every page once, in application code:

```go
// buildMap lists every page of the site: the index and one frame per demo.
func buildMap(exs []examples.Example, feats []examples.FeatureRow) site.Pages {
	m := site.Pages{{Path: "/", Node: pages.Index(exs, feats)}}
	for _, ex := range exs {
		m = append(m, site.Page{Path: pages.FramePath(examples.Alpine, ex.Slug), Node: pages.Frame(examples.Alpine, ex)})
		if ex.HasDemo() {
			m = append(m, site.Page{Path: pages.FramePath(examples.HxLive, ex.Slug), Node: pages.Frame(examples.HxLive, ex)})
		}
	}
	return m
}
```

`pages.FramePath` builds a directory-style path so a static host serves it as
`<path>/index.html` without redirects:

```go
// FramePath is the site path of one demo frame. Directory-style so a static
// host serves it as <path>/index.html without redirects.
func FramePath(lib examples.Lib, slug string) string {
	return "/frame/" + string(lib) + "/" + slug + "/"
}
```

The same `Pages` list serves and exports. `newHandler` wraps it for `net/http`:

```go
// newHandler serves the site map plus the bundle and a health check, all
// mounted under base.
func newHandler(v *vite.Vite, m site.Pages, base string) (http.Handler, error) {
	pagesHandler, err := m.Handler()
	if err != nil {
		return nil, err
	}
	base = site.URL(site.NewContext(context.Background(), base), "/") // normalised, ends with /
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if !v.Dev() {
		mux.Handle(staticURL(base), v.StaticHandler())
	}
	if base == "/" {
		mux.Handle("/", pagesHandler)
	} else {
		mux.Handle(base, http.StripPrefix(strings.TrimSuffix(base, "/"), pagesHandler))
	}
	return site.Middleware(base, mux), nil
}
```

`export` renders it to disk instead, alongside the built bundle:

```go
// export writes the site and the bundle to dir for static hosting under
// base. bundle is the root of the built Vite output (the dist directory).
func export(ctx context.Context, v *vite.Vite, m site.Pages, base, dir string, bundle fs.FS) error {
	ctx = site.NewContext(vite.NewContext(ctx, v), base)
	if err := m.Export(ctx, dir); err != nil {
		return err
	}
	skip := func(p string) bool {
		return p == ".vite" || strings.HasPrefix(p, ".vite/") || p == ".gitkeep"
	}
	if err := site.CopyFS(filepath.Join(dir, "static"), bundle, skip); err != nil {
		return fmt.Errorf("copy bundle: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, ".nojekyll"), nil, 0o644)
}
```

Serving with a non-root `-base` mounts the bundle and the pages under that
prefix, so the live server and the export answer the same URLs.

## Link with the base path

Every internal link goes through `site.URL` instead of a literal path, so it
carries whatever base the request (or export) was built with:

```gsx
component column(lib examples.Lib, ex examples.Example) {
	{{ src := site.URL(ctx, FramePath(lib, ex.Slug)) }}
	<div class="flex min-w-0 flex-col gap-3">
		<iframe data-frame src={src} ...></iframe>
		...
	</div>
}
```

The base itself is carried in the request context by `site.Middleware`, which
`newHandler` wraps the whole mux in:

```go
return site.Middleware(base, mux), nil
```

`export` puts the same base into the export's context with `site.NewContext`
instead, since there is no request to carry it.

## Why the base path must reach Vite

Fonts referenced from `web/gsxui/fonts/fonts.css` end up in the built CSS as
`url(<base>assets/font.woff2)` — whatever base Vite was built with. The Go
side serves the bundle at `StaticURL`. Unless both are the same
`<siteBase>static/`, the font URLs in the CSS point somewhere the server
doesn't serve, and the fonts 404. `vite.config.ts` derives `base` from the
`SITE_BASE` environment variable so the two stay in lock step:

```js
  // The Go side serves the bundle at <base>static/ (vite.StaticURL). Vite must
  // write the same prefix into CSS url() references, so both read SITE_BASE.
  const siteBase = "/" + (process.env.SITE_BASE || "/").replace(/^\/+|\/+$/g, "");
  const assetBase = (siteBase === "/" ? "/" : siteBase + "/") + "static/";
```

The Go side derives `StaticURL` the same way, from the same base the site is
mounted under:

```go
// staticURL is where the Vite bundle is served relative to the site base.
// It must match Vite's `base` at build time (vite.config.ts reads SITE_BASE).
func staticURL(base string) string {
	return site.URL(site.NewContext(context.Background(), base), "/static/")
}
```

This is not automatic today: the `gsx init` template ships `base: "/"` in
`vite.config.ts` alongside `StaticURL "/static/"` in the Go config, and the
two never agree. The built CSS references `url(/assets/font.woff2)` (from
`base: "/"`), while the bundle is served at `/static/assets/` (from
`StaticURL "/static/"`) — so CSS-referenced assets (fonts, images) 404 in
production at any base, root included. The fix is to derive both from one
setting, `SITE_BASE`, as here: that gives `/static/` at the root and
`/<repo>/static/` on a project site, matching whatever `-base` the Go side
was started with.

## Publish

`npm run export` builds the bundle, regenerates routes, and runs the export
with whatever base the environment supplies:

```json
"export": "npm run build && go tool gsx generate && go run . -export out -base ${SITE_BASE:-/}"
```

`.github/workflows/pages.yml` runs that with `SITE_BASE` set to the project's
GitHub Pages path on every push to `main`, then publishes the result:

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: npm run export
        env:
          # Project sites live under /<repo>/; a user/org site or custom domain would use /
          SITE_BASE: /${{ github.event.repository.name }}/
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: out

  deploy:
    needs: build
    runs-on: ubuntu-latest
    permissions:
      pages: write
      id-token: write
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

## Test it like it will be served

`cmd/servestatic` serves an exported directory under a base path with
`http.StripPrefix`, standing in for whatever host will actually serve the
export. `e2e/playwright.config.ts` runs the Playwright suite against `go run
.` in development, at the root path. `e2e/static.config.ts` runs the same
specs against `cmd/servestatic` serving the export under the project's
GitHub Pages base, so the export is exercised the way Pages will serve it
before it ever reaches production.

## Where this goes next

The `site` package has no dependency on this repo's pages, examples, or
templates — it only knows about `gsx.Node` and a list of paths, so it can move
into gsx itself as-is. From there, content collections (a directory of
Markdown or data files turned into pages) and a `gsx build` command that walks
a `site.Pages` list and calls `Export` would turn this pattern into a small built-in
static-site generator, rather than something every project copies in by hand.
