import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import { rewriteDocLinks } from "./rewrite-doc-links.mjs";

const REPO = fileURLToPath(new URL("../../../", import.meta.url));
const transform = rewriteDocLinks({
  base: "/",
  repo: "https://github.com/xdb-dev/xdb",
});

function rewrite(from, url) {
  const tree = { type: "root", children: [{ type: "link", url, children: [] }] };
  transform(tree, { path: `${REPO}${from}` });
  return tree.children[0].url;
}

describe("rewriteDocLinks", () => {
  it("routes a page link from a page", () => {
    expect(rewrite("site/src/content/docs/guides/read-and-write.mdx", "../start.md#install")).toBe(
      "/docs/start/#install",
    );
  });

  it("routes a concept link from a page", () => {
    expect(
      rewrite(
        "site/src/content/docs/guides/read-and-write.mdx",
        "../../../../../docs/concepts/versioning.md",
      ),
    ).toBe("/docs/concepts/versioning/");
  });

  it("routes a concept link and the concept index from a concept page", () => {
    const from = "docs/concepts/records.md";
    expect(rewrite(from, "tuples.md")).toBe("/docs/concepts/tuples/");
    expect(rewrite(from, "README.md")).toBe("/docs/concepts/");
  });

  it("links a file that the site does not show to GitHub", () => {
    const from = "docs/concepts/records.md";
    expect(rewrite(from, "../README.md")).toBe(
      "https://github.com/xdb-dev/xdb/blob/main/docs/README.md",
    );
    expect(rewrite(from, "../../core/record.go")).toBe(
      "https://github.com/xdb-dev/xdb/blob/main/core/record.go",
    );
  });
});
