// Package examples holds the side-by-side demos: an Alpine.js fragment, its
// hx-live port where one exists, and notes, all read verbatim from disk so the
// source shown on the page is the source that runs in the iframe.
package examples

import (
	"bytes"
	"embed"
	"errors"
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

// Group is the Alpine docs section a feature belongs to.
type Group string

const (
	Directive Group = "directive"
	Magic     Group = "magic"
	Global    Group = "global"
)

// Groups lists groups in display order.
var Groups = []Group{Directive, Magic, Global}

// Label is the heading shown for the group.
func (g Group) Label() string {
	switch g {
	case Directive:
		return "Directives"
	case Magic:
		return "Magics"
	case Global:
		return "Globals"
	}
	return ""
}

// Status says how faithfully hx-live can reproduce the Alpine feature.
type Status string

const (
	Equivalent Status = "equivalent"
	Workaround Status = "workaround"
	None       Status = "none"
)

// Example is one card: the Alpine fragment, its hx-live port when Status is
// not None, and notes.
type Example struct {
	Slug      string
	Title     string
	Group     Group
	Features  []string // Alpine feature names this card demonstrates
	Status    Status
	Alpine    string // raw fragment
	HxLive    string // raw fragment; empty when Status == None
	NotesHTML string // notes.md rendered to HTML
}

// HasDemo reports whether the card has an hx-live frame.
func (e Example) HasDemo() bool { return e.Status != None }

// Fragment returns the raw fragment for lib.
func (e Example) Fragment(l Lib) string {
	if l == Alpine {
		return e.Alpine
	}
	return e.HxLive
}

// FeatureRow is one line of the matrix.
type FeatureRow struct {
	Feature string
	Slug    string
	Title   string
	Group   Group
	Status  Status
}

type row struct {
	slug, title string
	group       Group
	features    []string
	status      Status
}

// order fixes card order, titles and metadata; fragments come from disk.
var order = []row{
	{"counter", "Counter", Directive, []string{"x-data", "x-text", "$data"}, Equivalent},
	{"dropdown", "Dropdown with click-outside", Directive, []string{"x-show", "x-on (.outside)"}, Equivalent},
	{"classbind", "Class binding", Directive, []string{"x-bind (class object)"}, Equivalent},
	{"search", "Search filtering a list", Directive, []string{"x-model (filtering)", "x-for (filtering)"}, Workaround},
	{"tabs", "Tabs", Directive, []string{"x-bind (class, tabs)"}, Equivalent},
	{"transition", "Transition", Directive, []string{"x-transition"}, Workaround},
}

// featureOrder is the matrix order: Alpine docs order within each group.
var featureOrder = []string{
	"x-bind (class object)", "x-bind (class, tabs)", "x-data", "x-for (filtering)",
	"x-model (filtering)", "x-on (.outside)", "x-show", "x-text", "x-transition",
	"$data",
}

// Load reads every card from the embedded files. A missing or empty file, a
// stray hxlive.html on a None row, or a feature claimed by two cards is an
// error so the server refuses to start rather than serve a wrong page.
func Load() ([]Example, error) {
	md := goldmark.New()
	exs := make([]Example, 0, len(order))
	claimed := map[string]string{}
	for _, o := range order {
		alpine, err := read(o.slug, "alpine.html")
		if err != nil {
			return nil, err
		}
		var hxlive string
		if o.status == None {
			if _, err := fs.Stat(files, o.slug+"/hxlive.html"); err == nil {
				return nil, fmt.Errorf("examples: %s has status none but an hxlive.html", o.slug)
			}
		} else {
			hxlive, err = read(o.slug, "hxlive.html")
			if err != nil {
				return nil, err
			}
		}
		notes, err := read(o.slug, "notes.md")
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := md.Convert([]byte(notes), &buf); err != nil {
			return nil, fmt.Errorf("examples: %s/notes.md: %w", o.slug, err)
		}
		for _, f := range o.features {
			if prev, dup := claimed[f]; dup {
				return nil, fmt.Errorf("examples: feature %q claimed by %s and %s", f, prev, o.slug)
			}
			claimed[f] = o.slug
		}
		exs = append(exs, Example{
			Slug:      o.slug,
			Title:     o.title,
			Group:     o.group,
			Features:  o.features,
			Status:    o.status,
			Alpine:    alpine,
			HxLive:    hxlive,
			NotesHTML: buf.String(),
		})
	}
	return exs, nil
}

// Features returns one matrix line per feature in featureOrder. It errors when
// featureOrder and the cards disagree, so the matrix can never silently omit
// or invent a feature.
func Features(exs []Example) ([]FeatureRow, error) {
	byFeature := map[string]Example{}
	for _, ex := range exs {
		for _, f := range ex.Features {
			byFeature[f] = ex
		}
	}
	rows := make([]FeatureRow, 0, len(featureOrder))
	for _, f := range featureOrder {
		ex, ok := byFeature[f]
		if !ok {
			return nil, fmt.Errorf("examples: featureOrder lists %q but no card claims it", f)
		}
		rows = append(rows, FeatureRow{Feature: f, Slug: ex.Slug, Title: ex.Title, Group: ex.Group, Status: ex.Status})
		delete(byFeature, f)
	}
	if len(byFeature) > 0 {
		var missing []error
		for f, ex := range byFeature {
			missing = append(missing, fmt.Errorf("examples: card %s claims %q which is not in featureOrder", ex.Slug, f))
		}
		return nil, errors.Join(missing...)
	}
	return rows, nil
}

// ByGroup returns the cards in g, in card order.
func ByGroup(exs []Example, g Group) []Example {
	var out []Example
	for _, ex := range exs {
		if ex.Group == g {
			out = append(out, ex)
		}
	}
	return out
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
