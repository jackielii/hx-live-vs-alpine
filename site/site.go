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
