import { dirname, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * A page and a concept page link to a file with a relative path. GitHub
 * shows the concept pages in /docs/concepts, and a relative path is the
 * only link that works there:
 *
 *     [Records](records.md)                    a concept page
 *     [errors.go](../../core/errors.go)        a source file
 *
 * This plugin rewrites these links for the site. A link to a page or a
 * concept page becomes a site route. A link to another file becomes a link
 * to that file on GitHub.
 */

const PAGES_DIR = fileURLToPath(new URL("../content/docs/", import.meta.url));
const CONCEPTS_DIR = fileURLToPath(new URL("../../../docs/concepts/", import.meta.url));
const REPO_DIR = fileURLToPath(new URL("../../../", import.meta.url));

const EXTERNAL = /^(?:[a-z][a-z0-9+.-]*:|\/\/|#)/i;
const DOC_EXT = /\.mdx?$/;

// The ID of the page at absPath, as src/content.config.ts makes it, or
// undefined if the site does not show the file. An index or a README is
// the index of its directory.
function idFor(absPath) {
  let rel;
  if (isInside(PAGES_DIR, absPath)) {
    rel = relative(PAGES_DIR, absPath);
  } else if (isInside(CONCEPTS_DIR, absPath)) {
    rel = `concepts/${relative(CONCEPTS_DIR, absPath)}`;
  } else {
    return undefined;
  }
  const path = rel
    .split(sep)
    .join("/")
    .replace(DOC_EXT, "")
    .replace(/(^|\/)(index|README)$/, "")
    .replace(/\/$/, "");
  return path ? `docs/${path}` : "docs";
}

function isInside(dir, absPath) {
  const rel = relative(dir, absPath);
  return rel !== "" && !rel.startsWith("..") && !rel.startsWith(sep);
}

function walk(node, visit) {
  if (!node || typeof node !== "object") return;
  if (Array.isArray(node)) {
    for (const child of node) walk(child, visit);
    return;
  }
  // A definition node holds the URL of a reference-style link.
  if (node.type === "link" || node.type === "definition") visit(node);
  if (Array.isArray(node.children)) walk(node.children, visit);
}

/**
 * @param {object} options
 * @param {string} options.base   The Astro `base`, for example "/".
 * @param {string} options.repo   The repository URL, with no trailing slash.
 * @param {string} [options.branch]
 */
export function rewriteDocLinks({ base, repo, branch = "main" }) {
  const prefix = base === "/" ? "" : base.replace(/\/$/, "");

  return (tree, file) => {
    const from = file?.path ?? file?.history?.[0];
    if (!from) return;
    const fromDir = dirname(from);

    walk(tree, (node) => {
      const url = node.url;
      if (typeof url !== "string" || url === "" || EXTERNAL.test(url)) return;

      const [pathPart, ...rest] = url.split(/(?=[#?])/);
      const suffix = rest.join("");
      if (pathPart === "") return;

      const target = resolve(fromDir, pathPart);

      const id = DOC_EXT.test(pathPart) ? idFor(target) : undefined;
      if (id) {
        node.url = `${prefix}/${id}/${suffix}`;
        return;
      }

      if (isInside(REPO_DIR, target)) {
        const rel = relative(REPO_DIR, target).split(sep).join("/");
        node.url = `${repo}/blob/${branch}/${rel}${suffix}`;
      }
    });
  };
}
