# hx-live vs Alpine Comparison App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A gsx web app that renders six Alpine.js examples and their like-for-like htmx 4 `hx-live` ports side by side, each demo running live in its own iframe next to the exact markup that produced it.

**Architecture:** Raw HTML fragments live on disk under `examples/<slug>/` and are embedded by a small `examples` package. Two gsx pages consume them: `pages.Frame` renders one fragment inside a minimal document that loads exactly one library (served as its own Vite entry), and `pages.Index` composes the comparison page from gsxui cards, iframes pointing at the frame route, and escaped source blocks. Plain `net/http` with the `gsxhq/vite` middleware, mirroring `../gsxhq/hx-live-spike`.

**Tech Stack:** Go 1.26, gsx (published module, `go tool gsx`), gsxui (card, badge, button), gsxhq/vite v0.3.2, Vite 6 + Tailwind 4, `htmx.org@4.0.0`, `alpinejs@3.17.1`, goldmark for notes, Playwright for parity tests.

**Spec:** `docs/superpowers/specs/2026-09-04-hx-live-vs-alpine-design.md`

## Global Constraints

- Module path: `github.com/jackielii/hx-live-vs-alpine`.
- Library versions pinned exactly: `htmx.org@4.0.0`, `alpinejs@3.17.1`. No CDN at runtime.
- Alpine and hx-live never share a document. The outer bundle (`web/main.js`) imports neither. `web/frame-alpine.js` imports only Alpine; `web/frame-hxlive.js` imports only htmx + hx-live.
- In `web/frame-hxlive.js`, `./htmx-setup.js` (which sets `window.htmx`) must be imported BEFORE `htmx.org/dist/ext/hx-live.js`.
- The fragment shown in the `<pre><code>` block and the fragment injected into the iframe are the same bytes from the same file.
- Any CSS class a fragment relies on (`.active`, `.on`, `.fade`) is styled by a `<style>` inside that fragment, so the shown source is complete.
- Slugs, in display order: `counter`, `dropdown`, `search`, `tabs`, `transition`, `classbind`.
- Unexported identifiers unless something must cross a package boundary or be serialised.
- Commit messages end with `Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb`.
- Two deliberate deviations from the spec's layout, both simplifications: (1) gsx is used as the published module (as the spike does), not via a `replace` to `../gsxhq/gsx`; (2) the manifest lives in package `examples` (directory `examples/`) rather than a root `examples.go`, because `pages` must import it and Go cannot import package `main`. gsxui's `sidebar` is not used (420 lines, renders children twice); the nav is a plain `<nav>` of ghost buttons.

---

## File structure

| Path | Responsibility |
|---|---|
| `go.mod`, `package.json`, `vite.config.ts`, `gsx.toml`, `gsxui.json` | scaffold from `gsx init` + `gsxui init`; only edits: three Rollup inputs, npm deps, scripts |
| `main.go` | `vite.New`, `newHandler(v, exs)` builds the mux, `main` wires port + graceful shutdown |
| `main_test.go` | `httptest` tests over `newHandler` with a stub prod manifest |
| `examples/examples.go` | `Lib`, `Example`, `Load()`; embeds `*/alpine.html`, `*/hxlive.html`, `*/notes.md`; renders notes to HTML with goldmark |
| `examples/examples_test.go` | `Load` returns the six slugs in order with non-empty fields |
| `examples/<slug>/{alpine.html,hxlive.html,notes.md}` | the content |
| `pages/frame.gsx` | `Frame(lib, ex)`: iframe shell document |
| `pages/index.gsx` | `Index(exs)`, `column(lib, ex)`: outer comparison page |
| `ui/` | gsxui-vendored `card.gsx`, `badge.gsx`, `button.gsx`, `merge/`, `icon/` (untouched) |
| `web/main.js` | outer bundle: devpanel, gsxui runtime + css, `frames.js`, `style.css` |
| `web/frames.js` | parent-side `message` listener that sets iframe heights |
| `web/frame-alpine.js` | Alpine entry |
| `web/frame-hxlive.js` | htmx + hx-live entry |
| `web/htmx-setup.js` | `window.htmx = htmx` |
| `web/frame.css` | classless base styles for demos |
| `web/frame-resize.js` | child-side ResizeObserver → `postMessage` |
| `e2e/playwright.config.ts`, `e2e/parity.spec.ts` | parity tests parameterised over both libs |

