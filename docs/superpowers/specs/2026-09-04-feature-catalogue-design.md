# Feature catalogue: every Alpine core feature against hx-live

**Date:** 2026-09-04
**Extends:** `2026-09-04-hx-live-vs-alpine-design.md` (the six-example app, merged to `main` at 4a6a368)
**Status:** approved design, pre-implementation

## Goal

Grow the comparison from six scenario examples into a catalogue that covers
every Alpine.js core feature: 18 directives, 9 magics, 3 globals. Each
feature gets a card with the Alpine docs example and, where one exists, a
like-for-like hx-live port. Where none exists the card says so and explains
what an hx-live user does instead. A matrix at the top of the page lists every
feature with its status. Plugins (mask, intersect, persist, focus, collapse,
anchor, morph, sort, resize) are out of scope.

Non-goals: emulating features hx-live lacks (no teleport shim, no `x-for`
runtime), plugin coverage, a static export.

## Status vocabulary

| Status | Meaning |
|---|---|
| `equivalent` | a port with the same DOM shape and the same observable behaviour |
| `workaround` | same user-visible result through a different model (state in DOM, CSS, static HTML) |
| `none` | no client-side equivalent; the card's right column is prose |

## Manifest changes (`examples/examples.go`)

```go
type Group string   // "directive" | "magic" | "global"
type Status string  // "equivalent" | "workaround" | "none"

type Example struct {
    Slug      string
    Title     string
    Group     Group
    Features  []string // Alpine names this card demonstrates, e.g. "x-data", "$store"
    Status    Status
    Alpine    string   // raw fragment
    HxLive    string   // raw fragment; empty when Status == none
    NotesHTML string
}
```

`order` becomes a slice of `{slug, title, group, features, status}`. `Load`
reads `alpine.html` and `notes.md` for every row and `hxlive.html` only when
status is not `none`; it errors when `hxlive.html` exists for a `none` row or
is missing for any other row. A feature name may appear on exactly one row;
`Load` errors on duplicates.

New helper: `func Features(exs []Example) []FeatureRow` returning one row per
feature in Alpine docs order (see the matrix section), each `{Feature, Slug,
Title, Status}`. The Go test asserts every one of the 30 feature names below
is present exactly once.

## Page changes (`pages/index.gsx`)

- **Matrix** above the cards: a table with columns Feature, Status, Card. Feature is the Alpine name; Status is a `ui.Badge` (`equivalent` default variant, `workaround` secondary, `none` outline); Card is an anchor to `#<slug>`. Grouped by three subheadings: Directives, Magics, Globals.
- **Sidebar** groups links under the same three headings, in card order.
- **Card header** shows a status badge next to the title. For `equivalent` and `workaround` rows the notes render in the header as today.
- **Card body** for demo rows is unchanged (two columns, iframe + source). For `none` rows the left column is the Alpine demo and source; the right column carries a "No hx-live equivalent" badge and the notes prose (which is therefore not repeated in the header).
- Frame route: `GET /frame/hxlive/<slug>` returns 404 for `none` rows.

## Card order and content

Cards appear in this order. Alpine fragments are the docs' first example for
that feature, verbatim, with one systematic adaptation: where the docs use
`alert()` or `console.log()`, the side effect writes into a visible element
instead, and the notes say "adapted from the docs: alert replaced by text".
Every class a fragment relies on is styled by a `<style>` inside that
fragment. Existing cards keep their fragments.

### Directives

**counter** (exists) — features `x-data`, `x-text`, `$data` — equivalent.

**tabs** (exists) — feature `x-bind (class, tabs)` — equivalent. Scenario card kept from the first spec.

**dropdown** (exists) — features `x-show`, `x-on (.outside)` — equivalent.

**classbind** (exists) — feature `x-bind (class object)` — equivalent.

**bind** — feature `x-bind` — equivalent.
```html
<!-- alpine (docs verbatim) -->
<div x-data="{ placeholderText: 'Type here...' }">
    <input type="text" x-bind:placeholder="placeholderText">
</div>

<!-- hxlive -->
<div data-placeholder="Type here...">
    <input type="text" :placeholder="data.placeholder">
</div>
```
Notes: any attribute binds with `:attr`; the value comes from the closest `data-*`. If hx-live maps `data-placeholder-text` to `data.placeholderText` the way `dataset` does, the implementer may keep the docs' name; otherwise `placeholder` stays.

