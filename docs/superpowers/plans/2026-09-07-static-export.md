# Static Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render the whole site to static files under a configurable base path and deploy it to GitHub Pages from a workflow, through a generic `site` package that can be contributed to gsx as a pattern.

**Architecture:** A `site.Map` is a list of `(path, gsx.Node)` pages. The Go server serves the map through `Map.Handler()`; the exporter writes the same map to `out/<path>/index.html` through `Map.Export()` and copies the Vite bundle to `out/static/`. A base path reaches both the Go side (`site.URL`, vite `StaticURL`) and Vite (`base`), so every URL, including font URLs in CSS, carries the `/hx-live-vs-alpine/` prefix on Pages.

**Tech Stack:** unchanged plus `os.CopyFS`-style copying in Go 1.26, `actions/deploy-pages`.

**Spec:** `docs/superpowers/specs/2026-09-07-static-export-design.md`

## Global Constraints

- Package `site` imports nothing from this app (only stdlib and `github.com/gsxhq/gsx`); it must be liftable into gsx unchanged.
- Page paths start and end with `/`. Frame pages are `/frame/<lib>/<slug>/` in both server and export modes.
- Base path is normalised to leading and trailing slash; `/` means none. `site.URL(ctx, "/x/")` with base `/repo/` is `/repo/x/`; with base `/` it is `/x/`.
- Vite builds with `base = <siteBase>static/`; the vite package gets `StaticURL = <siteBase>static/`. Both read the same `SITE_BASE`.
- The outer page still imports neither Alpine nor hx-live; shown source stays escaped; the frame isolation is untouched.
- One Playwright spec set runs under two configs and must pass 48/48 under both; assertions are never weakened.
- No generated output (`out/`, `dist/*`) is committed. Never leave a server on 8898 or 8899.
- Unexported identifiers unless crossing a package boundary. Commit messages end with `Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb`.
- Every task ends with `go tool gsx generate && go vet ./... && go test ./...` green.

---

## File structure

| Path | Responsibility |
|---|---|
| `site/site.go` | `Page`, `Map`, `Handler`, `Export`, base-path context, `Middleware`, `CopyFS` |
| `site/site_test.go` | unit tests with a `gsx.Raw` node and `fstest.MapFS` |
| `pages/paths.go` | `FramePath(lib, slug)` shared by the map builder and the index page |
| `main.go` | flags, `buildMap`, `newHandler(v, m, base)`, `export`, `main` |
| `main_test.go` | route tests on trailing-slash paths; export test |
| `pages/index.gsx` | iframe src via `site.URL` |
| `cmd/servestatic/main.go` | file server for the exported directory under a base path |
| `vite.config.ts`, `package.json`, `.gitignore` | base-aware build, `export` and `e2e:static` scripts, ignore `out/` |
| `e2e/*.spec.ts`, `e2e/static.config.ts` | base-relative navigation; second config against the export |
| `.github/workflows/pages.yml` | build, export, deploy |
| `README.md`, `docs/pattern/static-export.md` | usage; the contribution draft |

---

### Task 1: The `site` package

**Files:**
- Create: `site/site.go`, `site/site_test.go`

**Interfaces:**
- Produces:
  ```go
  package site
  type Page struct { Path string; Node gsx.Node }
  type Map []Page
  func (m Map) Handler() (http.Handler, error)
  func (m Map) Export(ctx context.Context, dir string) error
  func NewContext(ctx context.Context, base string) context.Context
  func Base(ctx context.Context) string
  func URL(ctx context.Context, path string) string
  func Middleware(base string, next http.Handler) http.Handler
  func CopyFS(dir string, fsys fs.FS, skip func(path string) bool) error
  ```

- [ ] **Step 1: Write the failing tests**