---

### Task 1: Scaffold the project

**Files:**
- Create (via tools): `go.mod`, `go.sum`, `package.json`, `package-lock.json`, `vite.config.ts`, `gsx.toml`, `gsxui.json`, `gsxui.preset.json`, `main.go`, `app.gsx`, `.env`, `.env.example`, `.gitignore`, `dist/.gitkeep`, `public/*`, `web/*`, `ui/*`
- Modify: `.gitignore`, `vite.config.ts`, `package.json`, `web/main.js`
- Delete: `app.gsx`, `web/counter.js`

**Interfaces:**
- Produces: a building project where `go tool gsx generate && go build ./... && npm run build` succeed. `main.go` still serves the scaffold's page for now (replaced in Task 3).

- [ ] **Step 1: Run gsx init into the current directory**

Both `gsx` and `gsxui` binaries are already installed at `~/go/bin`. The directory currently holds only `.git`, `.gitignore`, `.remember/`, `docs/`; `gsx init` only refuses when `go.mod` or `package.json` exists.

```bash
cd /Users/jackieli/personal/hx-live-vs-alpine
gsx init . --module github.com/jackielii/hx-live-vs-alpine --yes
```

Expected: scaffold files written, then `go get -tool`, `go mod tidy`, `npm install` run without prompting. `ls` shows `app.gsx main.go go.mod package.json vite.config.ts web public dist .env`.

- [ ] **Step 2: Run gsxui init and add the three components**

```bash
gsxui init
gsxui add card badge button
go tool gsx generate
```

Expected: `gsx.toml` contains `class_merger = "github.com/jackielii/hx-live-vs-alpine/ui/merge.Merge"`; `ui/card.gsx`, `ui/badge.gsx`, `ui/button.gsx`, `ui/merge/merge.go`, `ui/icon/` exist; `web/gsxui/` exists; `vite.config.ts` now includes `tailwindcss()`; `web/main.js` now imports `./gsxui/index.js` and `./gsxui/index.css`.

- [ ] **Step 3: Restore the `.remember/` ignore and add Playwright ignores**

`gsx init` overwrote `.gitignore`. Replace it with:

```
/node_modules
/dist/*
!/dist/.gitkeep
/bin/
*.x.go
.env
/test-results/
/playwright-report/
.remember/
```

- [ ] **Step 4: Install the pinned libraries and Playwright**

```bash
npm install --save-exact alpinejs@3.17.1 htmx.org@4.0.0
npm install --save-dev @playwright/test@^1.62.1
npx playwright install chromium
```

Expected: `package.json` `dependencies` has `"alpinejs": "3.17.1"` and `"htmx.org": "4.0.0"` (no caret).

- [ ] **Step 5: Add the three Rollup inputs and npm scripts**

In `vite.config.ts` change the `build` block to:

```ts
    build: {
      manifest: true,
      outDir: "dist",
      rollupOptions: {
        input: ["web/main.js", "web/frame-alpine.js", "web/frame-hxlive.js"],
      },
    },
```

In `package.json` set `scripts` to:

```json
  "scripts": {
    "dev": "go tool gsx dev",
    "build": "vite build",
    "e2e:serve": "npm run build && go tool gsx generate && GO_PORT=8899 go run .",
    "e2e": "playwright test -c e2e"
  },
```

- [ ] **Step 6: Create the frame entries and shared web files**

`web/htmx-setup.js`:
```js
import htmx from "htmx.org";
window.htmx = htmx;
```

`web/frame-resize.js`:
```js
// Runs inside each demo iframe. Reports the document height to the parent so
// the outer page can size the iframe to its content.
function report() {
  const height = Math.ceil(document.documentElement.getBoundingClientRect().height);
  window.parent.postMessage({ type: "hx-vs-alpine:height", height }, "*");
}
new ResizeObserver(report).observe(document.documentElement);
window.addEventListener("load", report);
report();
```

