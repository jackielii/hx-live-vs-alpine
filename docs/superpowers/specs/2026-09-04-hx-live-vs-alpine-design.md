# hx-live vs Alpine: side-by-side comparison app

**Date:** 2026-09-04
**Module:** `github.com/jackielii/hx-live-vs-alpine`
**Status:** approved design, pre-implementation

## Goal

A gsx web app that renders six small UI examples twice, side by side: once
written in Alpine.js, once written in htmx 4's `hx-live` extension. Each column
shows the working widget and the exact markup that produced it. The examples,
their behaviour parity, and the per-example notes are the raw material for a
later blog post comparing the two approaches.

Non-goals: a static export, a general component library, server-driven
(htmx swap) examples, benchmarking.

## Key constraints discovered

- **Both libraries claim the `:attr` shorthand.** hx-live disables its `:`
  short form and logs a warning when `window.Alpine` exists at init time.
  Sharing one document would force the hx-live side into the `hx-live:text`
  long form, misrepresenting its idiomatic syntax. Therefore every demo runs
  in its own iframe, and each iframe loads exactly one library.
- **hx-live has no loops, no `x-model`, no reactive store.** The DOM is the
  state. Ports keep the same DOM shape but move state into `data-*`, ARIA
  attributes, or input values. Where a port changes the model rather than the
  syntax (the search example), the notes say so explicitly.
- **Versions pinned:** `htmx.org@4.0.0` (stable, published 2026-08-28; ships
  `dist/ext/hx-live.js`) and `alpinejs@3.17.1` (matches the local checkout at
  `../alpine`).

## Architecture

Approach chosen: gsx shell pages + raw HTML example fragments on disk +
one iframe route per (library, example). Rejected: examples as gsx components
(rendered output drifts from authored source) and a static-site generator
(gives up the gsx dev loop; can be added later since all routes are stateless
GETs).

### Layout

```
go.mod                 module github.com/jackielii/hx-live-vs-alpine
                       replace github.com/gsxhq/gsx => ../gsxhq/gsx
                       tool github.com/gsxhq/gsx/cmd/gsx
main.go                net/http mux + github.com/gsxhq/vite middleware, embeds dist/
examples.go            manifest type, loader over embed.FS, list of six examples
examples/<slug>/
  alpine.html          raw fragment: exactly what a reader would write
  hxlive.html          raw fragment: the like-for-like hx-live port
  notes.md             short commentary: what differs and why (blog seed)
pages/index.gsx        outer comparison page built from gsxui components
pages/frame.gsx        iframe shell: one Vite frame entry + one fragment
ui/                    gsxui-vendored components: card, badge, tabs, separator,
                       sidebar, scroll-area (plus whatever they depend on)
web/main.js            outer bundle: gsxui runtime, index.css, app styles
web/frame-alpine.js    import "alpinejs" (window.Alpine + Alpine.start), frame.css, frame-resize.js
web/frame-hxlive.js    import "./htmx-setup.js" (window.htmx = htmx) THEN
                       "htmx.org/dist/ext/hx-live.js", frame.css, frame-resize.js
web/frame.css          small classless base styles so raw demos look decent
web/frame-resize.js    ResizeObserver on document.body -> postMessage({height}) to parent
vite.config.ts         spike config with three Rollup inputs
e2e/                   Playwright parity specs (one per example)
```

Scaffolded with `gsx init` then `gsxui init` and `gsxui add ...`, mirroring
`../gsxhq/hx-live-spike`. No CDN at runtime: both libraries come from npm and
are served by Vite as separate entries. The outer bundle never imports Alpine
or hx-live.

Slugs: `counter`, `dropdown`, `search`, `tabs`, `transition`, `classbind`.

### Manifest

```go
type example struct {
    slug, title string
    alpine, hxlive string // raw fragment bytes
    notes string          // markdown, rendered on the outer page
}
```

`examples.go` embeds `examples/` and builds the ordered slice at startup.
A missing or empty fragment file is a startup error (log.Fatal), not a
runtime 404.

### Routes

| Route | Renders |
|---|---|
| `GET /` | gsxui layout: sidebar with one link per example; main column with one card per example. Card header: title and notes. Card body: two-column grid. Each column: library badge, the iframe, and the fragment escaped inside `<pre><code>`. |
| `GET /frame/{lib}/{slug}` | Minimal document: `<html><head>` with the Vite assets for the matching frame entry, `<body>` containing the fragment injected unescaped. `lib` is `alpine` or `hxlive`. Unknown lib or slug returns 404. |
| `/static/`, `/__vite/` | handled by the vite package exactly as in the spike. |

The outer page listens for `message` events from its iframes and sets each
iframe's height from the reported value, so demos that grow (dropdown opens,
list filters) never clip. Fragments contain no links, so no `<base>` handling
is needed in frames.

Dev loop: `gsx dev` (Vite on :5173 proxying to Go on :7777). Prod:
`vite build && go tool gsx generate && go build`.

## The six ports

Alpine sources are verbatim from Alpine's docs where a docs example exists
(`../alpine/packages/docs/src/en/start-here.md` and `directives/*.md`).

### 1. counter