`site/site_test.go`:
```go
package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gsxhq/gsx"
)

func page(p, body string) Page { return Page{Path: p, Node: gsx.Raw(body)} }

func TestURLJoinsBaseAndPath(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"/", "/", "/"},
		{"/", "/frame/a/", "/frame/a/"},
		{"/repo/", "/frame/a/", "/repo/frame/a/"},
		{"/repo", "/frame/a/", "/repo/frame/a/"},
		{"repo", "/", "/repo/"},
		{"", "/x/", "/x/"},
	}
	for _, c := range cases {
		ctx := NewContext(context.Background(), c.base)
		if got := URL(ctx, c.path); got != c.want {
			t.Errorf("base %q path %q: got %q, want %q", c.base, c.path, got, c.want)
		}
	}
	if Base(context.Background()) != "/" {
		t.Error("unset base should read as /")
	}
}

func TestHandlerServesExactPathsOnly(t *testing.T) {
	m := Map{page("/", "<h1>home</h1>"), page("/a/b/", "<p>ab</p>")}
	h, err := m.Handler()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	for path, want := range map[string]int{"/": 200, "/a/b/": 200, "/nope/": 404, "/a/": 404} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Errorf("%s: status %d, want %d", path, res.StatusCode, want)
		}
	}
	res, _ := http.Get(srv.URL + "/a/b/")
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content type %q", ct)
	}
	res.Body.Close()
}

func TestMapValidation(t *testing.T) {
	bad := []Map{
		{page("a/", "x")},
		{page("/a", "x")},
		{page("/../a/", "x")},
		{page("/a/", "x"), page("/a/", "y")},
		{{Path: "/a/", Node: nil}},
	}
	for i, m := range bad {
		if _, err := m.Handler(); err == nil {
			t.Errorf("case %d: Handler accepted invalid map", i)
		}
		if err := m.Export(context.Background(), t.TempDir()); err == nil {
			t.Errorf("case %d: Export accepted invalid map", i)
		}
	}
}

func TestExportWritesDirectoryIndexes(t *testing.T) {
	dir := t.TempDir()
	m := Map{page("/", "<h1>home</h1>"), page("/a/b/", "<p>ab</p>")}
	if err := m.Export(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{"index.html": "<h1>home</h1>", "a/b/index.html": "<p>ab</p>"} {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != want {
			t.Errorf("%s: %q, want %q", rel, b, want)
		}
	}
	file := filepath.Join(dir, "index.html")
	if err := m.Export(context.Background(), file); err == nil {
		t.Error("Export into a file path should fail")
	}
}

func TestMiddlewareSetsBase(t *testing.T) {
	var got string
	h := Middleware("/repo/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = URL(r.Context(), "/x/")
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got != "/repo/x/" {
		t.Errorf("got %q", got)
	}
}

func TestCopyFSSkips(t *testing.T) {
	src := fstest.MapFS{
		"assets/a.js":         {Data: []byte("a")},
		".vite/manifest.json": {Data: []byte("{}")},
		".gitkeep":            {Data: []byte("")},
	}
	dir := t.TempDir()
	skip := func(p string) bool { return p == ".vite" || strings.HasPrefix(p, ".vite/") || p == ".gitkeep" }
	if err := CopyFS(dir, src, skip); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "assets", "a.js")); err != nil || string(b) != "a" {
		t.Errorf("a.js not copied: %v %q", err, b)
	}
	for _, p := range []string{".vite/manifest.json", ".gitkeep"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err == nil {
			t.Errorf("%s should have been skipped", p)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./site/
```
Expected: build failure (package has no non-test file).

- [ ] **Step 3: Write `site/site.go`**