`web/frame.css`:
```css
body {
  margin: 0;
  padding: 16px;
  font: 15px/1.5 system-ui, -apple-system, "Segoe UI", sans-serif;
  color: #18181b;
  background: #fff;
}
button {
  font: inherit;
  padding: 4px 12px;
  border: 1px solid #a1a1aa;
  border-radius: 6px;
  background: #f4f4f5;
  cursor: pointer;
}
button:hover { background: #e4e4e7; }
input {
  font: inherit;
  padding: 4px 8px;
  border: 1px solid #a1a1aa;
  border-radius: 6px;
}
ul { margin: 8px 0 0; padding-left: 20px; }
nav { display: flex; gap: 4px; margin-bottom: 8px; }
```

`web/frame-alpine.js`:
```js
import "./frame.css";
import "./frame-resize.js";
import Alpine from "alpinejs";
window.Alpine = Alpine;
Alpine.start();
```

`web/frame-hxlive.js`:
```js
import "./frame.css";
import "./frame-resize.js";
// Order matters: hx-live.js reads the global `htmx` at eval time.
import "./htmx-setup.js";
import "htmx.org/dist/ext/hx-live.js";
```

`web/frames.js`:
```js
// Runs in the outer page. Each demo iframe posts its content height; match the
// sender to its iframe and set the height so nothing clips.
window.addEventListener("message", (e) => {
  if (!e.data || e.data.type !== "hx-vs-alpine:height") return;
  for (const frame of document.querySelectorAll("iframe[data-frame]")) {
    if (frame.contentWindow === e.source) frame.style.height = e.data.height + "px";
  }
});
```

Replace `web/main.js` with:
```js
// gsx dev panel: Cmd-D / Ctrl-D
import "virtual:gsx-devpanel";
import "./gsxui/index.js";
import "./frames.js";
import "./gsxui/index.css";
import "./style.css";
```

Replace `web/style.css` with:
```css
pre code { font-size: 0.8125rem; line-height: 1.5; }
```

Delete `web/counter.js`.

- [ ] **Step 7: Verify the scaffold builds**

```bash
go tool gsx generate && go build ./... && npm run build
ls dist/.vite/manifest.json
```

