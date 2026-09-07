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