```go
// Package site turns a list of pages into both an HTTP handler and a static
// export. A site is a Map of paths to gsx nodes; Handler serves it, Export
// writes it to disk as directory indexes. The same nodes render either way.
//
// Base-path plumbing lets one build serve from "/" locally and from
// "/<repo>/" on a project site: put the base in the context (Middleware or
// NewContext) and build every internal link with URL.
package site

import (
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

// Map is the whole site in render order.
type Map []Page

func (m Map) validate() error {
	seen := map[string]bool{}
	for _, p := range m {
		switch {
		case !strings.HasPrefix(p.Path, "/") || !strings.HasSuffix(p.Path, "/"):
			return fmt.Errorf("site: path %q must start and end with /", p.Path)
		case strings.Contains(p.Path, ".."):
			return fmt.Errorf("site: path %q must not contain ..", p.Path)
		case p.Node == nil:
			return fmt.Errorf("site: path %q has no node", p.Path)
		case seen[p.Path]:
			return fmt.Errorf("site: duplicate path %q", p.Path)
		}
		seen[p.Path] = true
	}
	return nil
}

// Handler serves every page at exactly its path with GET. Anything else is a
// 404 from the mux.
func (m Map) Handler() (http.Handler, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	for _, p := range m {
		node := p.Node
		mux.HandleFunc("GET "+p.Path+"{$}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := node.Render(r.Context(), w); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	}
	return mux, nil
}

// Export renders every page to dir/<path>/index.html. ctx must already carry
// whatever the nodes read (asset resolution, the base path); Export adds
// nothing. Existing files are overwritten; unrelated files are left alone.
func (m Map) Export(ctx context.Context, dir string) error {
	if err := m.validate(); err != nil {
		return err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return fmt.Errorf("site: %s is not a directory", dir)
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

// URL prefixes an absolute site path with the base path.
func URL(ctx context.Context, path string) string {
	return strings.TrimSuffix(Base(ctx), "/") + path
}

// Middleware puts base into every request's context.
func Middleware(base string, next http.Handler) http.Handler {
	base = normalize(base)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), base)))
	})
}

// CopyFS copies fsys into dir, creating directories as needed and skipping
// any entry (and its subtree) for which skip returns true.
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

- [ ] **Step 4: Run tests to verify they pass**

```bash
go vet ./site/ && go test ./site/
```
Expected: `ok`. If `gsx.Raw(body).Render` writes something other than the exact body (for example a trailing newline), adjust the test's `want` to `strings.TrimSpace` both sides and note it in the report; do not change the package.

- [ ] **Step 5: Commit**

```bash
git add site
git commit -m "Add site package: a page map that serves and exports

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 2: Wire the app to the site map, with export

**Files:**
- Create: `pages/paths.go`
- Modify: `main.go`, `main_test.go`, `pages/index.gsx`
- Delete: `public/` directory

**Interfaces:**
- Consumes: everything from Task 1.
- Produces: `pages.FramePath(lib examples.Lib, slug string) string` (returns `/frame/<lib>/<slug>/`); `buildMap(exs, feats) site.Map`; `newHandler(v *vite.Vite, m site.Map, base string) (http.Handler, error)`; `export(ctx, v, m, base, dir) error`; flags `-base`, `-export`.

- [ ] **Step 1: Write the failing tests**

Replace `main_test.go` entirely:

