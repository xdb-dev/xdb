import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { openSync } from "fontkit";
import { describe, expect, it } from "vitest";

import { renderMermaid } from "./render-mermaid.mjs";

const require = createRequire(import.meta.url);
const REPO = fileURLToPath(new URL("../../../", import.meta.url));

const FONTS = {
  regular: "@fontsource/inter/files/inter-latin-400-normal.woff",
  bold: "@fontsource/inter/files/inter-latin-700-normal.woff",
  mono: "@fontsource/jetbrains-mono/files/jetbrains-mono-latin-400-normal.woff",
  monoBold: "@fontsource/jetbrains-mono/files/jetbrains-mono-latin-700-normal.woff",
};

function treeWith(value) {
  return { type: "root", children: [{ type: "code", lang: "mermaid", value }] };
}

describe("renderMermaid", () => {
  const transform = renderMermaid({ fonts: FONTS });

  it("renders a Mermaid block of a concept page to an SVG", () => {
    const tree = treeWith("flowchart LR\n  A --> B");
    transform(tree, { path: `${REPO}docs/concepts/daemon.md` });
    const [node] = tree.children;
    expect(node.type).toBe("html");
    expect(node.value).toMatch(/^<figure class="frame pele-fig">/);
    expect(node.value).toContain('class="pele pele-flowchart"');
    expect(node.value).toContain("var(--pele-fg");
  });

  it("stops the build on a Mermaid block in a page", () => {
    const tree = treeWith("flowchart LR\n  A --> B");
    expect(() =>
      transform(tree, { path: `${REPO}site/src/content/docs/guides/read-and-write.mdx` }),
    ).toThrow(/belongs in a concept page/);
  });

  it("keeps a label inside its box in the site font", () => {
    const tree = treeWith("flowchart LR\n  A[EventApprovalStart]");
    transform(tree, { path: `${REPO}docs/concepts/daemon.md` });
    const box = tree.children[0].value.match(
      /<g class="pele-node[^"]*" data-id="A"[^>]*><rect [^>]*width="([\d.]+)"/,
    );
    expect(box).not.toBeNull();
    // The width of the label in Inter at 16px.
    const font = openSync(require.resolve(FONTS.regular));
    const label = font.layout("EventApprovalStart").advanceWidth / font.unitsPerEm;
    expect(Number(box[1])).toBeGreaterThan(label * 16);
  });
});