**on** — feature `x-on (modifiers)` — equivalent. Adapted from the docs' `@keyup.enter` example.
```html
<!-- alpine -->
<div x-data="{ message: '' }">
    <input type="text" @keyup.enter="message = 'Enter pressed'">
    <span x-text="message"></span>
</div>

<!-- hxlive -->
<div data-message="">
    <input type="text" hx-on="keyup[key=='Enter'] -> data.message = 'Enter pressed'">
    <span :text="data.message"></span>
</div>
```
Notes: Alpine's modifiers (`.enter`, `.prevent`, `.debounce`) are hx-on's event filters and modifiers (`keyup[key=='Enter']`, `.prevent` via `event.preventDefault()`, `debounce:`).

**init** — feature `x-init` — equivalent. Adapted (console.log to text).
```html
<!-- alpine -->
<div x-data="{ message: '' }" x-init="message = 'Initialised!'">
    <span x-text="message"></span>
</div>

<!-- hxlive -->
<div data-message="" hx-on:load="data.message = 'Initialised!'">
    <span :text="data.message"></span>
</div>
```

**html** — feature `x-html` — equivalent. Docs verbatim on the Alpine side.
```html
<!-- hxlive -->
<div data-username="<strong>calebporzio</strong>">
    Username: <span :html="data.username"></span>
</div>
```

**effect** — feature `x-effect` — equivalent. Adapted (console.log to a derived value).
```html
<!-- alpine -->
<div x-data="{ label: 'Hello', length: 0 }" x-effect="length = label.length">
    <button @click="label += ' World!'">Change Message</button>
    <span x-text="length"></span>
</div>

<!-- hxlive -->
<div data-label="Hello" data-length="0" hx-live="data.length = data.label.length">
    <button hx-on:click="data.label += ' World!'">Change Message</button>
    <span :text="data.length"></span>
</div>
```
Notes: `x-effect` tracks the properties it reads; `hx-live` re-runs on any DOM change. Same result, different trigger.

**ignore** — feature `x-ignore` — equivalent. Docs example plus one visible sibling so the effect is observable.
```html
<!-- alpine -->
<div x-data="{ label: 'processed' }">
    <span x-text="label"></span>
    <div x-ignore>
        <span x-text="label">untouched</span>
    </div>
</div>

<!-- hxlive -->
<div data-label="processed">
    <span :text="data.label"></span>
    <div hx-ignore>
        <span :text="data.label">untouched</span>
    </div>
</div>
```

**ref** — features `x-ref`, `$refs` — equivalent. Docs verbatim inside an `x-data` wrapper (the docs page provides one implicitly).
```html
<!-- alpine -->
<div x-data>
    <button @click="$refs.text.remove()">Remove Text</button>

    <span x-ref="text">Hello 👋</span>
</div>

<!-- hxlive -->
<div>
    <button hx-on:click="q('#text').remove()">Remove Text</button>

    <span id="text">Hello 👋</span>
</div>
```
Notes: hx-live has no ref registry; an id or a directional query (`q('next span')`) does the job.

**model** — feature `x-model` — workaround. Docs verbatim on the Alpine side.
```html
<!-- hxlive -->
<div>
    <input type="text">

    <span :text="q('previous input').value"></span>
</div>
```
Notes: the input is the state; there is nothing to bind two ways. Writing back is `q('input').value = ...` in a handler.

**for** — feature `x-for` — workaround. Docs example plus a push, so the list changes.
```html
<!-- alpine -->
<div x-data="{ colors: ['Red', 'Orange', 'Yellow'] }">
    <ul>
        <template x-for="color in colors">
            <li x-text="color"></li>
        </template>
    </ul>
    <button @click="colors.push('Green')">Add Green</button>
</div>

<!-- hxlive -->
<div>
    <ul>
        <li>Red</li>
        <li>Orange</li>
        <li>Yellow</li>
    </ul>
    <button hx-on:click="q('previous ul').insert('end', '<li>Green</li>')">Add Green</button>
</div>
```
Notes: the list is HTML, not data. Additions are `insert()`, or an htmx request when the server owns the list.

