# XDB Site

The site is the landing page at `/` and the docs at `/docs/`. Astro and Starlight build both into `site/dist`.

```bash
make site-install   # pnpm install
make site-dev       # dev server at http://localhost:4321
make site-build     # static build to site/dist
make site-check     # astro check
make site-test      # vitest
make docs-links     # dead and site-absolute links in docs/ and the pages
```

## Landing Page

The landing page is `src/pages/index.astro`. Its styles and scripts are in `public/`, and Astro copies `public/` into the build without changes.

- `src/pages/index.astro` contains the pitch. Each section supports one part of the tagline and links to the guides in `src/content/docs/guides/`:

  | Section | Contents | Guides |
  | ------- | -------- | ------ |
  | Hero | Product description, install command, one tuple in Go and the CLI, and the tuple anatomy (fig. 0) | |
  | `#demo` | A recorded session of the CLI, replayed in a terminal. See below. | |
  | `#how` | Formats convert to tuples, and tuples go to any backend (fig. 1) | `import-types`, `define-a-schema`, `choose-a-backend` |
  | `#agents` | The CLI grammar (fig. 2) | `use-with-agents` |
  | `#start` | Installation and first record | `start` |
  | `#guides` | A card for each guide | all |

  Put feature documentation in the guides, not on the landing page. The figures are the components of the docs. See [Figures](#figures).
- `public/tokens.css`: the colours and fonts. The docs load the same file.
- `public/site.css`: styles. Cream background, one blue accent, dashed section rules. The `.underlined` rule draws the accent line under "Storage is a detail."
- `public/og.html` / `public/og.png`: the social card. See below.
- `public/fonts/Excalifont-Regular.woff2`: Excalidraw's hand-drawn font, Latin subset (SIL Open Font License 1.1). The tagline, the figure captions, and the replay labels use it.

Inter and JetBrains Mono load from Google Fonts and fall back to system fonts when offline.

## Docs

The docs have two sources:

- The pages for the users of XDB are in `src/content/docs/`: the overview (`index.md`), Get started (`start.md`), and one page per guide in `guides/`.
- The concept pages are in the `docs/concepts/` directory of the repository. GitHub and the site show the same files. The site does not show the rest of `docs/`.

`src/content.config.ts` loads both. A concept page keeps its path, so `docs/concepts/tuples.md` is the route `/docs/concepts/tuples/`. `astro.config.mjs` redirects the routes of the old `docs/howto/` pages to the guides.

- `src/sidebar.mjs` builds the sidebar: Overview, Get started, Guides, Concepts, and the reference links. The guides follow the `GUIDES` list. The Concepts group follows the index, `docs/concepts/README.md`. Each H2 of the index is a subgroup, with its pages in the order of its list.
- `src/components/PageTitle.astro` shows the frontmatter under the title. `description` is the lede, `package` becomes links to pkg.go.dev, and `read_when` becomes a "Read this when" list.
- `src/remark/strip-title-heading.mjs` removes the first H1 of a concept page, because Starlight shows the title. The H1 stays in the file for GitHub.
- `src/remark/rewrite-doc-links.mjs` rewrites relative links. A link to a page or a concept page becomes a site route. A link to another file in the repository becomes a link to GitHub.
- `src/styles/starlight.css` maps the tokens onto the Starlight variables.

Write each link as a relative path to a file, for example `../../../../../docs/concepts/tuples.md` from a guide. `make docs-links` finds the links that do not work.

### Figures

A page draws a figure with a component from `src/components/figures/`. The landing page uses the same components. Each component is a form: the page gives it the content as props, and the component sets the layout. The colours are the tokens, and the layout fits the frame, from the landing page to a phone. The header comment of each component says what it shows and when to use it.

| The figure answers | Forms |
| ------------------ | ----- |
| What is inside it? | `Anatomy`, `Nesting`, `Contract` |
| Where does it go? | `Flow`, `Pipe` |
| In what order? | `Sequence` |

```mdx
import Pipe from "../../../components/figures/Pipe.astro";

<Pipe
  alt="echo pipes JSON into xdb records create, which pipes JSON into jq."
  caption="json in, json out"
  steps={[{ line: ["echo '{}'"] }, { line: ["xdb records create <uri> -"], accent: true }]}
  joints={["stdin as -"]}
/>
```

- Give each figure an `alt` that says what it shows.
- A line of code is a list of runs: a string, or a `[text, hue]` pair. `src/components/figures/hues.ts` lists the hues.
- If two pages draw the same figure, put its parts in `src/figures/parts.ts`.
- If no form answers the question of a figure, add a form. Put the content in props and the layout in the component, so the next figure can use it.

A concept page draws a flow, a sequence, or a state as a Mermaid block. GitHub renders the block with Mermaid. `src/remark/render-mermaid.mjs` renders it at build time with [Pele](https://pele.run), and the SVG reads the `--pele-*` variables in `starlight.css`. Start each block with `accTitle` and `accDescr`. A Mermaid block in a page stops the build.

Pele sizes each box from the width of its label. The plugin measures each label in the font files of the site, from the `@fontsource` packages. If `tokens.css` changes a font, change the font files in `astro.config.mjs` too.

## Terminal Replay

The `#demo` section replays real sessions of the CLI. Each session comes from one of the agent task evals in `internal/evals/tasks` and uses the fixture data of that task. Each session is a chapter. A chapter shows a still frame under a play button. A click on the button starts the replay, and a click on the screen pauses it. `public/replay.js` types each command from `public/demo.json` and prints the recorded output. With `prefers-reduced-motion`, the play button shows the full session at once.

`make site-demo` builds the CLI and runs `scripts/record-demo.mjs`. For each session, the script converts the fixtures to NDJSON, runs the commands against a new daemon in a temporary HOME, and writes `demo.json`. Run it again after a change to the output of the CLI or to a fixture. If a command exits with an unexpected code, the script stops and writes nothing.

## Social Card

`public/og.png` (1200×630) is what Slack, iMessage and the rest render when someone
links the site. It is generated from `public/og.html`, which reuses `tokens.css` and
`site.css` to match the page design.
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

It renders at 2× and downsamples so the text stays crisp. Font and browser changes can affect the rendered PNG.

## Deploying

`.github/workflows/pages.yml` builds the site and publishes `site/dist` to
GitHub Pages at <https://xdb.dev>. It runs on every push to `main` that
changes `site/` or `docs/`, and can be run by hand from the Actions tab.
Keep every asset path in `public/` relative.
