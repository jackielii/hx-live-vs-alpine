package main

import (
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
)

// testManifest mimics dist/.vite/manifest.json after `vite build` so the
// handler runs in prod mode without Vite.
const testManifest = `{
  "web/main.js":         {"file": "assets/main.js", "isEntry": true, "css": ["assets/main.css"]},
  "web/frame-alpine.js": {"file": "assets/frame-alpine.js", "isEntry": true, "css": ["assets/frame.css"]},
  "web/frame-hxlive.js": {"file": "assets/frame-hxlive.js", "isEntry": true, "css": ["assets/frame.css"]}
}`

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	fsys := fstest.MapFS{"dist/.vite/manifest.json": {Data: []byte(testManifest)}}
	v, err := vite.New(vite.Config{Dist: fsys, DistDir: "dist"})
	if err != nil {
		t.Fatal(err)
	}
	exs, err := examples.Load()
	if err != nil {
		t.Fatal(err)
	}
	feats, err := examples.Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(v.Middleware(newHandler(v, exs, feats)))
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

func TestFrameServesOneLibraryAndTheRawFragment(t *testing.T) {
	srv := newTestServer(t)
	exs, _ := examples.Load()
	counter, _ := examples.Find(exs, "counter")

	status, body := get(t, srv, "/frame/hxlive/counter")
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

	status, body = get(t, srv, "/frame/alpine/counter")
	if status != 200 || !strings.Contains(body, counter.Alpine) || !strings.Contains(body, "/static/assets/frame-alpine.js") {
		t.Errorf("alpine frame wrong: status %d", status)
	}
	if strings.Contains(body, "frame-hxlive") {
		t.Error("alpine frame loads hxlive")
	}
}

func TestFrameUnknownLibOrSlugIs404(t *testing.T) {
	srv := newTestServer(t)
	for _, p := range []string{"/frame/react/counter", "/frame/alpine/nope"} {
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
			src := "/frame/" + string(lib) + "/" + ex.Slug
			if !strings.Contains(body, `src="`+src+`"`) {
				t.Errorf("%s: no iframe for %s", ex.Slug, lib)
			}
			escaped := html.EscapeString(ex.Fragment(lib))
			if !strings.Contains(body, escaped) {
				t.Errorf("%s: escaped %s fragment not shown", ex.Slug, lib)
			}
		}
		if ex.HxLive != "" && strings.Contains(body, ex.HxLive) {
			t.Errorf("%s: raw hxlive fragment leaked unescaped into the index", ex.Slug)
		}
		if !ex.HasDemo() && strings.Contains(body, `src="/frame/hxlive/`+ex.Slug+`"`) {
			t.Errorf("%s: none row has an hxlive iframe", ex.Slug)
		}
	}
}

func TestIndexRendersTheMatrix(t *testing.T) {
	srv := newTestServer(t)
	exs, _ := examples.Load()
	feats, _ := examples.Features(exs)
	_, body := get(t, srv, "/")
	start := strings.Index(body, `id="matrix"`)
	if start < 0 {
		t.Fatal("no matrix table")
	}
	matrix := body[start:]
	if end := strings.Index(matrix, "</table>"); end > 0 {
		matrix = matrix[:end]
	}
	// sections[0] is the thead; sections 1..3 are the group tbody sections, in
	// the order examples.Groups lists them (Directives, Magics, Globals).
	sections := strings.Split(matrix, "<tbody>")
	if len(sections) != len(examples.Groups)+1 {
		t.Fatalf("matrix has %d <tbody> sections, want %d", len(sections)-1, len(examples.Groups))
	}
	sectionFor := func(g examples.Group) string {
		for i, group := range examples.Groups {
			if group == g {
				return sections[i+1]
			}
		}
		return ""
	}
	for _, f := range feats {
		section := sectionFor(f.Group)
		if !strings.Contains(section, html.EscapeString(f.Feature)) {
			t.Errorf("%s: feature %q not found in its %q section", f.Slug, f.Feature, f.Group)
		}
		if !strings.Contains(section, `href="#`+f.Slug+`"`) {
			t.Errorf("%s: link to %s not found in its %q section", f.Feature, f.Slug, f.Group)
		}
		if !strings.Contains(section, string(f.Status)) {
			t.Errorf("%s: status %q not found in its %q section", f.Feature, f.Status, f.Group)
		}
		for i, g := range examples.Groups {
			if g == f.Group {
				continue
			}
			if strings.Contains(sections[i+1], html.EscapeString(f.Feature)) {
				t.Errorf("%s: feature belongs to %q but also appears in the %q section", f.Feature, f.Group, g)
			}
		}
	}
	for _, g := range examples.Groups {
		if !strings.Contains(matrix, g.Label()) {
			t.Errorf("matrix lacks group heading %q", g.Label())
		}
	}
}

func TestNoneRowsHaveNoHxliveFrame(t *testing.T) {
	srv := newTestServer(t)
	exs, _ := examples.Load()
	for _, ex := range exs {
		status, _ := get(t, srv, "/frame/hxlive/"+ex.Slug)
		if ex.HasDemo() && status != 200 {
			t.Errorf("%s: hxlive frame status %d", ex.Slug, status)
		}
		if !ex.HasDemo() && status != 404 {
			t.Errorf("%s: none row hxlive frame status %d, want 404", ex.Slug, status)
		}
		if status, _ := get(t, srv, "/frame/alpine/"+ex.Slug); status != 200 {
			t.Errorf("%s: alpine frame status %d", ex.Slug, status)
		}
	}
}
