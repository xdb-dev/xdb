/**
 * Each doc in /docs starts with an H1 that repeats its frontmatter title.
 * GitHub needs that H1. Starlight shows the title itself, so this plugin
 * removes the first H1 and no other.
 */
export function stripTitleHeading() {
  return (tree) => {
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