```go
package main

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/pages"
)

// testManifest mimics dist/.vite/manifest.json after `vite build` so the
// handler runs in prod mode without Vite.
const testManifest = `{
  "web/main.js":         {"file": "assets/main.js", "isEntry": true, "css": ["assets/main.css"]},
  "web/frame-alpine.js": {"file": "assets/frame-alpine.js", "isEntry": true, "css": ["assets/frame.css"]},
  "web/frame-hxlive.js": {"file": "assets/frame-hxlive.js", "isEntry": true, "css": ["assets/frame.css"]}
}`

func testVite(t *testing.T, base string) *vite.Vite {
	t.Helper()
	fsys := fstest.MapFS{"dist/.vite/manifest.json": {Data: []byte(testManifest)}}
	v, err := vite.New(vite.Config{Dist: fsys, DistDir: "dist", StaticURL: staticURL(base)})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func loadMap(t *testing.T) ([]examples.Example, site.Map) {
	t.Helper()
	exs, err := examples.Load()
	if err != nil {
		t.Fatal(err)
	}
	feats, err := examples.Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	return exs, buildMap(exs, feats)
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	v := testVite(t, "/")
	_, m := loadMap(t)
	h, err := newHandler(v, m, "/")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(v.Middleware(h))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, srv *httptest.Server, path string) (int, string) {
	t.Helper()
	res, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func TestBuildMapListsEveryPage(t *testing.T) {
	exs, m := loadMap(t)
	paths := map[string]bool{}
	for _, p := range m {
		paths[p.Path] = true
	}
	if !paths["/"] {
		t.Error("map lacks /")
	}
	demo := 0
	for _, ex := range exs {
		if !paths[pages.FramePath(examples.Alpine, ex.Slug)] {
			t.Errorf("%s: no alpine page", ex.Slug)
		}
		hx := paths[pages.FramePath(examples.HxLive, ex.Slug)]
		if hx != ex.HasDemo() {
			t.Errorf("%s: hxlive page present=%v, HasDemo=%v", ex.Slug, hx, ex.HasDemo())
		}
		if ex.HasDemo() {
			demo++
		}
	}
	if want := 1 + len(exs) + demo; len(m) != want {
		t.Errorf("map has %d pages, want %d", len(m), want)
	}
}

func TestFrameServesOneLibraryAndTheRawFragment(t *testing.T) {
	srv := newTestServer(t)
	exs, _ := examples.Load()
	counter, _ := examples.Find(exs, "counter")

	status, body := get(t, srv, "/frame/hxlive/counter/")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if !strings.Contains(body, counter.HxLive) {
		t.Error("hxlive frame does not contain the raw hxlive fragment")
	}
	if !strings.Contains(body, "/static/assets/frame-hxlive.js") {
		t.Error("hxlive frame does not load the hxlive entry")
	}
	if strings.Contains(body, "frame-alpine") || strings.Contains(body, "main.js") {
		t.Error("hxlive frame loads another bundle")
	}

	status, body = get(t, srv, "/frame/alpine/counter/")
	if status != 200 || !strings.Contains(body, counter.Alpine) || !strings.Contains(body, "/static/assets/frame-alpine.js") {
		t.Errorf("alpine frame wrong: status %d", status)
	}
	if strings.Contains(body, "frame-hxlive") {
		t.Error("alpine frame loads hxlive")
	}
}

func TestFrameUnknownIs404(t *testing.T) {
	srv := newTestServer(t)
	for _, p := range []string{"/frame/react/counter/", "/frame/alpine/nope/", "/frame/hxlive/teleport/", "/frame/alpine/"} {
		if status, _ := get(t, srv, p); status != 404 {
			t.Errorf("%s: status %d, want 404", p, status)
		}
	}
}

func TestIndexListsEveryExampleWithBothSources(t *testing.T) {
	srv := newTestServer(t)
	exs, _ := examples.Load()
	status, body := get(t, srv, "/")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if !strings.Contains(body, "/static/assets/main.js") {
		t.Error("index does not load the outer bundle")
	}
	for _, ex := range exs {
		if !strings.Contains(body, `id="`+ex.Slug+`"`) {
			t.Errorf("%s: no anchor", ex.Slug)
		}
		libs := []examples.Lib{examples.Alpine}
		if ex.HasDemo() {
			libs = append(libs, examples.HxLive)
		}
		for _, lib := range libs {
			if !strings.Contains(body, `src="`+pages.FramePath(lib, ex.Slug)+`"`) {
				t.Errorf("%s: no iframe for %s", ex.Slug, lib)
			}
			escaped := html.EscapeString(ex.Fragment(lib))
			if !strings.Contains(body, escaped) {
				t.Errorf("%s: escaped %s fragment not shown", ex.Slug, lib)
			}
		}
		if !ex.HasDemo() && strings.Contains(body, `src="`+pages.FramePath(examples.HxLive, ex.Slug)+`"`) {
			t.Errorf("%s: none row has an hxlive iframe", ex.Slug)
		}
		if ex.HxLive != "" && strings.Contains(body, ex.HxLive) {
			t.Errorf("%s: raw hxlive fragment leaked unescaped into the index", ex.Slug)
		}
	}
}

func TestExportWritesTheSiteUnderABase(t *testing.T) {
	v := testVite(t, "/repo/")
	exs, m := loadMap(t)
	dir := t.TempDir()
	if err := export(context.Background(), v, m, "/repo/", dir); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"index.html", ".nojekyll", "frame/alpine/counter/index.html", "frame/hxlive/counter/index.html", "frame/alpine/teleport/index.html"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "frame", "hxlive", "teleport")); err == nil {
		t.Error("none row got an hxlive page")
	}
	index, _ := os.ReadFile(filepath.Join(dir, "index.html"))
	body := string(index)
	if !strings.Contains(body, `src="/repo/frame/hxlive/counter/"`) {
		t.Error("index iframe src lacks the base prefix")
	}
	if !strings.Contains(body, `/repo/static/assets/main.js`) {
		t.Error("index asset URL lacks the base prefix")
	}
	if strings.Contains(body, `src="/frame/`) {
		t.Error("index still has an unprefixed frame src")
	}
	count := 0
	for _, ex := range exs {
		count++
		if ex.HasDemo() {
			count++
		}
	}
	got := 0
	filepath.WalkDir(filepath.Join(dir, "frame"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "index.html" {
			got++
		}
		return nil
	})
	if got != count {
		t.Errorf("exported %d frame pages, want %d", got, count)
	}
}
```
Add `"github.com/jackielii/hx-live-vs-alpine/site"` to the imports.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go tool gsx generate && go test .
```
Expected: build failure on `buildMap`, `staticURL`, `export`, `pages.FramePath`.