Expected: no errors; the manifest lists keys `web/main.js`, `web/frame-alpine.js`, `web/frame-hxlive.js`. (`app.gsx` is still the scaffold's page and `main.go` still references it; that is fine for this task.)

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "Scaffold gsx + gsxui project with pinned Alpine and htmx frame entries

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 2: The `examples` package and the six fragments

**Files:**
- Create: `examples/examples.go`, `examples/examples_test.go`
- Create: `examples/{counter,dropdown,search,tabs,transition,classbind}/{alpine.html,hxlive.html,notes.md}`

**Interfaces:**
- Produces:
  ```go
  package examples
  type Lib string
  const ( Alpine Lib = "alpine"; HxLive Lib = "hxlive" )
  func ParseLib(s string) (Lib, bool)
  func (l Lib) Label() string   // "Alpine.js" | "htmx hx-live"
  func (l Lib) Entry() string   // "web/frame-alpine.js" | "web/frame-hxlive.js"
  type Example struct { Slug, Title, Alpine, HxLive, NotesHTML string }
  func (e Example) Fragment(l Lib) string
  func Load() ([]Example, error)             // six, in display order
  func Find(exs []Example, slug string) (Example, bool)
  ```

- [ ] **Step 1: Write the failing test**

`examples/examples_test.go`:
```go
package examples

import (
	"strings"
	"testing"
)

func TestLoadReturnsSixExamplesInOrder(t *testing.T) {
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"counter", "dropdown", "search", "tabs", "transition", "classbind"}
	if len(exs) != len(want) {
		t.Fatalf("got %d examples, want %d", len(exs), len(want))
	}
	for i, ex := range exs {
		if ex.Slug != want[i] {
			t.Errorf("index %d: slug %q, want %q", i, ex.Slug, want[i])
		}
		if ex.Title == "" {
			t.Errorf("%s: empty title", ex.Slug)
		}
		if !strings.Contains(ex.Alpine, "x-data") {
			t.Errorf("%s: alpine fragment lacks x-data", ex.Slug)
		}
		if strings.Contains(ex.HxLive, "x-data") {
			t.Errorf("%s: hxlive fragment contains x-data", ex.Slug)
		}
		if strings.TrimSpace(ex.HxLive) == "" {
			t.Errorf("%s: empty hxlive fragment", ex.Slug)
		}
		if !strings.Contains(ex.NotesHTML, "<p>") {
			t.Errorf("%s: notes not rendered to HTML: %q", ex.Slug, ex.NotesHTML)
		}
	}
}

func TestFragmentAndFind(t *testing.T) {
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
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

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./examples/
```
Expected: FAIL, "undefined: Load" (package has no Go file yet, so the failure is a build error).

- [ ] **Step 3: Write the package**

```bash
go get github.com/yuin/goldmark@latest
```

`examples/examples.go`:
```go
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
```

- [ ] **Step 4: Write the six fragment pairs and notes**

`examples/counter/alpine.html`:
```html
<div x-data="{ count: 0 }">
    <button x-on:click="count++">Increment</button>
    <span x-text="count"></span>
</div>
```
`examples/counter/hxlive.html`:
```html
<div data-count="0">
    <button hx-on:click="data.count++">Increment</button>
    <span :text="data.count"></span>
</div>
```
`examples/counter/notes.md`:
```md
Alpine keeps `count` in a reactive object declared by `x-data`. hx-live keeps it in the DOM: `data.count` reads and writes the closest `data-count` attribute, JSON round-tripped so the number stays a number. `:text` re-runs after any DOM mutation, so the span follows the attribute.
```

`examples/dropdown/alpine.html`:
```html
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle</button>

    <div x-show="open" @click.outside="open = false">Contents...</div>
</div>
```
`examples/dropdown/hxlive.html`:
```html
<div>
    <button aria-expanded="false" hx-on:click="toggle('aria-expanded')">Toggle</button>

    <div :hidden="!q('previous button').attr('aria-expanded')"
         hx-on="click from:outside -> q('previous button').attr('aria-expanded', false)">Contents...</div>
</div>
```
`examples/dropdown/notes.md`:
```md
Open state lives in `aria-expanded` on the button, which doubles as the accessibility contract. `attr('aria-expanded')` reads back a boolean, so `:hidden` is a plain negation. Both libraries attach the outside listener on `document`, so the button's own click handler runs first in both; toggling open then closing does not fight.
```

`examples/search/alpine.html`:
```html
<div
    x-data="{
        search: '',

        items: ['foo', 'bar', 'baz'],

        get filteredItems() {
            return this.items.filter(
                i => i.startsWith(this.search)
            )
        }
    }"
>
    <input x-model="search" placeholder="Search...">

    <ul>
        <template x-for="item in filteredItems" :key="item">
            <li x-text="item"></li>
        </template>
    </ul>
</div>
```
`examples/search/hxlive.html`:
```html
<div>
    <input placeholder="Search...">

    <ul hx-live="for (let li of q('li in this')) li.hidden = !li.textContent.startsWith(q('previous input').value)">
        <li>foo</li>
        <li>bar</li>
        <li>baz</li>
    </ul>
</div>
```
`examples/search/notes.md`:
```md
This is the one that changes model, not just syntax. hx-live has no loop primitive and no two-way binding: the list is already HTML, so filtering means hiding rows, and the input's own `.value` is the state. Alpine derives DOM from data; hx-live derives attributes from DOM. A larger or server-owned list would be an htmx request, not client-side filtering.
```

`examples/tabs/alpine.html`:
```html
<div x-data="{ tab: 'a' }">
    <nav>
        <button @click="tab = 'a'" :class="{ active: tab === 'a' }">A</button>
        <button @click="tab = 'b'" :class="{ active: tab === 'b' }">B</button>
    </nav>
    <div x-show="tab === 'a'">Panel A</div>
    <div x-show="tab === 'b'">Panel B</div>
</div>
<style>
    .active { background: #18181b; color: #fff; border-color: #18181b; }
</style>
```
`examples/tabs/hxlive.html`:
```html
<div data-tab="a">
    <nav>
        <button hx-on:click="data.tab = 'a'" :.active="data.tab === 'a'">A</button>
        <button hx-on:click="data.tab = 'b'" :.active="data.tab === 'b'">B</button>
    </nav>
    <div :hidden="data.tab !== 'a'">Panel A</div>
    <div :hidden="data.tab !== 'b'">Panel B</div>
</div>
<style>
    .active { background: #18181b; color: #fff; border-color: #18181b; }
</style>
```
`examples/tabs/notes.md`:
```md
Closest like-for-like: the selected tab is a `data-tab` attribute on the wrapper, `:.active` binds one class, `:hidden` swaps panels. The more idiomatic hx-live shape is `role="tab"` buttons with `take('aria-selected', '[role=tab]')`, which moves the attribute between peers and drives CSS off `[aria-selected=true]`.
```

`examples/transition/alpine.html`:
```html
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle</button>

    <div x-show="open" x-transition>
        Hello 👋
    </div>
</div>
```
`examples/transition/hxlive.html`:
```html
<div data-open="false">
    <button hx-on:click="data.open = !data.open">Toggle</button>

    <div class="fade" :.open="data.open">
        Hello 👋
    </div>
</div>
<style>
    .fade { opacity: 0; visibility: hidden; transition: opacity .3s, visibility .3s; }
    .fade.open { opacity: 1; visibility: visible; }
</style>
```
`examples/transition/notes.md`:
```md
hx-live has no transition directive. Enter and leave are delegated to CSS: the binding only toggles a class, and the stylesheet animates opacity while flipping visibility so the hidden element stays out of the accessibility tree. The `<style>` is part of the shown source on purpose.
```

`examples/classbind/alpine.html`:
```html
<div x-data="{ pressed: false }">
    <button @click="pressed = ! pressed" :class="{ on: pressed }">Bold</button>
</div>
<style>
    .on { font-weight: bold; }
</style>
```
`examples/classbind/hxlive.html`:
```html
<div>
    <button aria-pressed="false"
            hx-on:click="toggle('aria-pressed')"
            :class="{ on: attr('aria-pressed') }">Bold</button>
</div>
<style>
    .on { font-weight: bold; }
</style>
```
`examples/classbind/notes.md`:
```md
Same object-form `:class` in both. Alpine's condition is a property; hx-live's is the element's own `aria-pressed`, toggled by `toggle('aria-pressed')`, which flips the string between `"true"` and `"false"` while `attr()` reads it back as a boolean.
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./examples/
```
Expected: `ok  github.com/jackielii/hx-live-vs-alpine/examples`.

- [ ] **Step 6: Commit**

```bash
git add examples go.mod go.sum
git commit -m "Add examples package with six Alpine/hx-live fragment pairs

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 3: Frame page and HTTP handler

**Files:**
- Create: `pages/frame.gsx`, `main_test.go`
- Modify: `main.go` (replace scaffold routing)
- Delete: `app.gsx`

**Interfaces:**
- Consumes: `examples.Load`, `examples.ParseLib`, `examples.Find`, `Lib.Entry`, `Example.Fragment`.
- Produces: `func newHandler(v *vite.Vite, exs []examples.Example) http.Handler` in package main; `pages.Frame(lib examples.Lib, ex examples.Example) gsx.Node`; route `GET /frame/{lib}/{slug}`. `GET /` is a temporary placeholder until Task 4.

- [ ] **Step 1: Write the failing tests**

`main_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go tool gsx generate && go test .
```
Expected: FAIL with "undefined: newHandler".

- [ ] **Step 3: Write `pages/frame.gsx`**

```gsx
// Package pages holds the gsx pages. Each renders its own <html> shell because
// Go cannot import package main.
package pages

import (
	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
)

// Frame is the document loaded inside one demo iframe. It loads exactly one
// library's Vite entry and injects the raw fragment unchanged.
component Frame(lib examples.Lib, ex examples.Example) {
	{{ v := vite.FromContext(ctx) }}
	{{ assets := v.Entry(lib.Entry()) }}
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ ex.Title } — { lib.Label() }</title>
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
		<body>
			{ gsx.Raw(ex.Fragment(lib)) }
		</body>
	</html>
}
```

- [ ] **Step 4: Rewrite `main.go`**

Replace the whole file (keep the scaffold's graceful-shutdown shape):

```go
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
func newHandler(v *vite.Vite, exs []examples.Example) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/public/", http.FileServerFS(publicFS))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if !v.Dev() {
		mux.Handle("/static/", v.StaticHandler())
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, pages.Index(exs))
	})
	mux.HandleFunc("GET /frame/{lib}/{slug}", func(w http.ResponseWriter, r *http.Request) {
		lib, ok := examples.ParseLib(r.PathValue("lib"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		ex, ok := examples.Find(exs, r.PathValue("slug"))
		if !ok {
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

	port := cmp.Or(os.Getenv("GO_PORT"), "7777")
	srv := &http.Server{Addr: ":" + port, Handler: v.Middleware(newHandler(v, exs))}

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
```

Until Task 4 exists, add a placeholder `pages/index.gsx` so this compiles:

```gsx
package pages

import "github.com/jackielii/hx-live-vs-alpine/examples"

// Index is the comparison page. Filled in by the next task.
component Index(exs []examples.Example) {
	<!DOCTYPE html>
	<html lang="en"><body><p>{ len(exs) } examples</p></body></html>
}
```

Delete `app.gsx` (and its generated `app.x.go` if present).

- [ ] **Step 5: Run tests to verify they pass**

```bash
go tool gsx generate && go vet ./... && go test ./...
```
Expected: all `ok`.

- [ ] **Step 6: Smoke it in the browser once**

```bash
npm run dev
```
Open the printed URL plus `/frame/hxlive/counter` and `/frame/alpine/counter`. Click Increment in each; the number increments. Check the browser console: no errors (the hx-live frame must NOT print the "Alpine detected" warning). Stop the dev server.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "Serve each demo fragment in its own single-library frame

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 4: The comparison page

**Files:**
- Modify: `pages/index.gsx` (replace placeholder), `main_test.go` (add test)

**Interfaces:**
- Consumes: `examples.Example` fields, `Lib.Label`, `Example.Fragment`, gsxui `ui.Card`, `ui.CardHeader`, `ui.CardTitle`, `ui.CardDescription`, `ui.CardContent`, `ui.Badge`, `ui.Button`.
- Produces: `pages.Index(exs []examples.Example)` rendering a sidebar nav, one card per example with id = slug, two columns each holding a `data-frame` iframe at `/frame/{lib}/{slug}` and the escaped fragment.

- [ ] **Step 1: Write the failing test**

Append to `main_test.go`:
```go
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
		for _, lib := range []examples.Lib{examples.Alpine, examples.HxLive} {
			src := "/frame/" + string(lib) + "/" + ex.Slug
			if !strings.Contains(body, `src="`+src+`"`) {
				t.Errorf("%s: no iframe for %s", ex.Slug, lib)
			}
			escaped := html.EscapeString(ex.Fragment(lib))
			if !strings.Contains(body, escaped) {
				t.Errorf("%s: escaped %s fragment not shown", ex.Slug, lib)
			}
		}
		if strings.Contains(body, ex.HxLive) {
			t.Errorf("%s: raw hxlive fragment leaked unescaped into the index", ex.Slug)
		}
	}
}
```
Add `"html"` to the test file's imports.

- [ ] **Step 2: Run test to verify it fails**

```bash
go tool gsx generate && go test .
```
Expected: FAIL on "no anchor" / "no iframe".

- [ ] **Step 3: Write `pages/index.gsx`**

```gsx
package pages

import (
	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/ui"
)

// Index is the comparison page: a nav of anchors and one card per example
// with the Alpine and hx-live demos side by side.
component Index(exs []examples.Example) {
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
				<nav class="sticky top-10 flex h-fit flex-col gap-1">
					<h1 class="mb-3 text-lg font-semibold">hx-live vs Alpine</h1>
					{ for _, ex := range exs {
						<ui.Button variant="ghost" href={"#" + ex.Slug} class="justify-start">{ ex.Title }</ui.Button>
					} }
				</nav>
				<main class="flex flex-col gap-10">
					<p class="text-muted-foreground max-w-prose">
						Six examples from the Alpine.js docs, each ported like-for-like to htmx 4's hx-live extension.
						Every demo runs in its own iframe loading only its library. The source under each demo is the
						exact fragment inside that iframe.
					</p>
					{ for _, ex := range exs {
						<ui.Card id={ex.Slug}>
							<ui.CardHeader>
								<ui.CardTitle>{ ex.Title }</ui.CardTitle>
								<ui.CardDescription class="prose prose-sm max-w-none">{ gsx.Raw(ex.NotesHTML) }</ui.CardDescription>
							</ui.CardHeader>
							<ui.CardContent class="grid grid-cols-2 gap-6 px-4">
								<column lib={examples.Alpine} ex={ex}/>
								<column lib={examples.HxLive} ex={ex}/>
							</ui.CardContent>
						</ui.Card>
					} }
				</main>
			</div>
		</body>
	</html>
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
			class="w-full rounded-md border bg-white"
			style="height: 120px"
		></iframe>
		<pre class="overflow-x-auto rounded-md bg-muted p-3"><code>{ ex.Fragment(lib) }</code></pre>
	</div>
}
```

Notes for the implementer: `id={ex.Slug}` and `class=...` on gsxui components land in their `attrs` passthrough. If the gsx compiler rejects a lowercase component name in markup (`<column .../>`), rename it `Column` in both places. If `prose` classes have no effect (Tailwind typography plugin not installed), drop them; the notes are a single paragraph and read fine unstyled.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go tool gsx generate && go vet ./... && go test ./...
```
Expected: all `ok`.

- [ ] **Step 5: Look at it**

```bash
npm run dev
```
Open the printed URL. Expect: nav on the left, six cards, each with two badges, two iframes that grow to fit their content within a second of load, and two source blocks. Open the dropdown demo in both columns: the iframe grows to show "Contents...". Stop the dev server.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "Add side-by-side comparison page with live frames and source

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 5: Playwright parity tests

**Files:**
- Create: `e2e/playwright.config.ts`, `e2e/parity.spec.ts`

**Interfaces:**
- Consumes: routes `/frame/{lib}/{slug}`; the npm `e2e:serve` and `e2e` scripts from Task 1.

- [ ] **Step 1: Write the config**

`e2e/playwright.config.ts`:
```ts
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  use: { baseURL: "http://localhost:8899" },
  webServer: {
    command: "npm run e2e:serve",
    port: 8899,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
```

- [ ] **Step 2: Write the parity spec**

Every test runs once per library with identical assertions. Selectors are role- or text-based so they match both DOM shapes.

`e2e/parity.spec.ts`:
```ts
import { test, expect, type Page } from "@playwright/test";

const libs = ["alpine", "hxlive"] as const;

function collectErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error") errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(String(e)));
  return errors;
}

for (const lib of libs) {
  test.describe(lib, () => {
    test("counter increments", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/counter`);
      const count = page.locator("span");
      await expect(count).toHaveText("0");
      await page.getByRole("button", { name: "Increment" }).click();
      await expect(count).toHaveText("1");
      await page.getByRole("button", { name: "Increment" }).click();
      await expect(count).toHaveText("2");
      expect(errors).toEqual([]);
    });

    test("dropdown toggles and closes on outside click", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/dropdown`);
      const contents = page.getByText("Contents...");
      await expect(contents).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(contents).toBeVisible();
      // body has 16px padding; (2,2) is padding, outside every element.
      await page.locator("body").click({ position: { x: 2, y: 2 } });
      await expect(contents).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(contents).toBeVisible();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(contents).toBeHidden();
      expect(errors).toEqual([]);
    });

    test("search filters the list", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/search`);
      const visible = page.locator("li:visible");
      await expect(visible).toHaveCount(3);
      await page.getByPlaceholder("Search...").fill("ba");
      await expect(visible).toHaveCount(2);
      await expect(visible).toHaveText(["bar", "baz"]);
      await page.getByPlaceholder("Search...").fill("baz");
      await expect(visible).toHaveCount(1);
      await page.getByPlaceholder("Search...").fill("");
      await expect(visible).toHaveCount(3);
      expect(errors).toEqual([]);
    });

    test("tabs switch panels and active class", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/tabs`);
      const a = page.getByRole("button", { name: "A" });
      const b = page.getByRole("button", { name: "B" });
      await expect(page.getByText("Panel A")).toBeVisible();
      await expect(page.getByText("Panel B")).toBeHidden();
      await expect(a).toHaveClass(/active/);
      await expect(b).not.toHaveClass(/active/);
      await b.click();
      await expect(page.getByText("Panel B")).toBeVisible();
      await expect(page.getByText("Panel A")).toBeHidden();
      await expect(b).toHaveClass(/active/);
      await expect(a).not.toHaveClass(/active/);
      expect(errors).toEqual([]);
    });

    test("transition shows and hides", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/transition`);
      const hello = page.getByText("Hello");
      await expect(hello).toBeHidden();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(hello).toBeVisible();
      await page.getByRole("button", { name: "Toggle" }).click();
      await expect(hello).toBeHidden();
      expect(errors).toEqual([]);
    });

    test("class binding toggles .on", async ({ page }) => {
      const errors = collectErrors(page);
      await page.goto(`/frame/${lib}/classbind`);
      const bold = page.getByRole("button", { name: "Bold" });
      await expect(bold).not.toHaveClass(/\bon\b/);
      await bold.click();
      await expect(bold).toHaveClass(/\bon\b/);
      await bold.click();
      await expect(bold).not.toHaveClass(/\bon\b/);
      expect(errors).toEqual([]);
    });
  });
}
```

- [ ] **Step 3: Run the suite**

```bash
npm run e2e
```
Expected: 12 passed (6 tests × 2 libraries). If an hx-live test fails while its Alpine twin passes, the port is wrong, not the test: fix the fragment in `examples/<slug>/hxlive.html`, note the reason in that example's `notes.md`, and re-run. Do not weaken the assertion.

- [ ] **Step 4: Run the Go tests once more (fragments may have changed)**

```bash
go tool gsx generate && go test ./...
```
Expected: all `ok`.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "Add Playwright parity tests running every example under both libraries

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```

