# hx-live vs Alpine

Six examples from the Alpine.js docs, each ported like-for-like to htmx 4's
`hx-live` extension, rendered side by side with the source that runs them.
Raw material for a blog post.

Every demo runs in its own iframe that loads exactly one library, because
Alpine and hx-live both claim the `:attr` shorthand and hx-live disables its
short form when it detects Alpine. The hx-live ports target htmx 4.0.0's typed state bags (`data.*`, `aria.*`, `class.*`).

## Run

    npm install
    npm run dev          # gsx dev: Vite + Go with reload

## Test

    npm install && npm run build            # once, so dist/ exists for go:embed
    go tool gsx generate && go test ./...   # routes and manifest
    npm run e2e                             # Playwright parity, both libraries

## Layout

- `examples/<slug>/alpine.html`, `hxlive.html`, `notes.md`: the content.
- `pages/frame.gsx`: the single-library iframe document.
- `pages/index.gsx`: the comparison page.
- `web/frame-alpine.js`, `web/frame-hxlive.js`: one Vite entry per library.

Pinned: `htmx.org@4.0.0`, `alpinejs@3.17.1`.
