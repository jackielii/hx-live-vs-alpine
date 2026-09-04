package main

import (
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
	srv := httptest.NewServer(v.Middleware(newHandler(v, exs)))
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
