# XDB landing page

Static, no build step. Open `index.html` in a browser or serve the directory:

```bash
python3 -m http.server -d site 8000
```

## Layout

- `index.html` — the page. Each section answers one reader question, in order:

  | Section     | Question                        | Contents                                                                 |
  | ----------- | ------------------------------- | ------------------------------------------------------------------------ |
  | hero        | What is this?                   | the promise, the install line, fig. 0                                     |
  | `#model`    | What is the data?               | tuple anatomy, nesting, attribute URIs, value types, schema features      |
  | `#import`   | Do I have to re-model?          | Go structs, protobuf, JSON Schema, `schemas diff --check`                 |
  | `#backends` | Where does it live?             | storage forms, type mapping, config, the driver capability stack          |
  | `#ops`      | What can I do to it?            | three doors, then read, write, CAS, dry run, bulk, watch                  |
  | `#agents`   | Why is it easier for an agent?  | the grammar, describe, errors, composition, shorthand, skills, discovery  |
  | `#go`       | Can I embed it?                 | repository layer, JSON-RPC server, introspection                          |
  | `#start`    | How do I begin?                 | three commands                                                            |

  Every feature has exactly one home. When you add one, put it in the section
  whose question it answers rather than appending a card to the end.
- `site.css` — styles. Cream background, one blue accent, dashed section rules.
- `site.js` — the hand-drawn figures. Each `figure(...)` call renders one SVG with [rough.js](https://roughjs.com) using fixed seeds, so the sketches are identical on every load.
- `vendor/rough.js` — rough.js 4.6.6 (MIT).
- `fonts/Excalifont-Regular.woff2` — Excalidraw's hand-drawn font, Latin subset (SIL Open Font License 1.1). Used only inside figures and captions.

Inter and JetBrains Mono load from Google Fonts and fall back to system fonts when offline.

The design exploration that led here is in `docs/landing/mock.html`. The
information architecture and the reasoning behind it are in
`docs/plans/2026-09-05-landing-page-ia.md`.