```html
<!-- alpine -->
<div x-data="{ count: 0 }">
    <button x-on:click="count++">Increment</button>
    <span x-text="count"></span>
</div>

<!-- hxlive -->
<div data-count="0">
    <button hx-on:click="data.count++">Increment</button>
    <span :text="data.count"></span>
</div>
```

Notes: `data.count` is a JSON round-trip on the closest `[data-count]`
ancestor; the number stays a number.

### 2. dropdown (show/hide + click outside)

```html
<!-- alpine -->
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle</button>
    <div x-show="open" @click.outside="open = false">Contents...</div>
</div>

<!-- hxlive -->
<div>
    <button aria-expanded="false" hx-on:click="toggle('aria-expanded')">Toggle</button>
    <div :hidden="!q('previous button').attr('aria-expanded')"
         hx-on="click from:outside -> q('previous button').attr('aria-expanded', false)">Contents...</div>
</div>
```

Notes: state lives in `aria-expanded`, which is also the accessibility
contract. `attr('aria-expanded')` returns a boolean. Both libraries attach the
outside listener on `document`, so the button's own click handler runs before
the outside handler in both; toggling open then closing does not fight.

### 3. search (x-model + getter + x-for)

```html
<!-- alpine -->
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

<!-- hxlive -->
<div>
    <input placeholder="Search...">
    <ul hx-live="for (let li of q('li in this')) li.hidden = !li.textContent.startsWith(q('previous input').value)">
        <li>foo</li>
        <li>bar</li>
        <li>baz</li>
    </ul>
</div>
```

Notes: hx-live has no loop primitive and no two-way binding. The list is
already HTML, so filtering is hiding rows; the input's own `.value` is the
state. This is a model change, not a syntax change: Alpine derives DOM from
data, hx-live derives attributes from DOM. Larger or server-owned lists would
be an htmx request, not client filtering.

### 4. tabs

```html
<!-- alpine -->
<div x-data="{ tab: 'a' }">
    <nav>
        <button @click="tab = 'a'" :class="{ active: tab === 'a' }">A</button>
        <button @click="tab = 'b'" :class="{ active: tab === 'b' }">B</button>
    </nav>
    <div x-show="tab === 'a'">Panel A</div>
    <div x-show="tab === 'b'">Panel B</div>
</div>

<!-- hxlive -->
<div data-tab="a">
    <nav>
        <button hx-on:click="data.tab = 'a'" :.active="data.tab === 'a'">A</button>
        <button hx-on:click="data.tab = 'b'" :.active="data.tab === 'b'">B</button>
    </nav>
    <div :hidden="data.tab !== 'a'">Panel A</div>
    <div :hidden="data.tab !== 'b'">Panel B</div>
</div>
```

Notes: closest like-for-like. The idiomatic hx-live alternative is
`role="tab"` + `take('aria-selected', '[role=tab]')`; mentioned in notes, not
shown as the primary port.

### 5. transition

```html
<!-- alpine -->
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle</button>
    <div x-show="open" x-transition>Hello 👋</div>
</div>

<!-- hxlive -->
<div data-open="false">
    <button hx-on:click="data.open = !data.open">Toggle</button>
    <div class="fade" :.open="data.open">Hello 👋</div>
</div>
<style>
    .fade { opacity: 0; visibility: hidden; transition: opacity .3s, visibility .3s; }
    .fade.open { opacity: 1; visibility: visible; }
</style>
```

Notes: hx-live has no transition directive; enter/leave is delegated to CSS.
The `<style>` is part of the shown source on purpose.

### 6. classbind (toggle button)

```html
<!-- alpine -->
<div x-data="{ pressed: false }">
    <button @click="pressed = ! pressed" :class="{ on: pressed }">Bold</button>
</div>

<!-- hxlive -->
<div>
    <button aria-pressed="false"
            hx-on:click="toggle('aria-pressed')"
            :class="{ on: attr('aria-pressed') }">Bold</button>
</div>
<style> .on { font-weight: bold; } </style>
```

Both fragments share the same `.on` style; the Alpine fragment carries the
same `<style>` block.

If browser verification shows a port behaving differently from its Alpine
twin, the port is adjusted and the reason recorded in that example's
`notes.md`.

## Verification

- **Go tests** (package main, `httptest`): manifest loads six examples with
  both fragments non-empty; `GET /` is 200 and contains every slug anchor and
  both escaped fragments; `GET /frame/{lib}/{slug}` is 200, contains the raw
  fragment, and references only that library's entry; unknown lib or slug is
  404. Tests run in the vite package's prod mode over a stub `dist` so Vite
  is not needed.
- **Playwright parity specs** in `e2e/`, one file per example, each spec
  parameterised over `alpine` and `hxlive` frame URLs with identical
  assertions (counter reaches 1 after one click; dropdown opens on toggle and
  closes on outside click; typing `ba` leaves exactly two visible items;
  tab B click shows Panel B and hides Panel A; transition target becomes
  visible; toggle button gains `.on`). Runs against the prod build on a fixed
  port, as in the spike.
- **Manual pass** in Chrome for styling and iframe sizing.

## Error handling

Missing fragment: startup failure. Unknown route params: 404. Render error:
500 with message. No server state, no forms, no persistence.
