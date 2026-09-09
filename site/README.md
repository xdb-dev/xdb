# XDB Landing Page

The site is static HTML, CSS, and JavaScript. Open `index.html` in a browser, or run this command from the repository root:

```bash
python3 -m http.server -d site 8000
```

## Layout

- `index.html` contains the page copy and examples:

  | Section | Contents |
  | ------- | -------- |
  | Hero | Product description and install command |
  | `#model` | Tuples, resource URIs, value types, and schema rules |
  | `#import` | Go structs, protobuf, JSON Schema, and drift checks |
  | `#backends` | Storage layouts, type mappings, config, and driver interfaces |
  | `#ops` | Reads, writes, version checks, dry runs, bulk data, and watch |
  | `#agents` | CLI reference, errors, pipes, aliases, skills, and discovery |
  | `#go` | Embedded stores and JSON-RPC handlers |
  | `#start` | Installation and first record |

  Add feature documentation to the relevant section and avoid repeating it elsewhere.
- `site.css`: styles. Cream background, one blue accent, dashed section rules.
  The `.sketch` rule holds the hero underline: a rough.js stroke baked into a
  data URI, so the headline needs no JS. See below.

- `site.js`: the hand-drawn figures. Each `figure(...)` call renders one SVG with [rough.js](https://roughjs.com) using fixed seeds, so the sketches are identical on every load.

- `vendor/rough.js`: rough.js 4.6.6 (MIT).

- `og.html` / `og.png`: the social card. See below.

- `fonts/Excalifont-Regular.woff2`: Excalidraw's hand-drawn font, Latin subset (SIL Open Font License 1.1). Used only inside figures and captions.

Inter and JetBrains Mono load from Google Fonts and fall back to system fonts when offline.

## The Hero Underline

`.sketch` in `site.css` underlines "Your backend." with a rough.js line.
It uses the base settings from `site.js`: roughness 1.1, strokeWidth 1.4,
bowing 1.2, and seed 25. The resulting paths are stored in a data URI.
rough.js draws two strokes per line to give it a hand-drawn appearance.

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

## Social Card

`og.png` (1200×630) is what Slack, iMessage and the rest render when someone
links the site. It is generated from `og.html`, which reuses `site.css` and the
same rough.js sketching as the figures, to match the page design.
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

It renders at 2× and downsamples so the text stays crisp. Fixed seeds keep the sketch geometry consistent. Font and browser changes can still affect the rendered PNG.

## Deploying

`.github/workflows/pages.yml` publishes this directory to GitHub Pages at
<https://xdb.dev>. It runs on every push to `main` that touches
`site/`, and can be run by hand from the Actions tab. There is no build step:
the workflow uploads `site/` as-is, so keep every asset path relative.
