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
	"github.com/jackielii/hx-live-vs-alpine/site"
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

func loadMap(t *testing.T) ([]examples.Example, site.Pages) {
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
		if !strings.Contains(body, `href="https://github.com/jackielii/hx-live-vs-alpine/tree/main/examples/`+ex.Slug+`"`) {
			t.Errorf("%s: no source link", ex.Slug)
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

func TestServesUnderANonRootBase(t *testing.T) {
	v := testVite(t, "/repo/")
	_, m := loadMap(t)
	h, err := newHandler(v, m, "/repo/")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(v.Middleware(h))
	t.Cleanup(srv.Close)

	status, body := get(t, srv, "/repo/")
	if status != 200 {
		t.Fatalf("GET /repo/: status %d", status)
	}
	if !strings.Contains(body, `src="/repo/frame/hxlive/counter/"`) {
		t.Error("/repo/ page lacks the base-prefixed iframe src")
	}
	if !strings.Contains(body, "/repo/static/assets/main.js") {
		t.Error("/repo/ page lacks the base-prefixed asset URL")
	}

	if status, _ := get(t, srv, "/repo/frame/alpine/counter/"); status != 200 {
		t.Errorf("GET /repo/frame/alpine/counter/: status %d", status)
	}

	for _, p := range []string{"/", "/frame/alpine/counter/"} {
		if status, _ := get(t, srv, p); status != 404 {
			t.Errorf("GET %s: status %d, want 404", p, status)
		}
	}

	// The stub manifest has no backing file body for the asset, so we can't
	// assert 200 here; instead confirm the request reached StaticHandler (a
	// file server) rather than the page mux, by checking it didn't answer
	// with an HTML page.
	res, err := http.Get(srv.URL + "/repo/static/assets/main.js")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		t.Errorf("/repo/static/assets/main.js: content type %q looks like the page mux answered", ct)
	}
}

func TestExportWritesTheSiteUnderABase(t *testing.T) {
	v := testVite(t, "/repo/")
	exs, m := loadMap(t)
	dir := t.TempDir()
	bundle := fstest.MapFS{
		"assets/main.js":      {Data: []byte("//js")},
		".vite/manifest.json": {Data: []byte("{}")},
		".gitkeep":            {},
	}
	if err := export(context.Background(), v, m, "/repo/", dir, bundle); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"index.html", ".nojekyll", "frame/alpine/counter/index.html", "frame/hxlive/counter/index.html", "frame/alpine/teleport/index.html", "static/assets/main.js"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	for _, rel := range []string{"static/.vite", "static/.gitkeep"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil {
			t.Errorf("%s should have been skipped from the bundle copy", rel)
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
