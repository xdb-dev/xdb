import { readdirSync, readFileSync } from "node:fs";

// Starlight can generate a group only from src/content/docs. The guides
// are there, in an order that the list below sets. The concept pages are
// in /docs/concepts, so this file builds their group.
//
// The concept index, docs/concepts/README.md, has one H2 per topic and a
// list of links under it. The Concepts group follows the index: one
// subgroup per H2, with the pages in the order of its list. A page that
// the index does not list goes to the "More" subgroup, so each page gets
// an entry.

const GUIDES = [
  "define-a-schema",
  "import-types",
  "read-and-write",
  "choose-a-backend",
  "use-with-agents",
  "embed-in-go",
];

const CONCEPTS = new URL("../../docs/concepts/", import.meta.url);

function guides() {
  const onDisk = readdirSync(new URL("content/docs/guides/", import.meta.url))
    .filter((f) => /\.mdx?$/.test(f))
    .map((f) => f.replace(/\.mdx?$/, ""))
    .sort();
  const rank = (name) => GUIDES.indexOf(name) + 1 || Infinity;
  return onDisk
    .sort((a, b) => rank(a) - rank(b) || a.localeCompare(b))
    .map((name) => ({ slug: `docs/guides/${name}` }));
}

function conceptGroups() {
  const onDisk = readdirSync(CONCEPTS)
    .filter((f) => /\.mdx?$/.test(f) && !/^README\./.test(f))
    .map((f) => f.replace(/\.mdx?$/, ""));

  const groups = [];
  const text = readFileSync(new URL("README.md", CONCEPTS), "utf8");
  for (const line of text.split("\n")) {
    const heading = line.match(/^## (.+?)\s*$/);
    if (heading) {
      groups.push({ label: heading[1], pages: [] });
      continue;
    }
    const group = groups.at(-1);
    if (!group) continue;
    for (const [, page] of line.matchAll(/\]\(([\w-]+)\.mdx?\)/g)) {
      if (onDisk.includes(page) && !group.pages.includes(page)) {
        group.pages.push(page);
      }
    }
  }

  const listed = groups.flatMap((g) => g.pages);
  const more = onDisk.filter((page) => !listed.includes(page)).sort();
  if (more.length > 0) groups.push({ label: "More", pages: more });

  return groups
    .filter((g) => g.pages.length > 0)
    .map((g) => ({
      label: g.label,
      collapsed: true,
      items: g.pages.map((page) => ({ slug: `docs/concepts/${page}` })),
    }));
}

export const sidebar = [
  { label: "Overview", slug: "docs" },
  { label: "Get started", slug: "docs/start" },
  { label: "Guides", items: guides() },
  {
    label: "Concepts",
    collapsed: true,
    items: [{ label: "All concepts", slug: "docs/concepts" }, ...conceptGroups()],
  },
  {
    label: "CLI reference",
    link: "https://github.com/xdb-dev/xdb/blob/main/cmd/xdb/cli/CONTEXT.md",
    attrs: { target: "_blank" },
  },
  {
    label: "Go API",
    link: "https://pkg.go.dev/github.com/xdb-dev/xdb",
    attrs: { target: "_blank" },
  },
];