**search** (exists) — features `x-model (filtering)`, `x-for (filtering)` — workaround.

**if** — feature `x-if` — workaround.
```html
<!-- alpine (docs example inside its toggle) -->
<div x-data="{ open: false }">
    <button @click="open = ! open">Toggle</button>

    <template x-if="open">
        <div>Contents...</div>
    </template>
</div>

<!-- hxlive -->
<div data-open="false">
    <button hx-on:click="data.open = !data.open">Toggle</button>

    <div :hidden="!data.open">Contents...</div>
</div>
```
Notes: hx-live never removes elements; `:hidden` keeps them in the DOM. If removal matters (forms, focus order), it is a server swap.

**transition** (exists) — feature `x-transition` — workaround.

**cloak** — feature `x-cloak` — workaround. Docs verbatim inside an `x-data` wrapper, with the CSS the docs prescribe.
```html
<!-- alpine -->
<div x-data>
    <span x-cloak x-show="false">This will not 'blip' onto screen at any point</span>
</div>
<style>
    [x-cloak] { display: none !important; }
</style>

<!-- hxlive -->
<div>
    <span hx-cloak :hx-cloak="false" :hidden="true">This will not 'blip' onto screen at any point</span>
</div>
<style>
    [hx-cloak] { display: none !important; }
</style>
```
Notes: hx-live has no cloak directive; the element removes its own marker on first evaluation (`:hx-cloak="false"` removes the attribute), which happens in the same pass that applies `:hidden`.

**teleport** — feature `x-teleport` — none. Alpine: docs example without the `<body>` wrapper. Notes: no client-side teleport; use `<dialog>` or the Popover API (both render in the top layer regardless of DOM position), or place the element where it belongs server-side.

**modelable** — feature `x-modelable` — none. Alpine: docs verbatim. Notes: this is a component-API feature; hx-live has no components. Shared state lives on a common ancestor's `data-*`.

**id** — features `x-id`, `$id` — none. Alpine: docs verbatim (two labelled inputs). Notes: unique ids are the server's job; gsx generates them at render time.

### Magics

**el** — feature `$el` — equivalent.
```html
<!-- alpine -->
<div x-data>
    <button @click="$el.innerHTML = 'Hello World!'">Replace me with "Hello World!"</button>
</div>

<!-- hxlive -->
<div>
    <button hx-on:click="this.innerHTML = 'Hello World!'">Replace me with "Hello World!"</button>
</div>
```

**dispatch** — feature `$dispatch` — equivalent. Adapted (alert to text).
```html
<!-- alpine -->
<div x-data="{ message: '' }" @notify="message = 'Notified!'">
    <button @click="$dispatch('notify')">Notify</button>
    <span x-text="message"></span>
</div>

<!-- hxlive -->
<div data-message="" hx-on:notify="data.message = 'Notified!'">
    <button hx-on:click="trigger('notify')">Notify</button>
    <span :text="data.message"></span>
</div>
```
Notes: `trigger()` dispatches a bubbling CustomEvent (pass `true` as the third argument if the default does not bubble).

**root** — feature `$root` — equivalent. Adapted (alert to text).
```html
<!-- alpine -->
<div x-data data-message="Hello World!">
    <button @click="$el.textContent = $root.dataset.message">Say Hi</button>
</div>

<!-- hxlive -->
<div data-message="Hello World!">
    <button hx-on:click="this.textContent = data.message">Say Hi</button>
</div>
```
Notes: `data.*` already resolves to the closest ancestor, which is what `$root` gives Alpine.

**nexttick** — feature `$nextTick` — equivalent. Adapted (console.log to text).
```html
<!-- alpine -->
<div x-data="{ title: 'Hello', after: '' }">
    <button @click="title = 'Hello World!'; $nextTick(() => { after = $el.innerText })" x-text="title"></button>
    <span x-text="after"></span>
</div>

<!-- hxlive -->
<div data-title="Hello" data-after="">
    <button hx-on:click="data.title = 'Hello World!'; nextFrame().then(() => data.after = this.innerText)" :text="data.title"></button>
    <span :text="data.after"></span>
</div>
```
Expected in both: the span shows "Hello World!" after the click, proving the read happened after the re-render.

