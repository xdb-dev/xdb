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
  The `.sketch` rule holds the hero underline: a rough.js stroke baked into a
  data URI, so the headline needs no JS. See below.
- `site.js` — the hand-drawn figures. Each `figure(...)` call renders one SVG with [rough.js](https://roughjs.com) using fixed seeds, so the sketches are identical on every load.
- `vendor/rough.js` — rough.js 4.6.6 (MIT).
- `og.html` / `og.png` — the social card. See below.
- `fonts/Excalifont-Regular.woff2` — Excalidraw's hand-drawn font, Latin subset (SIL Open Font License 1.1). Used only inside figures and captions.

Inter and JetBrains Mono load from Google Fonts and fall back to system fonts when offline.

The design exploration that led here is in `docs/landing/mock.html`. The
information architecture and the reasoning behind it are in
`docs/plans/2026-09-05-landing-page-ia.md`.

## The hero underline

`.sketch` in `site.css` underlines "Store anywhere." with the same pen as the
figures. It is one rough.js line drawn with the base from `site.js` — roughness
1.1, strokeWidth 1.4, bowing 1.2, seed 25 — with the resulting two paths baked
into a data URI. rough.js strokes every line twice, and that doubling is what
reads as a pen rather than a rule.

To draw a different one, load `vendor/rough.js` in a page and dump the paths:

```js
const rc = rough.svg(document.querySelector("svg")); // viewBox 0 0 300 14
const g = rc.line(2, 7, 298, 7,
  { roughness: 1.1, strokeWidth: 1.4, bowing: 1.2, seed: 25, stroke: "#2b4ee6" });
[...g.querySelectorAll("path")].map((p) => p.getAttribute("d"));
```

Try a few seeds and keep one whose two strokes stay inside the viewBox. Then
substitute the paths into the data URI, encoding `<`, `>` and `#` as `%3C`,
`%3E` and `%23`. The SVG is `preserveAspectRatio="none"`, so it stretches to the
width of the headline at every breakpoint.

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
