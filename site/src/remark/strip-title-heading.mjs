import { relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Each concept page in /docs/concepts starts with an H1 that repeats its
 * frontmatter title. GitHub needs that H1. Starlight shows the title, so
 * this plugin removes the first H1 of a concept page and no other heading.
 */

const CONCEPTS_DIR = fileURLToPath(new URL("../../../docs/concepts/", import.meta.url));

export function stripTitleHeading() {
  return (tree, file) => {
    const rel = file.path ? relative(CONCEPTS_DIR, file.path) : "..";
    if (rel.startsWith("..") || rel.startsWith(sep)) return;

    const first = tree.children.findIndex(
      (node) => node.type !== "yaml" && node.type !== "toml",
    );
    if (first === -1) return;
    const node = tree.children[first];
    if (node.type === "heading" && node.depth === 1) {
      tree.children.splice(first, 1);
    }
  };
}
