import { readdirSync } from "node:fs";

// Starlight can generate a group only from src/content/docs. These pages
// come from /docs, so this file builds the groups from that directory.
//
// A group lists its pages in the order of `order`. The sidebar adds the
// pages that are not in a list to the end of their group, in alphabetical
// order. Thus a new file always gets a sidebar entry.
function slugsIn(dir) {
  let files;
  try {
    files = readdirSync(new URL(`../../docs/${dir}/`, import.meta.url));
  } catch {
    // A missing directory gives an empty group. It does not stop the build.
    return [];
  }
  return files
    .filter((f) => /\.mdx?$/.test(f) && !/^README\.mdx?$/.test(f))
    .map((f) => f.replace(/\.mdx?$/, ""))
    .sort((a, b) => a.localeCompare(b));
}

function ordered(names, order) {
  const rank = (name) => order.indexOf(name) + 1 || Infinity;
  return [...names].sort((a, b) => rank(a) - rank(b) || a.localeCompare(b));
}

// The concept pages are in one directory. These groups divide them in the
// same way as docs/concepts/README.md.
const CONCEPT_GROUPS = [
  {
    label: "Data model",
    pages: ["tuples", "records", "uris", "types", "schemas", "namespaces", "versioning"],
  },
  {
    label: "Storage",
    pages: ["stores", "drivers", "filters"],
  },
  {
    label: "Formats",
    pages: ["bring-your-own-types", "encoding"],
  },
  {
    label: "CLI and daemon",
    pages: ["config", "daemon", "errors"],
  },
];

function conceptGroups() {
  const all = slugsIn("concepts");
  const listed = CONCEPT_GROUPS.flatMap((g) => g.pages);
  const unlisted = all.filter((name) => !listed.includes(name));

  const groups = CONCEPT_GROUPS.map((g) => ({
    label: g.label,
    items: g.pages
      .filter((name) => all.includes(name))
      .map((name) => ({ slug: `docs/concepts/${name}` })),
  }));

  if (unlisted.length > 0) {
    groups.push({
      label: "More",
      items: unlisted.map((name) => ({ slug: `docs/concepts/${name}` })),
    });
  }
  return groups;
}

const HOWTO_ORDER = [
  "define-a-schema",
  "import-types",
  "read-and-write",
  "choose-a-backend",
  "use-with-agents",
  "embed-in-go",
];

export const sidebar = [
  {
    label: "Start here",
    items: [
      { slug: "docs", label: "Overview" },
      { slug: "docs/howto/get-started" },
    ],
  },
  {
    label: "How to",
    items: ordered(
      slugsIn("howto").filter((name) => name !== "get-started"),
      HOWTO_ORDER,
    ).map((name) => ({ slug: `docs/howto/${name}` })),
  },
  {
    label: "Concepts",
    items: [
      { slug: "docs/concepts", label: "All concepts" },
      ...conceptGroups(),
    ],
  },
  {
    label: "Reference",
    items: [
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
    ],
  },
];
