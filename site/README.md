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
- `og.html` / `og.png` — the social card. See below.
- `fonts/Excalifont-Regular.woff2` — Excalidraw's hand-drawn font, Latin subset (SIL Open Font License 1.1). Used only inside figures and captions.

Inter and JetBrains Mono load from Google Fonts and fall back to system fonts when offline.

The design exploration that led here is in `docs/landing/mock.html`. The
information architecture and the reasoning behind it are in
`docs/plans/2026-09-05-landing-page-ia.md`.

## Social card

`og.png` (1200×630) is what Slack, iMessage and the rest render when someone
links the site. It is generated from `og.html`, which reuses `site.css` and the
same rough.js sketching as the figures, so the card cannot drift from the page.
`og.html` is never linked from the site.

Regenerate it after editing `og.html`:

```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless --disable-gpu --hide-scrollbars \
  --force-device-scale-factor=2 --window-size=1200,630 \
  --virtual-time-budget=6000 \
  --screenshot=/tmp/og@2x.png "file://$PWD/site/og.html"
magick /tmp/og@2x.png -resize 1200x630 -strip site/og.png
```

It renders at 2× and downsamples so the text stays crisp. The sketch uses fixed
seeds, so an unchanged card regenerates to a byte-identical PNG.

## Deploying

`.github/workflows/pages.yml` publishes this directory to GitHub Pages at
<https://xdb.dev>. It runs on every push to `main` that touches
`site/`, and can be run by hand from the Actions tab. There is no build step:
the workflow uploads `site/` as-is, so keep every asset path relative.
