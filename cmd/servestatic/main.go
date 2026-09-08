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