---

### Task 6: README and final check

**Files:**
- Modify: `README.md` (scaffold's) 

- [ ] **Step 1: Replace `README.md`**

```md
# hx-live vs Alpine

Six examples from the Alpine.js docs, each ported like-for-like to htmx 4's
`hx-live` extension, rendered side by side with the source that runs them.
Raw material for a blog post.

Every demo runs in its own iframe that loads exactly one library, because
Alpine and hx-live both claim the `:attr` shorthand and hx-live disables its
short form when it detects Alpine.

## Run

    npm install
    npm run dev          # gsx dev: Vite + Go with reload

## Test

    go tool gsx generate && go test ./...   # routes and manifest
    npm run e2e                             # Playwright parity, both libraries

## Layout

- `examples/<slug>/alpine.html`, `hxlive.html`, `notes.md`: the content.
- `pages/frame.gsx`: the single-library iframe document.
- `pages/index.gsx`: the comparison page.
- `web/frame-alpine.js`, `web/frame-hxlive.js`: one Vite entry per library.

Pinned: `htmx.org@4.0.0`, `alpinejs@3.17.1`.
```

- [ ] **Step 2: Full verification**

```bash
go tool gsx generate && go vet ./... && go test ./... && npm run build && npm run e2e
```
Expected: all pass. Then `npm run dev`, open the page, scroll through all six cards, confirm both columns of every card work by hand.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "Document how to run and test the comparison app

Claude-Session: https://claude.ai/code/session_01FQkGtQ6t7hpCVjGzYD5ugb"
```