- [ ] **Step 3: Add `pages/paths.go`**

```go
package pages

import "github.com/jackielii/hx-live-vs-alpine/examples"

// FramePath is the site path of one demo frame. Directory-style so a static
// host serves it as <path>/index.html without redirects.
func FramePath(lib examples.Lib, slug string) string {
	return "/frame/" + string(lib) + "/" + slug + "/"
}
```

- [ ] **Step 4: Point the iframe at the base-aware URL**

In `pages/index.gsx`, add `"github.com/jackielii/hx-live-vs-alpine/site"` to the imports and change the `column` component's first line from

```gsx
	{{ src := "/frame/" + string(lib) + "/" + ex.Slug }}
```
to
```gsx
	{{ src := site.URL(ctx, FramePath(lib, ex.Slug)) }}
```

- [ ] **Step 5: Rewrite `main.go`**

```go
package main

import (
	"cmp"
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/pages"
	"github.com/jackielii/hx-live-vs-alpine/site"
)

//go:embed all:dist
var distFS embed.FS

// staticURL is where the Vite bundle is served relative to the site base.
// It must match Vite's `base` at build time (vite.config.ts reads SITE_BASE).
func staticURL(base string) string {
	return site.URL(site.NewContext(context.Background(), base), "/static/")
}

// buildMap lists every page of the site: the index and one frame per demo.
func buildMap(exs []examples.Example, feats []examples.FeatureRow) site.Map {
	m := site.Map{{Path: "/", Node: pages.Index(exs, feats)}}
	for _, ex := range exs {
		m = append(m, site.Page{Path: pages.FramePath(examples.Alpine, ex.Slug), Node: pages.Frame(examples.Alpine, ex)})
		if ex.HasDemo() {
			m = append(m, site.Page{Path: pages.FramePath(examples.HxLive, ex.Slug), Node: pages.Frame(examples.HxLive, ex)})
		}
	}
	return m
}

// newHandler serves the site map plus the bundle and a health check.
func newHandler(v *vite.Vite, m site.Map, base string) (http.Handler, error) {
	pagesHandler, err := m.Handler()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if !v.Dev() {
		mux.Handle("/static/", v.StaticHandler())
	}
	mux.Handle("/", pagesHandler)
	return site.Middleware(base, mux), nil
}

// export writes the site and the bundle to dir for static hosting under base.
func export(ctx context.Context, v *vite.Vite, m site.Map, base, dir string) error {
	ctx = site.NewContext(vite.NewContext(ctx, v), base)
	if err := m.Export(ctx, dir); err != nil {
		return err
	}
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}
	skip := func(p string) bool {
		return p == ".vite" || strings.HasPrefix(p, ".vite/") || p == ".gitkeep"
	}
	if err := site.CopyFS(filepath.Join(dir, "static"), dist, skip); err != nil {
		return fmt.Errorf("copy bundle: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, ".nojekyll"), nil, 0o644)
}

func main() {
	base := flag.String("base", "/", "path the site is mounted under, e.g. /hx-live-vs-alpine/")
	exportDir := flag.String("export", "", "write the site as static files to this directory and exit")
	flag.Parse()

	devURL := os.Getenv("VITE_DEV_URL") // "" in prod
	v, err := vite.New(vite.Config{DevURL: devURL, DevBase: "/__vite/", Dist: distFS, DistDir: "dist", StaticURL: staticURL(*base)})
	if err != nil {
		log.Fatal(err)
	}
	exs, err := examples.Load()
	if err != nil {
		log.Fatal(err)
	}
	feats, err := examples.Features(exs)
	if err != nil {
		log.Fatal(err)
	}
	m := buildMap(exs, feats)

	if *exportDir != "" {
		if v.Dev() {
			log.Fatal("export needs a production bundle: unset VITE_DEV_URL and run `npm run build` first")
		}
		if err := export(context.Background(), v, m, *base, *exportDir); err != nil {
			log.Fatal(err)
		}
		log.Printf("exported %d pages to %s (base %s)", len(m), *exportDir, *base)
		return
	}

	h, err := newHandler(v, m, *base)
	if err != nil {
		log.Fatal(err)
	}
	port := cmp.Or(os.Getenv("GO_PORT"), "7777")
	srv := &http.Server{Addr: ":" + port, Handler: v.Middleware(h)}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	log.Printf("listening on http://localhost:%s", port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
```

