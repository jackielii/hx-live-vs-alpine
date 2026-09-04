// Package examples holds the six side-by-side demos: an Alpine.js fragment, its
// hx-live port, and notes, all read verbatim from disk so the source shown on
// the page is the source that runs in the iframe.
package examples

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"

	"github.com/yuin/goldmark"
)

//go:embed */alpine.html */hxlive.html */notes.md
var files embed.FS

// Lib identifies which library a fragment targets.
type Lib string

const (
	Alpine Lib = "alpine"
	HxLive Lib = "hxlive"
)

// ParseLib maps a URL segment to a Lib.
func ParseLib(s string) (Lib, bool) {
	switch Lib(s) {
	case Alpine, HxLive:
		return Lib(s), true
	}
	return "", false
}

// Label is the human name shown on badges.
func (l Lib) Label() string {
	if l == Alpine {
		return "Alpine.js"
	}
	return "htmx hx-live"
}

// Entry is the Vite entry that loads exactly this library.
func (l Lib) Entry() string {
	return "web/frame-" + string(l) + ".js"
}

// Example is one demo pair.
type Example struct {
	Slug      string
	Title     string
	Alpine    string // raw fragment
	HxLive    string // raw fragment
	NotesHTML string // notes.md rendered to HTML
}

// Fragment returns the raw fragment for lib.
func (e Example) Fragment(l Lib) string {
	if l == Alpine {
		return e.Alpine
	}
	return e.HxLive
}

// order fixes display order and titles; everything else comes from disk.
var order = []struct{ slug, title string }{
	{"counter", "Counter"},
	{"dropdown", "Dropdown with click-outside"},
	{"search", "Search filtering a list"},
	{"tabs", "Tabs"},
	{"transition", "Transition"},
	{"classbind", "Class binding"},
}

// Load reads every example from the embedded files. A missing or empty file is
// an error so the server refuses to start rather than serving a blank demo.
func Load() ([]Example, error) {
	md := goldmark.New()
	exs := make([]Example, 0, len(order))
	for _, o := range order {
		alpine, err := read(o.slug, "alpine.html")
		if err != nil {
			return nil, err
		}
		hxlive, err := read(o.slug, "hxlive.html")
		if err != nil {
			return nil, err
		}
		notes, err := read(o.slug, "notes.md")
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := md.Convert([]byte(notes), &buf); err != nil {
			return nil, fmt.Errorf("examples: %s/notes.md: %w", o.slug, err)
		}
		exs = append(exs, Example{
			Slug:      o.slug,
			Title:     o.title,
			Alpine:    alpine,
			HxLive:    hxlive,
			NotesHTML: buf.String(),
		})
	}
	return exs, nil
}

// Find returns the example with slug.
func Find(exs []Example, slug string) (Example, bool) {
	for _, ex := range exs {
		if ex.Slug == slug {
			return ex, true
		}
	}
	return Example{}, false
}

func read(slug, name string) (string, error) {
	b, err := fs.ReadFile(files, slug+"/"+name)
	if err != nil {
		return "", fmt.Errorf("examples: %w", err)
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return "", fmt.Errorf("examples: %s/%s is empty", slug, name)
	}
	return string(b), nil
}
