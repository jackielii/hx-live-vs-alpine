# Feature Catalogue Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the six-example comparison app into a catalogue covering every Alpine.js core feature (18 directives, 9 magics, 3 globals) with a status matrix, like-for-like hx-live ports where they exist, and honest "no equivalent" cards where they don't.

**Architecture:** The `examples` manifest gains group, features and status per row and a `Features()` view for the matrix; the loader enforces that files on disk match the declared status. `pages/index.gsx` renders the matrix, a grouped sidebar, status badges, and a prose right column for `none` rows. The frame route refuses hx-live frames for `none` rows. Playwright gains a generic smoke over every frame plus parity tests for each new interactive row.

**Tech Stack:** unchanged: Go 1.26, gsx, gsxui (card, badge, button), gsxhq/vite, Vite 6 + Tailwind 4, `htmx.org@4.0.0`, `alpinejs@3.17.1`, goldmark, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-04-feature-catalogue-design.md` (extends `docs/superpowers/specs/2026-09-04-hx-live-vs-alpine-design.md`)

## Global Constraints

- Alpine fragments are the docs' first example for the feature, verbatim (`/Users/jackieli/personal/alpine/packages/docs/src/en/{directives,magics,globals}/*.md`), with one systematic adaptation: where the docs use `alert()` or `console.log()`, the side effect writes into a visible element and the notes say "adapted from the docs".
- hx-live ports keep the Alpine DOM shape and use htmx 4.0.0 idioms: typed bags `data.*`, `aria.*`, `class.*`, `attr.*`; bindings `:attr`, `:text`, `:html`, `:hidden`, `:.class`, `:class`; helpers `q()`, `trigger()`, `insert()`, `nextFrame()`; events via `hx-on:<event>` or `hx-on="<event>[filter] -> js"`. Authoritative docs: `/private/tmp/claude-501/-Users-jackieli-personal-hx-live-vs-alpine/ff925b44-d94f-4bf4-a102-1e159ec77ef1/scratchpad/hx-live-4.0.0.md` (the `../htmx` checkout docs are a stale beta).
- Any CSS class a fragment relies on is styled by a `<style>` inside that fragment. The shown source and the injected source are the same bytes.
- Alpine and hx-live never share a document; the outer bundle imports neither.
- Statuses: `equivalent` | `workaround` | `none`. `none` rows have `alpine.html` and `notes.md` only. Every feature name appears on exactly one card.
- Notes are blog copy: accurate, concise, no debugging history, no file paths.
- Parity assertions are never weakened to make an hx-live port pass; a failing hx-live port is fixed and the reason recorded in that row's `notes.md`. Alpine fragments are never edited to pass a test.
- Unexported identifiers unless something must cross a package boundary. Fragment files end with one trailing newline, 4-space indentation as given.
- Commit messages end with `Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb`.
- Every task ends with `go tool gsx generate && go vet ./... && go test ./...` green; tasks that touch fragments or pages also run `npm run build`. Never leave a server on 8899 (`lsof -i :8899`).

---

## File structure

| Path | Responsibility |
|---|---|
| `examples/examples.go` | manifest types (`Group`, `Status`, `Example`, `FeatureRow`), `order`, `featureOrder`, `Load`, `Features`, `ByGroup`, `Groups` |
| `examples/examples_test.go` | invariants: order, status/file agreement, feature uniqueness, matrix coverage |
| `examples/<slug>/` | 21 new directories with `alpine.html`, `notes.md`, and `hxlive.html` for demo rows |
| `main.go` | frame route 404 for `none` + hxlive; `newHandler(v, exs, feats)` |
| `main_test.go` | matrix, none-row routing |
| `pages/index.gsx` | matrix, grouped sidebar, status badges, `none` layout |
| `e2e/smoke.spec.ts` | every frame loads under both libraries, no console errors |
| `e2e/index.spec.ts` | frame count derived from the page, not a constant |
| `e2e/parity.spec.ts` | parity tests for the new interactive rows |
| `README.md` | one paragraph on the catalogue and statuses |

---

### Task 1: Manifest schema with group, features and status

**Files:**
- Modify: `examples/examples.go`
- Modify: `examples/examples_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces:
  ```go
  type Group string  // Directive, Magic, Global
  func (g Group) Label() string  // "Directives" | "Magics" | "Globals"
  var Groups = []Group{Directive, Magic, Global}
  type Status string // Equivalent, Workaround, None
  type Example struct { Slug, Title string; Group Group; Features []string; Status Status; Alpine, HxLive, NotesHTML string }
  func (e Example) HasDemo() bool
  type FeatureRow struct { Feature, Slug, Title string; Group Group; Status Status }
  func Features(exs []Example) ([]FeatureRow, error)  // matrix lines in featureOrder
  func ByGroup(exs []Example, g Group) []Example
  ```
  `Load`, `Find`, `Lib`, `ParseLib`, `Fragment` unchanged in signature.

- [ ] **Step 1: Replace the tests**

Replace `examples/examples_test.go` entirely:

```go
package examples

import (
	"strings"
	"testing"
)

func mustLoad(t *testing.T) []Example {
	t.Helper()
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return exs
}

func TestLoadKeepsTheOriginalSixFirst(t *testing.T) {
	exs := mustLoad(t)
	want := []string{"counter", "dropdown", "classbind"}
	for i, slug := range want {
		if exs[i].Slug != slug {
			t.Errorf("index %d: slug %q, want %q", i, exs[i].Slug, slug)
		}
	}
	for _, slug := range []string{"search", "tabs", "transition"} {
		if _, ok := Find(exs, slug); !ok {
			t.Errorf("%s missing", slug)
		}
	}
}

func TestLoadInvariants(t *testing.T) {
	exs := mustLoad(t)
	seen := map[string]string{}
	lastGroup := -1
	for _, ex := range exs {
		if ex.Title == "" || ex.Group == "" || ex.Status == "" || len(ex.Features) == 0 {
			t.Errorf("%s: incomplete metadata %+v", ex.Slug, ex)
		}
		if strings.TrimSpace(ex.Alpine) == "" {
			t.Errorf("%s: empty alpine fragment", ex.Slug)
		}
		if strings.Contains(ex.HxLive, "x-data") {
			t.Errorf("%s: hxlive fragment contains x-data", ex.Slug)
		}
		if ex.HasDemo() != (strings.TrimSpace(ex.HxLive) != "") {
			t.Errorf("%s: status %q but hxlive fragment present=%v", ex.Slug, ex.Status, ex.HxLive != "")
		}
		if !strings.Contains(ex.NotesHTML, "<p>") {
			t.Errorf("%s: notes not rendered", ex.Slug)
		}
		for _, f := range ex.Features {
			if prev, dup := seen[f]; dup {
				t.Errorf("feature %q on both %s and %s", f, prev, ex.Slug)
			}
			seen[f] = ex.Slug
		}
		gi := groupIndex(ex.Group)
		if gi < lastGroup {
			t.Errorf("%s: group %q out of order", ex.Slug, ex.Group)
		}
		lastGroup = gi
	}
}

func groupIndex(g Group) int {
	for i, x := range Groups {
		if x == g {
			return i
		}
	}
	return -1
}

func TestFeaturesCoversEveryCardFeatureOnce(t *testing.T) {
	exs := mustLoad(t)
	feats, err := Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	fromCards := map[string]bool{}
	for _, ex := range exs {
		for _, f := range ex.Features {
			fromCards[f] = true
		}
	}
	fromMatrix := map[string]int{}
	for _, f := range feats {
		fromMatrix[f.Feature]++
		ex, ok := Find(exs, f.Slug)
		if !ok || ex.Title != f.Title || ex.Group != f.Group || ex.Status != f.Status {
			t.Errorf("matrix line %q does not match card %q", f.Feature, f.Slug)
		}
	}
	for f := range fromCards {
		if fromMatrix[f] != 1 {
			t.Errorf("feature %q appears %d times in the matrix", f, fromMatrix[f])
		}
	}
	if len(fromMatrix) != len(fromCards) {
		t.Errorf("matrix has %d features, cards have %d", len(fromMatrix), len(fromCards))
	}
}

func TestByGroupAndLabels(t *testing.T) {
	exs := mustLoad(t)
	total := 0
	for _, g := range Groups {
		if g.Label() == "" {
			t.Errorf("group %q has no label", g)
		}
		total += len(ByGroup(exs, g))
	}
	if total != len(exs) {
		t.Errorf("ByGroup partitions %d of %d", total, len(exs))
	}
}

func TestFragmentAndFind(t *testing.T) {
	exs := mustLoad(t)
	ex, ok := Find(exs, "counter")
	if !ok {
		t.Fatal("counter not found")
	}
	if ex.Fragment(Alpine) != ex.Alpine || ex.Fragment(HxLive) != ex.HxLive {
		t.Error("Fragment does not select the right side")
	}
	if _, ok := Find(exs, "nope"); ok {
		t.Error("Find returned ok for unknown slug")
	}
}

func TestParseLib(t *testing.T) {
	for s, want := range map[string]Lib{"alpine": Alpine, "hxlive": HxLive} {
		got, ok := ParseLib(s)
		if !ok || got != want {
			t.Errorf("ParseLib(%q) = %q, %v", s, got, ok)
		}
	}
	if _, ok := ParseLib("react"); ok {
		t.Error("ParseLib accepted react")
	}
	if Alpine.Entry() != "web/frame-alpine.js" || HxLive.Entry() != "web/frame-hxlive.js" {
		t.Error("wrong entry paths")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./examples/
```
Expected: build failure, "undefined: Groups", "undefined: Features", etc.

- [ ] **Step 3: Rewrite `examples/examples.go`**

Replace the file with this. The six existing rows are re-keyed and reordered: counter, dropdown, classbind first (all directives), then search, tabs, transition further down (tabs is not a docs feature; it stays a scenario card under directives with feature `x-bind (class, tabs)`).

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go vet ./examples/ && go test ./examples/
```
Expected: `ok`. Then `go tool gsx generate && go build ./...` still succeeds (nothing else references the changed fields yet).

- [ ] **Step 5: Commit**

```bash
git add examples/examples.go examples/examples_test.go
git commit -m "Add group, features and status to the examples manifest

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 2: Directive cards with equivalent ports (bind, on, init, html, effect, ignore, ref)

**Files:**
- Create: `examples/{bind,on,init,html,effect,ignore,ref}/{alpine.html,hxlive.html,notes.md}`
- Modify: `examples/examples.go` (`order`, `featureOrder`)

**Interfaces:**
- Consumes: `row`, `order`, `featureOrder` from Task 1.
- Produces: seven new cards; features `x-bind`, `x-on (modifiers)`, `x-init`, `x-html`, `x-effect`, `x-ignore`, `x-ref`, `$refs`.

- [ ] **Step 1: Add the rows**

In `examples/examples.go`, `order` becomes (keep the existing six entries' content; only their position changes):

```go
var order = []row{
	{"counter", "Counter", Directive, []string{"x-data", "x-text", "$data"}, Equivalent},
	{"dropdown", "Dropdown with click-outside", Directive, []string{"x-show", "x-on (.outside)"}, Equivalent},
	{"classbind", "Class binding", Directive, []string{"x-bind (class object)"}, Equivalent},
	{"bind", "Attribute binding", Directive, []string{"x-bind"}, Equivalent},
	{"on", "Event modifiers", Directive, []string{"x-on (modifiers)"}, Equivalent},
	{"init", "Initialisation", Directive, []string{"x-init"}, Equivalent},
	{"html", "HTML binding", Directive, []string{"x-html"}, Equivalent},
	{"effect", "Effects", Directive, []string{"x-effect"}, Equivalent},
	{"ignore", "Ignoring a subtree", Directive, []string{"x-ignore"}, Equivalent},
	{"ref", "References", Directive, []string{"x-ref", "$refs"}, Equivalent},
	{"search", "Search filtering a list", Directive, []string{"x-model (filtering)", "x-for (filtering)"}, Workaround},
	{"tabs", "Tabs", Directive, []string{"x-bind (class, tabs)"}, Equivalent},
	{"transition", "Transition", Directive, []string{"x-transition"}, Workaround},
}
```

and `featureOrder` becomes:

```go
var featureOrder = []string{
	"x-bind", "x-bind (class object)", "x-bind (class, tabs)", "x-data", "x-effect",
	"x-for (filtering)", "x-html", "x-ignore", "x-init", "x-model (filtering)",
	"x-on (modifiers)", "x-on (.outside)", "x-ref", "x-show", "x-text", "x-transition",
	"$data", "$refs",
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./examples/
```
Expected: FAIL from `Load`: `examples: open bind/alpine.html: file does not exist`.

- [ ] **Step 3: Write the fragments and notes**

`examples/bind/alpine.html`:
```html
<div x-data="{ placeholderText: 'Type here...' }">
    <input type="text" x-bind:placeholder="placeholderText">
</div>
```
`examples/bind/hxlive.html`:
```html
<div data-placeholder="Type here...">
    <input type="text" :placeholder="data.placeholder">
</div>
```
`examples/bind/notes.md`:
```md
Any attribute binds with a `:` prefix in both. Alpine reads `placeholderText` from its component object; hx-live reads `data.placeholder` from the closest `data-placeholder` attribute.
```

`examples/on/alpine.html` (adapted from the docs' `@keyup.enter` example):
```html
<div x-data="{ message: '' }">
    <input type="text" @keyup.enter="message = 'Enter pressed'">
    <span x-text="message"></span>
</div>
```
`examples/on/hxlive.html`:
```html
<div data-message="">
    <input type="text" hx-on="keyup[key=='Enter'] -> data.message = 'Enter pressed'">
    <span :text="data.message"></span>
</div>
```
`examples/on/notes.md`:
```md
Adapted from the docs' `@keyup.enter` example. Alpine's dotted modifiers (`.enter`, `.prevent`, `.debounce`) become `hx-on`'s event grammar: a bracket filter such as `keyup[key=='Enter']`, modifiers such as `debounce:200ms`, and plain `event.preventDefault()` where Alpine would write `.prevent`.
```

`examples/init/alpine.html` (adapted: the docs log to the console):
```html
<div x-data="{ message: '' }" x-init="message = 'Initialised!'">
    <span x-text="message"></span>
</div>
```
`examples/init/hxlive.html`:
```html
<div data-message="" hx-on:load="data.message = 'Initialised!'">
    <span :text="data.message"></span>
</div>
```
`examples/init/notes.md`:
```md
Adapted from the docs, which log to the console. `x-init` runs once when Alpine initialises the element; htmx fires a `load` event when it processes an element, so `hx-on:load` is the same hook.
```

`examples/html/alpine.html`:
```html
<div x-data="{ username: '<strong>calebporzio</strong>' }">
    Username: <span x-html="username"></span>
</div>
```
`examples/html/hxlive.html`:
```html
<div data-username="<strong>calebporzio</strong>">
    Username: <span :html="data.username"></span>
</div>
```
`examples/html/notes.md`:
```md
`x-html` and `:html` both set `innerHTML`, with the same warning: never feed either untrusted markup. The hx-live value is a plain string in a `data-*` attribute; a value that is not valid JSON is returned as-is.
```

`examples/effect/alpine.html` (adapted: the docs log to the console):
```html
<div x-data="{ label: 'Hello', length: 0 }" x-effect="length = label.length">
    <button @click="label += ' World!'">Change Message</button>
    <span x-text="length"></span>
</div>
```
`examples/effect/hxlive.html`:
```html
<div data-label="Hello" data-length="0" hx-live="data.length = data.label.length">
    <button hx-on:click="data.label += ' World!'">Change Message</button>
    <span :text="data.length"></span>
</div>
```
`examples/effect/notes.md`:
```md
Adapted from the docs, which log to the console. `x-effect` tracks the reactive properties it reads and re-runs when one changes. An `hx-live` expression re-runs after any DOM mutation instead, so it needs no dependency tracking; the result is the same and the expression must simply be cheap and idempotent.
```

`examples/ignore/alpine.html` (docs example plus one visible sibling):
```html
<div x-data="{ label: 'processed' }">
    <span x-text="label"></span>
    <div x-ignore>
        <span x-text="label">untouched</span>
    </div>
</div>
```
`examples/ignore/hxlive.html`:
```html
<div data-label="processed">
    <span :text="data.label"></span>
    <div hx-ignore>
        <span :text="data.label">untouched</span>
    </div>
</div>
```
`examples/ignore/notes.md`:
```md
The docs example plus a sibling outside the ignored subtree so the difference is visible. `x-ignore` and htmx's `hx-ignore` both leave the subtree alone: the inner span keeps its original text while the outer one is bound.
```

`examples/ref/alpine.html` (docs example inside an `x-data` wrapper, which the docs page provides implicitly):
```html
<div x-data>
    <button @click="$refs.text.remove()">Remove Text</button>

    <span x-ref="text">Hello 👋</span>
</div>
```
`examples/ref/hxlive.html`:
```html
<div>
    <button hx-on:click="q('#text').remove()">Remove Text</button>

    <span id="text">Hello 👋</span>
</div>
```
`examples/ref/notes.md`:
```md
Alpine keeps a per-component registry of `x-ref` names behind `$refs`. hx-live has no registry; an id, or a directional query such as `q('next span')`, reaches the same element. Method calls pass through the `q()` proxy.
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go vet ./examples/ && go test ./examples/ && go tool gsx generate && go test ./...
```
Expected: all `ok` (the index test still passes because it checks the fragments it finds, not a count).

- [ ] **Step 5: Commit**

```bash
git add examples
git commit -m "Add directive cards: bind, on, init, html, effect, ignore, ref

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 3: Directive cards with workarounds and no equivalent (model, for, if, cloak, teleport, modelable, id)

**Files:**
- Create: `examples/{model,for,if,cloak}/{alpine.html,hxlive.html,notes.md}`
- Create: `examples/{teleport,modelable,id}/{alpine.html,notes.md}` (no hxlive.html)
- Modify: `examples/examples.go` (`order`, `featureOrder`)

**Interfaces:**
- Consumes: Task 1 types. Produces: seven cards; features `x-model`, `x-for`, `x-if`, `x-cloak`, `x-teleport`, `x-modelable`, `x-id`, `$id`.

- [ ] **Step 1: Add the rows**

Replace `order` with:

```go
var order = []row{
	{"counter", "Counter", Directive, []string{"x-data", "x-text", "$data"}, Equivalent},
	{"dropdown", "Dropdown with click-outside", Directive, []string{"x-show", "x-on (.outside)"}, Equivalent},
	{"classbind", "Class binding", Directive, []string{"x-bind (class object)"}, Equivalent},
	{"bind", "Attribute binding", Directive, []string{"x-bind"}, Equivalent},
	{"on", "Event modifiers", Directive, []string{"x-on (modifiers)"}, Equivalent},
	{"init", "Initialisation", Directive, []string{"x-init"}, Equivalent},
	{"html", "HTML binding", Directive, []string{"x-html"}, Equivalent},
	{"effect", "Effects", Directive, []string{"x-effect"}, Equivalent},
	{"ignore", "Ignoring a subtree", Directive, []string{"x-ignore"}, Equivalent},
	{"ref", "References", Directive, []string{"x-ref", "$refs"}, Equivalent},
	{"model", "Two-way binding", Directive, []string{"x-model"}, Workaround},
	{"for", "Loops", Directive, []string{"x-for"}, Workaround},
	{"search", "Search filtering a list", Directive, []string{"x-model (filtering)", "x-for (filtering)"}, Workaround},
	{"if", "Conditional rendering", Directive, []string{"x-if"}, Workaround},
	{"tabs", "Tabs", Directive, []string{"x-bind (class, tabs)"}, Equivalent},
	{"transition", "Transition", Directive, []string{"x-transition"}, Workaround},
	{"cloak", "Cloaking", Directive, []string{"x-cloak"}, Workaround},
	{"teleport", "Teleport", Directive, []string{"x-teleport"}, None},
	{"modelable", "Modelable", Directive, []string{"x-modelable"}, None},
	{"id", "Unique ids", Directive, []string{"x-id", "$id"}, None},
}
```

and `featureOrder` with:

```go
var featureOrder = []string{
	"x-bind", "x-bind (class object)", "x-bind (class, tabs)", "x-cloak", "x-data", "x-effect",
	"x-for", "x-for (filtering)", "x-html", "x-id", "x-if", "x-ignore", "x-init",
	"x-model", "x-model (filtering)", "x-modelable", "x-on (modifiers)", "x-on (.outside)",
	"x-ref", "x-show", "x-teleport", "x-text", "x-transition",
	"$data", "$id", "$refs",
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./examples/
```
Expected: FAIL, `model/alpine.html: file does not exist`.

- [ ] **Step 3: Write the fragments and notes**

`examples/model/alpine.html`:
```html
<div x-data="{ message: '' }">
    <input type="text" x-model="message">

    <span x-text="message"></span>
</div>
```
`examples/model/hxlive.html`:
```html
<div>
    <input type="text">

    <span :text="q('previous input').value"></span>
</div>
```
`examples/model/notes.md`:
```md
`x-model` keeps an input and a property in sync both ways. hx-live has no two-way binding because the input already is the state: read it with `q('previous input').value`, and when a handler must write it, assign to `.value`. hx-live re-evaluates on `input` events after a short debounce (`config.live.inputDebounce`, 100 ms by default), so the span lags a keystroke by that much.
```

`examples/for/alpine.html` (docs example plus a push, so the list changes):
```html
<div x-data="{ colors: ['Red', 'Orange', 'Yellow'] }">
    <ul>
        <template x-for="color in colors">
            <li x-text="color"></li>
        </template>
    </ul>
    <button @click="colors.push('Green')">Add Green</button>
</div>
```
`examples/for/hxlive.html`:
```html
<div>
    <ul>
        <li>Red</li>
        <li>Orange</li>
        <li>Yellow</li>
    </ul>
    <button hx-on:click="q('previous ul').insert('end', '<li>Green</li>')">Add Green</button>
</div>
```
`examples/for/notes.md`:
```md
The docs example with a button that appends an item. There is no loop primitive in hx-live: the list is HTML, not an array, so it is rendered by the server or written by hand. Adding to it is `insert()`, or an htmx request when the server owns the list. This is the largest model difference between the two libraries.
```

`examples/if/alpine.html` (the docs' `x-if` example inside the toggle it needs):
```html
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle</button>

    <template x-if="open">
        <div>Contents...</div>
    </template>
</div>
```
`examples/if/hxlive.html`:
```html
<div data-open="false">
    <button hx-on:click="data.open = !data.open">Toggle</button>

    <div :hidden="!data.open">Contents...</div>
</div>
```
`examples/if/notes.md`:
```md
`x-if` adds and removes the element from the DOM. hx-live never removes elements; `:hidden` keeps it in place and toggles visibility, exactly like `x-show`. When removal matters, for form submission or focus order, the honest hx-live answer is a server swap.
```

`examples/cloak/alpine.html` (docs example inside an `x-data` wrapper, with the CSS the docs prescribe):
```html
<div x-data>
    <span x-cloak x-show="false">This will not 'blip' onto screen at any point</span>
</div>
<style>
    [x-cloak] { display: none !important; }
</style>
```
`examples/cloak/hxlive.html`:
```html
<div>
    <span hx-cloak :hx-cloak="false" :hidden="true">This will not 'blip' onto screen at any point</span>
</div>
<style>
    [hx-cloak] { display: none !important; }
</style>
```
`examples/cloak/notes.md`:
```md
Alpine removes `x-cloak` when it initialises an element, so a `[x-cloak] { display: none }` rule hides markup until then. hx-live has no cloak directive, but a binding can remove any attribute: `:hx-cloak="false"` drops the marker in the same evaluation pass that applies `:hidden`, so the same CSS trick works.
```

`examples/teleport/alpine.html` (docs example without its `<body>` wrapper):
```html
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle Modal</button>

    <template x-teleport="body">
        <div x-show="open">
            Modal contents...
        </div>
    </template>
</div>

<div>Some other content placed AFTER the modal markup.</div>
```
`examples/teleport/notes.md`:
```md
`x-teleport` moves a subtree elsewhere in the document at runtime, usually to escape an ancestor's `overflow` or stacking context. hx-live has nothing like it. The modern answer is to not need it: `<dialog>` and the Popover API render in the top layer regardless of where the element sits, and anything else can be placed where it belongs by the server.
```

`examples/modelable/alpine.html`:
```html
<div x-data="{ number: 5 }">
    <div x-data="{ count: 0 }" x-modelable="count" x-model="number">
        <button @click="count++">Increment</button>
    </div>

    Number: <span x-text="number"></span>
</div>
```
`examples/modelable/notes.md`:
```md
`x-modelable` exposes a child component's property so a parent can `x-model` it. It only makes sense where components own private state. hx-live has no components and no private state: both elements would read and write the same `data-*` attribute on their common ancestor, and there would be nothing to expose.
```

`examples/id/alpine.html`:
```html
<div x-id="['text-input']">
    <label :for="$id('text-input')">Username</label>
    <!-- for="text-input-1" -->

    <input type="text" :id="$id('text-input')">
    <!-- id="text-input-1" -->
</div>

<div x-id="['text-input']">
    <label :for="$id('text-input')">Username</label>
    <!-- for="text-input-2" -->

    <input type="text" :id="$id('text-input')">
    <!-- id="text-input-2" -->
</div>
```
`examples/id/notes.md`:
```md
`x-id` and `$id` generate unique ids on the client so repeated components can pair labels with inputs. hx-live does not generate ids because the markup is already unique when it arrives: the server that rendered two copies of a component is the natural place to number them, and gsx does exactly that at render time.
```

- [ ] **Step 4: Relax the index test for rows without a port**

`TestIndexListsEveryExampleWithBothSources` in `main_test.go` expects an hx-live iframe for every row. Until Task 5 rewrites the page, `none` rows still get one, so only stop asserting the hx-live side for them. Replace the inner loop header:

```go
		libs := []examples.Lib{examples.Alpine}
		if ex.HasDemo() {
			libs = append(libs, examples.HxLive)
		}
		for _, lib := range libs {
```
(replacing `for _, lib := range []examples.Lib{examples.Alpine, examples.HxLive} {`), and guard the raw-leak check with `ex.HxLive != "" &&`.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go vet ./examples/ && go test ./examples/ && go tool gsx generate && go test ./...
```
Expected: all `ok`.

- [ ] **Step 6: Commit**

```bash
git add examples main_test.go
git commit -m "Add directive cards: model, for, if, cloak, teleport, modelable, id

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 4: Magic and global cards (el, dispatch, root, nexttick, store, watch, alpine-data, alpine-bind)

**Files:**
- Create: `examples/{el,dispatch,root,nexttick,store,watch}/{alpine.html,hxlive.html,notes.md}`
- Create: `examples/{alpine-data,alpine-bind}/{alpine.html,notes.md}`
- Modify: `examples/examples.go` (`order`, `featureOrder`), `examples/examples_test.go` (final matrix assertion)

**Interfaces:**
- Consumes: Task 1 types. Produces: the complete 27-card manifest and the final 34-line `featureOrder`.

- [ ] **Step 1: Add the final matrix assertion to the tests**

Append to `examples/examples_test.go`:

```go
func TestMatrixIsComplete(t *testing.T) {
	exs := mustLoad(t)
	feats, err := Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"x-bind", "x-bind (class object)", "x-bind (class, tabs)", "x-cloak", "x-data", "x-effect",
		"x-for", "x-for (filtering)", "x-html", "x-id", "x-if", "x-ignore", "x-init",
		"x-model", "x-model (filtering)", "x-modelable", "x-on (modifiers)", "x-on (.outside)",
		"x-ref", "x-show", "x-teleport", "x-text", "x-transition",
		"$data", "$dispatch", "$el", "$id", "$nextTick", "$refs", "$root", "$store", "$watch",
		"Alpine.bind", "Alpine.data", "Alpine.store",
	}
	if len(feats) != len(want) {
		t.Fatalf("matrix has %d lines, want %d", len(feats), len(want))
	}
	for i, f := range feats {
		if f.Feature != want[i] {
			t.Errorf("line %d: %q, want %q", i, f.Feature, want[i])
		}
	}
	if len(exs) != 28 {
		t.Errorf("got %d cards, want 28", len(exs))
	}
	none := 0
	for _, ex := range exs {
		if !ex.HasDemo() {
			none++
		}
	}
	if none != 5 {
		t.Errorf("got %d none rows, want 5", none)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./examples/ -run TestMatrixIsComplete
```
Expected: FAIL, "matrix has 26 lines, want 35".

- [ ] **Step 3: Add the rows**

Append to `order` after the `id` row:

```go
	{"el", "The current element", Magic, []string{"$el"}, Equivalent},
	{"dispatch", "Dispatching events", Magic, []string{"$dispatch"}, Equivalent},
	{"root", "The component root", Magic, []string{"$root"}, Equivalent},
	{"nexttick", "After the next render", Magic, []string{"$nextTick"}, Equivalent},
	{"store", "Shared state", Magic, []string{"$store", "Alpine.store"}, Equivalent},
	{"watch", "Watching a value", Magic, []string{"$watch"}, Workaround},
	{"alpine-data", "Reusable components", Global, []string{"Alpine.data"}, None},
	{"alpine-bind", "Reusable attribute bundles", Global, []string{"Alpine.bind"}, None},
```

Replace `featureOrder` with the final list:

```go
var featureOrder = []string{
	"x-bind", "x-bind (class object)", "x-bind (class, tabs)", "x-cloak", "x-data", "x-effect",
	"x-for", "x-for (filtering)", "x-html", "x-id", "x-if", "x-ignore", "x-init",
	"x-model", "x-model (filtering)", "x-modelable", "x-on (modifiers)", "x-on (.outside)",
	"x-ref", "x-show", "x-teleport", "x-text", "x-transition",
	"$data", "$dispatch", "$el", "$id", "$nextTick", "$refs", "$root", "$store", "$watch",
	"Alpine.bind", "Alpine.data", "Alpine.store",
}
```

- [ ] **Step 4: Write the fragments and notes**

`examples/el/alpine.html` (docs example inside an `x-data` wrapper):
```html
<div x-data>
    <button @click="$el.innerHTML = 'Hello World!'">Replace me with "Hello World!"</button>
</div>
```
`examples/el/hxlive.html`:
```html
<div>
    <button hx-on:click="this.innerHTML = 'Hello World!'">Replace me with "Hello World!"</button>
</div>
```
`examples/el/notes.md`:
```md
`$el` is the element the expression sits on. In `hx-on` and hx-live bindings that is plain `this`.
```

`examples/dispatch/alpine.html` (adapted: the docs alert):
```html
<div x-data="{ message: '' }" @notify="message = 'Notified!'">
    <button @click="$dispatch('notify')">Notify</button>
    <span x-text="message"></span>
</div>
```
`examples/dispatch/hxlive.html`:
```html
<div data-message="" hx-on:notify="data.message = 'Notified!'">
    <button hx-on:click="trigger('notify')">Notify</button>
    <span :text="data.message"></span>
</div>
```
`examples/dispatch/notes.md`:
```md
Adapted from the docs, which alert. `$dispatch` and `trigger()` both fire a bubbling `CustomEvent`, and both sides listen for it on an ancestor with the same attribute shape: `@notify` versus `hx-on:notify`.
```

`examples/root/alpine.html` (adapted: the docs alert):
```html
<div x-data data-message="Hello World!">
    <button @click="$el.textContent = $root.dataset.message">Say Hi</button>
</div>
```
`examples/root/hxlive.html`:
```html
<div data-message="Hello World!">
    <button hx-on:click="this.textContent = data.message">Say Hi</button>
</div>
```
`examples/root/notes.md`:
```md
Adapted from the docs, which alert. `$root` is the element carrying `x-data`. hx-live's `data.*` already resolves to the closest ancestor with that attribute, so the root lookup is implicit.
```

`examples/nexttick/alpine.html` (adapted: the docs log to the console):
```html
<div x-data="{ title: 'Hello', after: '' }">
    <button @click="title = 'Hello World!'; $nextTick(() => { after = $el.innerText })" x-text="title"></button>
    <span x-text="after"></span>
</div>
```
`examples/nexttick/hxlive.html`:
```html
<div data-title="Hello" data-after="">
    <button hx-on:click="data.title = 'Hello World!'; nextFrame().then(() => data.after = this.innerText)" :text="data.title"></button>
    <span :text="data.after"></span>
</div>
```
`examples/nexttick/notes.md`:
```md
Adapted from the docs, which log to the console. Both defer a read until after the pending re-render: the span shows the button's new text, not its old one. Alpine's `$nextTick` waits for its reactive flush; hx-live's `nextFrame()` waits for the next animation frame, after its mutation-driven recompute has run.
```

`examples/store/alpine.html` (adapted from the docs' darkMode store: two components share it):
```html
<div x-data>
    <button @click="$store.darkMode.toggle()">Toggle Dark Mode</button>
</div>

<div x-data :class="$store.darkMode.on && 'dark'">
    Content
</div>

<script>
    document.addEventListener('alpine:init', () => {
        Alpine.store('darkMode', {
            on: false,

            toggle() {
                this.on = ! this.on
            }
        })
    })
</script>
<style>
    .dark { background: #18181b; color: #fff; padding: 4px 8px; }
</style>
```
`examples/store/hxlive.html`:
```html
<div data-dark="false">
    <div>
        <button hx-on:click="data.dark = !data.dark">Toggle Dark Mode</button>
    </div>

    <div :class="{ dark: data.dark }">
        Content
    </div>
</div>
<style>
    .dark { background: #18181b; color: #fff; padding: 4px 8px; }
</style>
```
`examples/store/notes.md`:
```md
Adapted from the docs' dark-mode store so that two separate components share it. `Alpine.store` is a global reactive object registered in a script. In hx-live, shared state is a `data-*` attribute on whatever ancestor the components have in common, up to `<body>`; every `data.dark` inside resolves to it.
```

`examples/watch/alpine.html` (adapted: the docs log to the console):
```html
<div x-data="{ open: false, log: '' }" x-init="$watch('open', value => log = 'open is now ' + value)">
    <button @click="open = ! open">Toggle Open</button>
    <span x-text="log"></span>
</div>
```
`examples/watch/hxlive.html`:
```html
<div data-open="false" data-log="" hx-live="data.log = 'open is now ' + data.open">
    <button hx-on:click="data.open = !data.open">Toggle Open</button>
    <span :text="data.log"></span>
</div>
```
`examples/watch/notes.md`:
```md
Adapted from the docs, which log to the console. `$watch` runs a callback when one named property changes. hx-live has no per-property watcher; an `hx-live` expression is an effect that runs whenever anything changes, and it also runs once at load, so the hx-live span reads "open is now false" before any click while Alpine's starts empty.
```

`examples/alpine-data/alpine.html` (the docs' `dropdown` example with its placeholders filled in):
```html
<div x-data="dropdown">
    <button @click="toggle">Toggle</button>

    <div x-show="open">Contents...</div>
</div>

<script>
    document.addEventListener('alpine:init', () => {
        Alpine.data('dropdown', () => ({
            open: false,

            toggle() {
                this.open = ! this.open
            }
        }))
    })
</script>
```
`examples/alpine-data/notes.md`:
```md
`Alpine.data` registers a component definition that many elements can instantiate. hx-live has no component model on the client: reuse happens where the markup is produced. In this app the reusable unit is a gsx component that renders the `data-*` attribute and the bindings; the browser only ever sees the expanded result.
```

`examples/alpine-bind/alpine.html` (adapted from the docs' `Alpine.bind` example so it does something visible):
```html
<div x-data="{ clicks: 0 }">
    <button x-bind="SomeButton">Click</button>
    <span x-text="clicks"></span>
</div>

<script>
    document.addEventListener('alpine:init', () => {
        Alpine.bind('SomeButton', () => ({
            type: 'button',
            '@click'() {
                this.clicks++
            },
            ':disabled'() {
                return this.clicks >= 3
            },
        }))
    })
</script>
```
`examples/alpine-bind/notes.md`:
```md
Adapted from the docs so the bundle does something visible: the button counts to three and then disables itself. `Alpine.bind` packages attributes and listeners for reuse across elements. hx-live has no equivalent; an attribute bundle is a template concern, and a gsx component that emits `hx-on:click` and `:disabled` plays the same role.
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go vet ./examples/ && go test ./examples/ && go tool gsx generate && go test ./...
```
Expected: all `ok`.

- [ ] **Step 6: Commit**

```bash
git add examples
git commit -m "Add magic and global cards; complete the feature matrix

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 5: Matrix, grouped sidebar, status badges, none-row layout, frame 404

**Files:**
- Modify: `pages/index.gsx` (rewrite), `main.go`, `main_test.go`

**Interfaces:**
- Consumes: `examples.Features`, `examples.Groups`, `examples.ByGroup`, `Example.HasDemo`, `Example.Status`, `FeatureRow`.
- Produces: `pages.Index(exs []examples.Example, feats []examples.FeatureRow)`; `newHandler(v *vite.Vite, exs []examples.Example, feats []examples.FeatureRow) http.Handler`.

- [ ] **Step 1: Update the test server and add the tests**

In `main_test.go`, change `newTestServer` to compute the matrix:

```go
	exs, err := examples.Load()
	if err != nil {
		t.Fatal(err)
	}
	feats, err := examples.Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(v.Middleware(newHandler(v, exs, feats)))
```

In `TestIndexListsEveryExampleWithBothSources` (already relaxed in Task 3), add inside the outer `for _, ex := range exs` loop, after the per-lib loop:

```go
		if !ex.HasDemo() && strings.Contains(body, `src="/frame/hxlive/`+ex.Slug+`"`) {
			t.Errorf("%s: none row has an hxlive iframe", ex.Slug)
		}
```

Append two tests:

```go
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
	for _, f := range feats {
		if !strings.Contains(matrix, html.EscapeString(f.Feature)) {
			t.Errorf("matrix lacks feature %q", f.Feature)
		}
		if !strings.Contains(matrix, `href="#`+f.Slug+`"`) {
			t.Errorf("matrix lacks link to %s", f.Slug)
		}
		if !strings.Contains(matrix, string(f.Status)) {
			t.Errorf("matrix lacks status %q", f.Status)
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go tool gsx generate && go test .
```
Expected: build failure on `newHandler` arity.

- [ ] **Step 3: Update `main.go`**

`newHandler` takes the matrix and refuses hx-live frames for `none` rows:

```go
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
```

In `main()`, after `examples.Load()`:

```go
	feats, err := examples.Features(exs)
	if err != nil {
		log.Fatal(err)
	}
```
and pass `feats` to `newHandler`.

- [ ] **Step 4: Rewrite `pages/index.gsx`**

```gsx
package pages

import (
	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/ui"
)

// Index is the comparison page: a feature matrix, a grouped nav, and one card
// per example with the Alpine and hx-live demos side by side.
component Index(exs []examples.Example, feats []examples.FeatureRow) {
	{{ v := vite.FromContext(ctx) }}
	{{ assets := v.Entry("web/main.js") }}
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>hx-live vs Alpine</title>
			{ for _, href := range assets.CSS {
				<link rel="stylesheet" href={href}/>
			} }
			{ for _, src := range assets.Preloads {
				<link rel="modulepreload" href={src}/>
			} }
			{ for _, src := range assets.JS {
				<script type="module" src={src}></script>
			} }
		</head>
		<body class="bg-background text-foreground">
			<div class="mx-auto grid max-w-7xl grid-cols-[220px_1fr] gap-10 px-6 py-10">
				<nav class="sticky top-10 flex h-fit max-h-[calc(100vh-5rem)] flex-col gap-1 overflow-y-auto">
					<h1 class="mb-3 text-lg font-semibold">hx-live vs Alpine</h1>
					<ui.Button variant="ghost" size="sm" href="#matrix" class="h-auto justify-start whitespace-normal text-left">Feature matrix</ui.Button>
					{ for _, g := range examples.Groups {
						<h2 class="mt-3 mb-1 text-xs font-semibold uppercase text-muted-foreground">{ g.Label() }</h2>
						{ for _, ex := range examples.ByGroup(exs, g) {
							<ui.Button variant="ghost" size="sm" href={"#" + ex.Slug} class="h-auto justify-start whitespace-normal text-left">{ ex.Title }</ui.Button>
						} }
					} }
				</nav>
				<main class="flex flex-col gap-10">
					<p class="text-muted-foreground max-w-prose">
						Every Alpine.js core feature, each with its docs example and a like-for-like htmx 4 hx-live port
						where one exists. Every demo runs in its own iframe loading only its library. The source under each
						demo is the exact fragment inside that iframe.
					</p>
					<matrix feats={feats}/>
					{ for _, ex := range exs {
						<ui.Card id={ex.Slug}>
							<ui.CardHeader>
								<ui.CardTitle class="flex items-center gap-2">
									{ ex.Title }
									<statusBadge status={ex.Status}/>
								</ui.CardTitle>
								{ if ex.HasDemo() {
									<ui.CardDescription class="space-y-2">{ gsx.Raw(ex.NotesHTML) }</ui.CardDescription>
								} }
							</ui.CardHeader>
							<ui.CardContent class="grid grid-cols-2 gap-6 px-4">
								<column lib={examples.Alpine} ex={ex}/>
								{ if ex.HasDemo() {
									<column lib={examples.HxLive} ex={ex}/>
								} else {
									<noEquivalent ex={ex}/>
								} }
							</ui.CardContent>
						</ui.Card>
					} }
				</main>
			</div>
		</body>
	</html>
}

// matrix is the feature table: one line per Alpine feature, grouped.
component matrix(feats []examples.FeatureRow) {
	<table id="matrix" class="w-full text-sm">
		<thead>
			<tr class="text-left text-muted-foreground">
				<th class="py-1 pr-4 font-medium">Feature</th>
				<th class="py-1 pr-4 font-medium">Status</th>
				<th class="py-1 font-medium">Card</th>
			</tr>
		</thead>
		{ for _, g := range examples.Groups {
			<tbody>
				<tr>
					<th colspan="3" class="pt-4 pb-1 text-left text-xs font-semibold uppercase text-muted-foreground">{ g.Label() }</th>
				</tr>
				{ for _, f := range feats {
					{ if f.Group == g {
						<tr class="border-t">
							<td class="py-1 pr-4 font-mono text-xs">{ f.Feature }</td>
							<td class="py-1 pr-4"><statusBadge status={f.Status}/></td>
							<td class="py-1"><a class="underline" href={"#" + f.Slug}>{ f.Title }</a></td>
						</tr>
					} }
				} }
			</tbody>
		} }
	</table>
}

// statusBadge maps a status to a badge variant.
component statusBadge(status examples.Status) {
	{ if status == examples.Equivalent {
		<ui.Badge>{ string(status) }</ui.Badge>
	} else {
		{ if status == examples.Workaround {
			<ui.Badge variant="secondary">{ string(status) }</ui.Badge>
		} else {
			<ui.Badge variant="outline">{ string(status) }</ui.Badge>
		} }
	} }
}

// column is one side of a card: badge, live iframe, escaped source.
component column(lib examples.Lib, ex examples.Example) {
	{{ src := "/frame/" + string(lib) + "/" + ex.Slug }}
	<div class="flex min-w-0 flex-col gap-3">
		<ui.Badge variant="secondary">{ lib.Label() }</ui.Badge>
		<iframe
			data-frame
			src={src}
			title={ex.Title + " — " + lib.Label()}
			class="w-full rounded-md border bg-white box-content"
			style="height: 120px"
		></iframe>
		<pre class="overflow-x-auto rounded-md bg-muted p-3"><code>{ ex.Fragment(lib) }</code></pre>
	</div>
}

// noEquivalent is the right column of a card whose feature hx-live lacks.
component noEquivalent(ex examples.Example) {
	<div class="flex min-w-0 flex-col gap-3">
		<ui.Badge variant="outline">No hx-live equivalent</ui.Badge>
		<div class="space-y-2 text-sm text-muted-foreground">{ gsx.Raw(ex.NotesHTML) }</div>
	</div>
}
```

Notes for the implementer: if gsx rejects `else if`, the nested form above is already used; if it rejects `{ if ... }` directly inside a `{ for ... }` body, wrap the inner `<tr>` in a `<template>`-free helper component `matrixRow(f examples.FeatureRow, g examples.Group)` that does the comparison. If `size="sm"` is not a Button size, drop it. Keep behaviour and record any change in the report.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go tool gsx generate && go vet ./... && go test ./... && npm run build
```
Expected: all `ok`, build succeeds.

- [ ] **Step 6: Commit**

```bash
git add main.go main_test.go pages/index.gsx
git commit -m "Render the feature matrix, grouped nav and no-equivalent cards

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 6: Playwright smoke and parity for every new row

**Files:**
- Create: `e2e/smoke.spec.ts`
- Modify: `e2e/index.spec.ts`, `e2e/parity.spec.ts`
- Possibly modify: `examples/<slug>/hxlive.html` and `notes.md` for ports that fail in a real browser (never `alpine.html`, never an assertion)

**Interfaces:**
- Consumes: routes from Task 5; fragments from Tasks 2 to 4.

- [ ] **Step 1: Write the smoke spec**

`e2e/smoke.spec.ts`:
```ts
import { test, expect } from "@playwright/test";

test("every demo frame loads under both libraries without errors", async ({ page, context }) => {
  test.setTimeout(120_000);
  await page.goto("/");
  const srcs = await page
    .locator("iframe[data-frame]")
    .evaluateAll((els) => els.map((e) => e.getAttribute("src")!));
  expect(srcs.length).toBeGreaterThan(12);
  const hx = srcs.filter((s) => s.startsWith("/frame/hxlive/"));
  const al = srcs.filter((s) => s.startsWith("/frame/alpine/"));
  expect(hx.length).toBeLessThan(al.length); // none rows have no hx-live frame
  for (const src of srcs) {
    const p = await context.newPage();
    const errors: string[] = [];
    p.on("console", (m) => {
      if (m.type() === "error") errors.push(m.text());
    });
    p.on("pageerror", (e) => errors.push(String(e)));
    const res = await p.goto(src);
    expect(res?.status(), src).toBe(200);
    await expect(p.locator("body")).not.toBeEmpty();
    await p.waitForTimeout(150);
    expect(errors, src).toEqual([]);
    await p.close();
  }
});
```

- [ ] **Step 2: Make the index spec derive its count**

Replace the `toHaveCount(12)` line in `e2e/index.spec.ts` with:

```ts
  const count = await frames.count();
  expect(count).toBeGreaterThan(12);
  expect(count % 2).toBe(1); // 23 demo rows × 2 + 5 none rows × 1 = 51
```

- [ ] **Step 3: Add parity tests for the new interactive rows**

Append inside the `for (const lib of libs) { test.describe(lib, () => { ... }) }` block in `e2e/parity.spec.ts`, after the existing six tests:

```ts
    test("bind sets the placeholder", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/bind`);
      await expect(page.locator("input")).toHaveAttribute("placeholder", "Type here...");
      expect(errors).toEqual([]);
    });

    test("on reacts to the Enter key only", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/on`);
      const input = page.locator("input");
      await input.fill("abc");
      await expect(page.locator("span")).toHaveText("");
      await input.press("Enter");
      await expect(page.locator("span")).toHaveText("Enter pressed");
      expect(errors).toEqual([]);
    });

    test("init runs once at load", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/init`);
      await expect(page.locator("span")).toHaveText("Initialised!");
      expect(errors).toEqual([]);
    });

    test("html renders markup", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/html`);
      await expect(page.locator("span strong")).toHaveText("calebporzio");
      expect(errors).toEqual([]);
    });

    test("effect derives a value", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/effect`);
      await expect(page.locator("span")).toHaveText("5");
      await page.getByRole("button", { name: "Change Message" }).click();
      await expect(page.locator("span")).toHaveText("12");
      expect(errors).toEqual([]);
    });

    test("ignore leaves the subtree alone", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/ignore`);
      const spans = page.locator("span");
      await expect(spans.nth(0)).toHaveText("processed");
      await expect(spans.nth(1)).toHaveText("untouched");
      expect(errors).toEqual([]);
    });

    test("ref removes the referenced element", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/ref`);
      await expect(page.getByText("Hello")).toBeVisible();
      await page.getByRole("button", { name: "Remove Text" }).click();
      await expect(page.getByText("Hello")).toHaveCount(0);
      expect(errors).toEqual([]);
    });

    test("model mirrors the input", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/model`);
      await page.locator("input").fill("hi");
      await expect(page.locator("span")).toHaveText("hi");
      expect(errors).toEqual([]);
    });

    test("for appends to the list", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/for`);
      await expect(page.locator("li")).toHaveText(["Red", "Orange", "Yellow"]);
      await page.getByRole("button", { name: "Add Green" }).click();
      await expect(page.locator("li")).toHaveText(["Red", "Orange", "Yellow", "Green"]);
      expect(errors).toEqual([]);
    });

    test("if toggles the contents", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/if`);
      await expect(page.getByText("Contents...")).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(page.getByText("Contents...")).toBeVisible();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(page.getByText("Contents...")).toBeHidden();
      expect(errors).toEqual([]);
    });

    test("cloak hides the element and removes its marker", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/cloak`);
      await expect(page.getByText("This will not")).toBeHidden();
      await expect(page.locator("[x-cloak], [hx-cloak]")).toHaveCount(0);
      expect(errors).toEqual([]);
    });

    test("el is the current element", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/el`);
      await page.getByRole("button").click();
      await expect(page.getByRole("button")).toHaveText("Hello World!");
      expect(errors).toEqual([]);
    });

    test("dispatch reaches an ancestor listener", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/dispatch`);
      await page.getByRole("button", { name: "Notify" }).click();
      await expect(page.locator("span")).toHaveText("Notified!");
      expect(errors).toEqual([]);
    });

    test("root reads the component root", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/root`);
      await page.getByRole("button", { name: "Say Hi" }).click();
      await expect(page.getByRole("button")).toHaveText("Hello World!");
      expect(errors).toEqual([]);
    });

    test("nexttick reads after the re-render", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/nexttick`);
      await page.getByRole("button").click();
      await expect(page.getByRole("button")).toHaveText("Hello World!");
      await expect(page.locator("span")).toHaveText("Hello World!");
      expect(errors).toEqual([]);
    });

    test("store is shared across components", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/store`);
      const content = page.getByText("Content");
      await expect(content).not.toHaveClass(/\bdark\b/);
      await page.getByRole("button", { name: "Toggle Dark Mode" }).click();
      await expect(content).toHaveClass(/\bdark\b/);
      await page.getByRole("button", { name: "Toggle Dark Mode" }).click();
      await expect(content).not.toHaveClass(/\bdark\b/);
      expect(errors).toEqual([]);
    });

    test("watch reports the new value", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/watch`);
      await page.getByRole("button", { name: "Toggle Open" }).click();
      await expect(page.locator("span")).toHaveText("open is now true");
      await page.getByRole("button", { name: "Toggle Open" }).click();
      await expect(page.locator("span")).toHaveText("open is now false");
      expect(errors).toEqual([]);
    });
```

- [ ] **Step 4: Run the suite and fix hx-live ports, never assertions**

```bash
npm run e2e
```
Expected: 46 tests (2 index/smoke + 22 × 2 parity). Likely trouble spots and the fallback port for each, to be applied only if the hx-live test fails while the Alpine twin passes:

- **init**: if `hx-on:load` never fires for elements present at page load, use the extended form `hx-on="load -> data.message = 'Initialised!'"`; if that also fails, use `hx-on="htmx:load -> ..."`. Update the notes to name the event that works.
- **dispatch**: if the ancestor listener does not fire, pass `true` for bubbles: `trigger('notify', null, true)`. Update the notes.
- **nexttick**: if `data` or `this` is not reachable inside the `.then` callback, capture first: `hx-on:click="let el = this; data.title = 'Hello World!'; nextFrame().then(() => q(el).data.after = el.innerText)"`. Update the notes.
- **bind**: if `data.placeholder` reads `undefined`, the attribute name mapping differs; try `data['placeholder']` and, failing that, report the console output rather than guessing.
- **cloak**: if `[hx-cloak]` remains after load, replace the marker mechanism with `hx-on:load="attr['hx-cloak'] = false"` and update the notes.
- **for**: if `q('previous ul')` resolves nothing from the button, use `q('#colors')` with `id="colors"` on the `<ul>` and mention the id in the notes.
- **model**: if the span lags because of the input debounce, the assertion still passes (Playwright retries); do not add waits to the fragment.

Any change to an hx-live fragment must keep the Alpine DOM shape and be reflected in that row's `notes.md`.

- [ ] **Step 5: Go tests still green**

```bash
go tool gsx generate && go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add e2e examples
git commit -m "Add smoke and parity tests covering every catalogue row

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 7: README and full verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the README**

Replace the first two paragraphs of `README.md` with:

```md
# hx-live vs Alpine

Every Alpine.js core feature (18 directives, 9 magics, 3 globals) beside its
htmx 4 `hx-live` counterpart: the Alpine docs example on the left, a
like-for-like port on the right where one exists, and an honest note where
one does not. A matrix at the top of the page gives each feature a status:
`equivalent`, `workaround`, or `none`. Raw material for a blog post.

Every demo runs in its own iframe that loads exactly one library, because
Alpine and hx-live both claim the `:attr` shorthand and hx-live disables its
short form when it detects Alpine. The hx-live ports target htmx 4.0.0's typed
state bags (`data.*`, `aria.*`, `class.*`).
```

In the Layout section, change the first bullet to:

```md
- `examples/<slug>/alpine.html`, `notes.md`, and `hxlive.html` for rows with a port: the content. `examples/examples.go` holds the manifest (group, features, status).
```

- [ ] **Step 2: Full verification**

```bash
go tool gsx generate && go vet ./... && go test ./... && npm run build && npm run e2e
lsof -i :8899
```
Expected: Go ok, Playwright 46 passed, no listener on 8899.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "Describe the feature catalogue in the README

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```
