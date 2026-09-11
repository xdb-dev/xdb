// A link in /docs must work on GitHub and on the site. A relative path to
// a file in the repository works in both places, because
// site/src/remark/rewrite-doc-links.mjs rewrites it for the site.
//
// This script finds the links that the plugin cannot correct: links that
// start with "/", and relative links to files that do not exist.
// It does not read docs/plans or docs/research. The site does not show them.
//
//   node site/scripts/check-links.mjs [docsDir]

import { readdirSync, readFileSync, existsSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";

const docsDir = resolve(process.argv[2] ?? "docs");
const SKIP = new Set(["plans", "research"]);
const EXTERNAL = /^(?:[a-z][a-z0-9+.-]*:|\/\/|#)/i;

function walk(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = join(dir, e.name);
    if (e.isDirectory()) return SKIP.has(e.name) ? [] : walk(p);
    return /\.mdx?$/.test(e.name) ? [p] : [];
  });
}

// Go code looks like a Markdown link, for example fs.New[T]("./x").
// Remove the code before the scan.
function stripCode(text) {
  return text
    .replace(/^```[\s\S]*?^```/gm, "")
    .replace(/`[^`\n]*`/g, "");
}

const problems = [];

for (const file of walk(docsDir)) {
  const body = stripCode(readFileSync(file, "utf8"));
  const rel = relative(process.cwd(), file);

  for (const [, url] of body.matchAll(/\]\(([^)\s]+)\)/g)) {
    if (EXTERNAL.test(url)) continue;

    if (url.startsWith("/")) {
      problems.push(`${rel}: site-absolute link ${url}`);
      continue;
    }

    const target = resolve(dirname(file), url.split("#")[0].split("?")[0]);
    if (!existsSync(target)) {
      problems.push(`${rel}: dead link ${url}`);
    }
  }
}

if (problems.length > 0) {
  console.error(problems.join("\n"));
  console.error(
    `\n${problems.length} problem(s). A link must be a relative path to a ` +
      `file that exists, for example ../concepts/tuples.md`,
  );
  process.exit(1);
}

console.log("docs links ok");
