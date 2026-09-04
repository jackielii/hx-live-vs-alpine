package main

import (
	"cmp"
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/pages"
)

//go:embed all:dist
var distFS embed.FS

//go:embed all:public
var publicFS embed.FS

// newHandler builds the mux. Pure over its inputs so tests can drive it.
func newHandler(v *vite.Vite, exs []examples.Example, feats []examples.FeatureRow) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/public/", http.FileServerFS(publicFS))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if !v.Dev() {
		mux.Handle("/static/", v.StaticHandler())
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, pages.Index(exs, feats))
	})
	mux.HandleFunc("GET /frame/{lib}/{slug}", func(w http.ResponseWriter, r *http.Request) {
		lib, ok := examples.ParseLib(r.PathValue("lib"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		ex, ok := examples.Find(exs, r.PathValue("slug"))
		if !ok || (lib == examples.HxLive && !ex.HasDemo()) {
			http.NotFound(w, r)
			return
		}
		render(w, r, pages.Frame(lib, ex))
	})
	return mux
}

func render(w http.ResponseWriter, r *http.Request, n gsx.Node) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := n.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	devURL := os.Getenv("VITE_DEV_URL") // "" in prod
	v, err := vite.New(vite.Config{DevURL: devURL, DevBase: "/__vite/", Dist: distFS, DistDir: "dist"})
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

	port := cmp.Or(os.Getenv("GO_PORT"), "7777")
	srv := &http.Server{Addr: ":" + port, Handler: v.Middleware(newHandler(v, exs, feats))}

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