Delete the `public/` directory (`git rm -r public`); its embed is gone.

- [ ] **Step 6: Run tests to verify they pass**

```bash
go tool gsx generate && go vet ./... && go test ./...
```
Expected: all `ok`. If the vite package's `New` rejects a `StaticURL` without a leading slash or panics on the context helper name, check `/Users/jackieli/personal/gsxhq/vite/context.go` for the exact `NewContext` name and adjust; report any change.

- [ ] **Step 7: Smoke the export locally**

```bash
npm run build && go tool gsx generate && go run . -export /tmp/hxout -base /repo/ && ls /tmp/hxout /tmp/hxout/static/assets | head && grep -o 'src="/repo/frame/[^"]*"' /tmp/hxout/index.html | head -2 && rm -rf /tmp/hxout
```
Expected: `index.html`, `.nojekyll`, `frame/`, `static/`; asset files; iframe srcs prefixed with `/repo/`.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "Serve and export the site from one page map with a base path

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 3: Base-aware Vite build, static file server, Playwright under both configs

**Files:**
- Modify: `vite.config.ts`, `package.json`, `.gitignore`, `e2e/index.spec.ts`, `e2e/smoke.spec.ts`, `e2e/parity.spec.ts`, `e2e/playwright.config.ts`
- Create: `cmd/servestatic/main.go`, `e2e/static.config.ts`

**Interfaces:**
- Consumes: `-export`/`-base` flags from Task 2.
- Produces: `npm run export` (honours `SITE_BASE`), `npm run e2e:static`, `go run ./cmd/servestatic -dir out -base /x/ -port 8898`.

- [ ] **Step 1: Make Vite's base follow `SITE_BASE`**

In `vite.config.ts`, inside `defineConfig(({ command, mode }) => {` after the `env` line, add:

