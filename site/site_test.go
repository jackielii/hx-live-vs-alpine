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

func TestHandlerRedirectsSlashlessPagePath(t *testing.T) {
	h, err := Map{page("/a/b/", "<p>ab</p>")}.Handler()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Get(srv.URL + "/a/b")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode < 300 || res.StatusCode > 308 {
		t.Fatalf("status %d, want a redirect", res.StatusCode)
	}
	if loc := res.Header.Get("Location"); loc != "/a/b/" {
		t.Errorf("Location %q, want /a/b/", loc)
	}
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
