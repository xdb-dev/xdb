import { createRequire } from "node:module";
import { relative, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { openSync } from "fontkit";
import { render } from "pele";

/**
 * A concept page in /docs/concepts can draw a diagram as a fenced Mermaid
 * block:
 *
 *     ```mermaid
 *     sequenceDiagram
 *       accTitle: The start of the daemon
 *       CLI->>Daemon: re-exec
 *     ```
 *
 * GitHub draws the block with Mermaid. This plugin renders each block to an
 * SVG at build time with Pele, so the page ships no script. The SVG takes its
 * colors from the --pele-* variables in src/styles/starlight.css.
 *
 * A page in src/content/docs draws with src/components/figures/. A
 * Mermaid block in such a page stops the build.
 *
 * Pele sizes each box from the width of its label. Without a browser, Pele
 * guesses the width from a table for a generic sans-serif font. The site
 * fonts are wider, so a label can overflow its box. This plugin measures
 * each label in the font file that the page uses.
 *
 * Give each diagram an accTitle. Pele writes it as the title of the SVG.
 */

const REPO_DIR = fileURLToPath(new URL("../../../", import.meta.url));
const CONCEPTS_DIR = fileURLToPath(new URL("../../../docs/concepts/", import.meta.url));
const require = createRequire(import.meta.url);

// Pele passes the style of a label as bit flags.
const BOLD = 1;
const MONO = 4;

// fonts maps each style to a font file of a package, for example
// "@fontsource/inter/files/inter-latin-400-normal.woff".
function fontMeasurer(fonts) {
  const files = {
    0: openSync(require.resolve(fonts.regular)),
    [BOLD]: openSync(require.resolve(fonts.bold)),
    [MONO]: openSync(require.resolve(fonts.mono)),
    [MONO | BOLD]: openSync(require.resolve(fonts.monoBold)),
  };
  const cache = new Map();
  return {
    width(text, size, style) {
      // The font files have no italic. An italic label uses the upright width.
      const font = files[style & (BOLD | MONO)];
      const key = `${style & (BOLD | MONO)}:${text}`;
      let em = cache.get(key);
      if (em === undefined) {
        em = font.layout(text).advanceWidth / font.unitsPerEm;
        cache.set(key, em);
      }
      return em * size;
    },
  };
}

// Pele accepts an ID prefix of letters, digits, "-", and "_". Two diagrams
// on one page must not share a prefix.
function prefixFor(path, index) {
  const rel = relative(REPO_DIR, path).split(sep).join("-");
  return `pele-${rel.replace(/\.mdx?$/, "").replace(/[^A-Za-z0-9_-]/g, "-")}-${index}`;
}

function walk(node, visit) {
  if (!node || !Array.isArray(node.children)) return;
  node.children.forEach((child, index) => {
    visit(child, index, node);
    walk(child, visit);
  });
}

/**
 * @param {object} options
 * @param {{regular: string, bold: string, mono: string, monoBold: string}} options.fonts
 * @param {number} [options.maxWidth] the width of the frame content, in pixels.
 *   Pele wraps the labels and moves the parts to fit it. To fit, Pele can
 *   break a word in the middle, so leave it unset when the labels are
 *   identifiers. Then the SVG keeps its own width and scales down to fit.
 */
export function renderMermaid({ fonts, maxWidth }) {
  const measurer = fontMeasurer(fonts);
  return (tree, file) => {
    let count = 0;
    walk(tree, (node, index, parent) => {
      if (node.type !== "code" || node.lang !== "mermaid") return;
      const rel = file.path ? relative(CONCEPTS_DIR, file.path) : "..";
      if (rel.startsWith("..") || rel.startsWith(sep)) {
        throw new Error(
          `${file.path}: a Mermaid block belongs in a concept page. Draw a page with a figure component.`,
        );
      }
      // A syntax error stops the build, with the line and the column.
      const { svg } = render(node.value, {
        idPrefix: prefixFor(file.path, count++),
        measurer,
        maxWidth,
        // The column is narrower than Pele's breakpoint. Keep the direction
        // that the source gives.
        autoDirection: false,
      });
      parent.children[index] = {
        type: "html",
        value: `<figure class="frame pele-fig">${svg}</figure>`,
      };
    });
  };
}
