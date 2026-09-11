# XDB Site

The site is the landing page at `/` and the docs at `/docs/`. Astro and Starlight build both into `site/dist`.

```bash
make site-install   # pnpm install
make site-dev       # dev server at http://localhost:4321
make site-build     # static build to site/dist
make site-check     # astro check
make docs-links     # dead and site-absolute links in docs/
```

## Landing Page

The landing page is static HTML, CSS, and JavaScript in `public/`. Astro copies `public/` into the build without changes.

- `public/index.html` contains the pitch. Each section has a lede, a figure, and a link to its guide in `docs/howto/`:

  | Section | Contents | Guide |
  | ------- | -------- | ----- |
  | Hero | Product description, install command, one tuple in Go and the CLI, and the tuple anatomy (fig. 0) | |
  | `#how` | Formats convert to tuples, and tuples go to any backend (fig. 1) | |
  | `#model` | Records, resource URIs, and the resource hierarchy (fig. 2) | `define-a-schema` |
  | `#import` | A tagged Go struct and the import commands (fig. 3) | `import-types` |
  | `#backends` | Storage layouts (fig. 4) | `choose-a-backend` |
  | `#ops` | Go, JSON-RPC, and the CLI on one store (fig. 5) | `read-and-write`, `embed-in-go` |
  | `#agents` | The CLI grammar (fig. 6) | `use-with-agents` |
  | `#start` | Installation and first record | `get-started` |

  Put feature documentation in the guides, not on the landing page.
- `public/tokens.css`: the colours and fonts. The docs load the same file.
- `public/site.css`: styles. Cream background, one blue accent, dashed section rules.
  The `.sketch` rule holds the hero underline: a rough.js stroke baked into a
  data URI, so the headline needs no JS. See below.

- `public/site.js`: the hand-drawn figures. Each `figure(...)` call renders one SVG with [rough.js](https://roughjs.com) using fixed seeds, so the sketches are identical on every load.

- `public/vendor/rough.js`: rough.js 4.6.6 (MIT).

- `public/og.html` / `public/og.png`: the social card. See below.

- `public/fonts/Excalifont-Regular.woff2`: Excalidraw's hand-drawn font, Latin subset (SIL Open Font License 1.1). Used only inside figures and captions.

Inter and JetBrains Mono load from Google Fonts and fall back to system fonts when offline.

## Docs

`src/content.config.ts` loads the pages from the `docs/` directory of the repository, so GitHub and the site show the same files. The site does not show `docs/plans/` or `docs/research/`.

- `src/sidebar.mjs` builds the sidebar from `docs/howto/` and `docs/concepts/`. A new file gets an entry at the end of its group. To put a concept page in a named group, add its name to `CONCEPT_GROUPS`.
- `src/components/PageTitle.astro` shows the frontmatter under the title. `description` is the lede, `package` becomes links to pkg.go.dev, and `read_when` becomes a "Read this when" list.
- `src/remark/strip-title-heading.mjs` removes the first H1 of each page, because Starlight shows the title. The H1 stays in the file for GitHub.
- `src/remark/rewrite-doc-links.mjs` rewrites relative links. A link to a doc becomes a site route. A link to another file in the repository becomes a link to GitHub.
- `src/styles/starlight.css` maps the tokens onto the Starlight variables.

Write each link in `docs/` as a relative path to a file, for example `../concepts/tuples.md`. This link works on GitHub and on the site. `make docs-links` finds the links that do not work.

## The Hero Underline

`.sketch` in `public/site.css` underlines "Storage is a detail." with a rough.js line.
It uses the base settings from `site.js`: roughness 1.1, strokeWidth 1.4,
bowing 1.2, and seed 25. The resulting paths are stored in a data URI.
rough.js draws two strokes per line to give it a hand-drawn appearance.

To draw a different one, load `public/vendor/rough.js` in a page and dump the paths:

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

`public/og.png` (1200×630) is what Slack, iMessage and the rest render when someone
links the site. It is generated from `public/og.html`, which reuses `tokens.css`,
`site.css`, and the same rough.js sketching as the figures, to match the page design.
`og.html` is never linked from the site.

Regenerate it after editing `og.html`:

```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless --disable-gpu --hide-scrollbars \
  --force-device-scale-factor=2 --window-size=1200,630 \
  --virtual-time-budget=6000 \
  --screenshot=/tmp/og@2x.png "file://$PWD/site/public/og.html"
magick /tmp/og@2x.png -resize 1200x630 -strip site/public/og.png
```

It renders at 2× and downsamples so the text stays crisp. Fixed seeds keep the sketch geometry consistent. Font and browser changes can still affect the rendered PNG.

## Deploying

`.github/workflows/pages.yml` builds the site and publishes `site/dist` to
GitHub Pages at <https://xdb.dev>. It runs on every push to `main` that
changes `site/` or `docs/`, and can be run by hand from the Actions tab.
Keep every asset path in `public/` relative.