**store** — features `$store`, `Alpine.store` — equivalent. Adapted from the docs' darkMode example: two independent components share state.
```html
<!-- alpine -->
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

<!-- hxlive -->
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
Notes: a store is a `data-*` attribute on whatever ancestor both components share, up to `<body>`.

**watch** — feature `$watch` — workaround. Adapted (console.log to text).
```html
<!-- alpine -->
<div x-data="{ open: false, log: '' }" x-init="$watch('open', value => log = 'open is now ' + value)">
    <button @click="open = ! open">Toggle Open</button>
    <span x-text="log"></span>
</div>

<!-- hxlive -->
<div data-open="false" data-log="" hx-live="data.log = 'open is now ' + data.open">
    <button hx-on:click="data.open = !data.open">Toggle Open</button>
    <span :text="data.log"></span>
</div>
```
Notes: no per-key watcher; an `hx-live` expression is an effect that also runs once at load, so the hx-live side shows "open is now false" before any click while Alpine's span starts empty.

### Globals

**alpine-data** — feature `Alpine.data` — none. Alpine: the docs' `dropdown` example with the placeholders filled in ("Toggle", "Contents..."). Notes: reusable components are a server concern; in this app the reusable unit is a gsx component that emits the `data-*` and bindings.

**alpine-bind** — feature `Alpine.bind` — none. Alpine: the docs' `Alpine.bind('SomeButton', ...)` example verbatim. Notes: same as above; attribute bundles are a template concern.

## Matrix order

Directives: x-bind, x-bind (class object), x-bind (class, tabs), x-cloak, x-data, x-effect, x-for, x-for (filtering), x-html, x-id, x-if, x-ignore, x-init, x-model, x-model (filtering), x-modelable, x-on (modifiers), x-on (.outside), x-ref, x-show, x-teleport, x-text, x-transition.
Magics: $data, $dispatch, $el, $id, $nextTick, $refs, $root, $store, $watch.
Globals: Alpine.bind, Alpine.data, Alpine.store.

Thirty-five lines, thirty distinct Alpine features (the "(filtering)", "(class object)", "(class, tabs)", "(.outside)" and "(modifiers)" lines show a feature more than once and count once). The existing tabs scenario card stays, keyed as `x-bind (class, tabs)`.

## Verification

- **Go tests:** `Load` returns rows in the order above; every `none` row has empty `HxLive` and every other row a non-empty one; `Features()` lists every matrix line exactly once; `GET /` contains the matrix with an anchor per line and a card per slug; `GET /frame/hxlive/teleport` (and every `none` slug) is 404 while `GET /frame/alpine/teleport` is 200; `GET /frame/hxlive/<demo slug>` is 200.
- **Playwright smoke (`e2e/smoke.spec.ts`):** load `/`, collect every `iframe[data-frame]` src, visit each directly, assert no console errors or page errors and a non-empty body. The count of srcs equals twice the number of demo rows.
- **Playwright parity (`e2e/parity.spec.ts`)**, per library, for the new interactive rows: bind (placeholder attribute equals "Type here..."), on (type text, press Enter, span reads "Enter pressed"), init (span reads "Initialised!" without interaction), html (a `strong` element with text "calebporzio"), effect (span reads 5, click, reads 12), ignore (first span reads "processed", inner span reads "untouched"), ref (click removes "Hello"), if (toggle shows then hides "Contents..."), cloak (the span is hidden and no `[x-cloak]`/`[hx-cloak]` element remains), el (button text becomes "Hello World!"), dispatch (span reads "Notified!"), root (button text becomes "Hello World!"), nexttick (second span reads "Hello World!"), store (click, the Content div has class `dark`; click again, it does not), watch (click, span reads "open is now true"), model (type "hi", span reads "hi"), for (three items, click, four items with last "Green").
- **Index e2e** keeps the sizing test; the expected frame count comes from the collected srcs, not a constant.
- **Manual pass** in Chrome for the matrix, badges, and `none` cards.

## Error handling

Unchanged: bad manifest or missing file is a startup failure; unknown routes 404.