```ts
  // The Go side serves the bundle at <base>static/ (vite.StaticURL). Vite must
  // write the same prefix into CSS url() references, so both read SITE_BASE.
  const siteBase = "/" + (process.env.SITE_BASE || "/").replace(/^\/+|\/+$/g, "");
  const assetBase = (siteBase === "/" ? "/" : siteBase + "/") + "static/";
```
and change the `base:` line to `base: command === "serve" ? "/__vite/" : assetBase,`.

- [ ] **Step 2: Scripts and ignores**

`package.json` scripts:
```json
  "scripts": {
    "dev": "go tool gsx dev",
    "build": "vite build && touch dist/.gitkeep",
    "export": "npm run build && go tool gsx generate && go run . -export out -base ${SITE_BASE:-/}",
    "e2e:serve": "npm run build && go tool gsx generate && GO_PORT=8899 go run .",
    "e2e": "playwright test -c e2e",
    "e2e:static:serve": "SITE_BASE=/hx-live-vs-alpine/ npm run export && go run ./cmd/servestatic -dir out -base /hx-live-vs-alpine/ -port 8898",
    "e2e:static": "playwright test -c e2e/static.config.ts"
  },
```
Append `/out/` to `.gitignore`.

- [ ] **Step 3: The static file server**

`cmd/servestatic/main.go`:
```go
// Command servestatic serves an exported site directory under a base path,
// the way GitHub Pages serves a project site. Used by the static e2e config.
package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
)

func main() {
	dir := flag.String("dir", "out", "exported site directory")
	base := flag.String("base", "/", "path the site is mounted under")
	port := flag.String("port", "8898", "port to listen on")
	flag.Parse()

	b := "/" + strings.Trim(*base, "/")
	if b != "/" {
		b += "/"
	}
	mux := http.NewServeMux()
	mux.Handle(b, http.StripPrefix(strings.TrimSuffix(b, "/"), http.FileServer(http.Dir(*dir))))
	log.Printf("serving %s at http://localhost:%s%s", *dir, *port, b)
	log.Fatal(http.ListenAndServe(":"+*port, mux))
}
```

- [ ] **Step 4: Base-relative navigation in the specs**

- `e2e/parity.spec.ts`: every `page.goto(\`/frame/${lib}/<slug>\`)` becomes `page.goto(\`./frame/${lib}/<slug>/\`)` (relative, trailing slash). 22 occurrences; do it with one search-and-replace of `` goto(`/frame/${lib}/ `` → `` goto(`./frame/${lib}/ `` and then append `/` before the closing backtick on each of those lines.
- `e2e/smoke.spec.ts`: `page.goto("/")` → `page.goto("./")`; the two `startsWith("/frame/…")` filters become `includes("/frame/hxlive/")` and `includes("/frame/alpine/")`.
- `e2e/index.spec.ts`: `page.goto("/")` → `page.goto("./")`; the hx selector becomes `'iframe[data-frame][src*="/frame/hxlive/"]'`. Then append a font check inside the test, after the frame loop:
```ts
  // The main stylesheet references fonts by absolute URL; they must resolve
  // under the current base (this is what a wrong Vite `base` breaks).
  const css = await page.locator('link[rel="stylesheet"]').first().getAttribute("href");
  expect(css).toBeTruthy();
  const cssBody = await (await page.request.get(new URL(css!, page.url()).toString())).text();
  const font = cssBody.match(/url\(([^)]+\.woff2)\)/);
  expect(font, "stylesheet references a font").toBeTruthy();
  const fontRes = await page.request.get(new URL(font![1], page.url()).toString());
  expect(fontRes.status(), font![1]).toBe(200);
```

- [ ] **Step 5: The static config**

`e2e/static.config.ts`:
```ts
import { defineConfig } from "@playwright/test";

// Same specs, served from the exported directory under the project-site base,
// the way GitHub Pages will serve them.
export default defineConfig({
  testDir: ".",
  use: { baseURL: "http://localhost:8898/hx-live-vs-alpine/" },
  webServer: {
    command: "npm run e2e:static:serve",
    url: "http://localhost:8898/hx-live-vs-alpine/",
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
});
```
Leave `e2e/playwright.config.ts` unchanged.

- [ ] **Step 6: Run both suites**

```bash
npm run e2e && npm run e2e:static
lsof -i :8898 -i :8899
```
Expected: 48 passed twice; nothing listening afterwards. The font check runs under both configs and must pass under both (the Go server now serves fonts because Vite's base matches `StaticURL`). If Playwright resolves `"./"` against a `baseURL` without a trailing slash in an unexpected way, set `baseURL: "http://localhost:8899/"` in `e2e/playwright.config.ts`.

- [ ] **Step 7: Go tests still green, commit**

```bash
go tool gsx generate && go vet ./... && go test ./...
git add -A
git commit -m "Build assets under the site base and test the export like GitHub Pages serves it

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 4: GitHub Pages workflow and README

**Files:**
- Create: `.github/workflows/pages.yml`
- Modify: `README.md`

- [ ] **Step 1: The workflow**

`.github/workflows/pages.yml`:
```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

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
          SITE_BASE: /hx-live-vs-alpine/
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: out

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

- [ ] **Step 2: README**

Add a section after "Test":

```md
## Deploy

The site is static. `npm run export` renders every page to `out/` and copies
the bundle to `out/static/`; `SITE_BASE=/hx-live-vs-alpine/ npm run export`
prefixes every URL for a GitHub Pages project site. `.github/workflows/pages.yml`
does that on each push to `main` and publishes `out/` with `actions/deploy-pages`
(repo setting: Pages → Source → GitHub Actions).

`npm run e2e:static` runs the same Playwright suite against the exported
directory served under the base path, the way Pages serves it.
```

In the Test section add `npm run e2e:static  # same suite against the export` after the `npm run e2e` line. In Layout add `- `site/`: the page map that both serves and exports (candidate gsx pattern).`

- [ ] **Step 3: Validate the workflow file and commit**

```bash
npx --yes @action-validator/cli .github/workflows/pages.yml 2>/dev/null || python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pages.yml')); print('yaml ok')"
git add -A
git commit -m "Deploy the exported site to GitHub Pages on push to main

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 5: The gsx pattern draft

**Files:**
- Create: `docs/pattern/static-export.md`

- [ ] **Step 1: Write the pattern**

Model it on `/Users/jackieli/personal/gsxhq/gsx/docs/guide/patterns/render-once.md` (read it first for tone and section shape). Content, in this order:

1. Title `# Static export` and a two-sentence framing: a gsx site whose pages are pure functions of build-time data can be served by `net/http` in development and written to disk for static hosting, from one list of pages.
2. `## Copy the helper` with the full contents of `site/site.go` (verbatim, package name `site`).
3. `## Declare the site` showing `buildMap` and `pages.FramePath` from this repo, then `newHandler` and `export` from `main.go`, trimmed to the essentials.
4. `## Link with the base path` showing the `site.URL(ctx, ...)` call in a `.gsx` file and the `site.Middleware` wrapping.
5. `## Why the base path must reach Vite` with the font URL story: Vite writes `url(<base>assets/font.woff2)` into CSS, the Go side serves the bundle at `StaticURL`; both must be `<siteBase>static/`, so `vite.config.ts` derives `base` from `SITE_BASE` (show the four lines) and the Go side derives `StaticURL` the same way. Note that the gsx init template's `base: "/"` with `StaticURL "/static/"` breaks font URLs in prod today.
6. `## Publish` with the `export` npm script and the Pages workflow (verbatim).
7. `## Test it like it will be served` with `cmd/servestatic` and the two Playwright configs, three sentences.
8. `## Where this goes next` — one short paragraph: the `site` package is app-agnostic; content collections and a `gsx build` command over a `site.Map` would make it a static-site generator.

No file paths from this machine; repo-relative paths only.

- [ ] **Step 2: Commit**

```bash
git add docs/pattern/static-export.md
git commit -m "Draft the static-export pattern for contribution to gsx

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```
