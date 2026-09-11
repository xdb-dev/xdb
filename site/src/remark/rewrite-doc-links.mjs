import { dirname, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * GitHub and this site both show the files in /docs. A relative path to a
 * file in the repository is the only link that works on GitHub:
 *
 *     [Records](records.md)                    a doc
 *     [errors.go](../../core/errors.go)        a source file
 *
 * This plugin rewrites these links for the site. A link to a doc becomes
 * a site route. A link to another file becomes a link to that file on
 * GitHub. Write each link so that it works on GitHub.
 */

const DOCS_DIR = fileURLToPath(new URL("../../../docs/", import.meta.url));
const REPO_DIR = fileURLToPath(new URL("../../../", import.meta.url));

const EXTERNAL = /^(?:[a-z][a-z0-9+.-]*:|\/\/|#)/i;
const DOC_EXT = /\.mdx?$/;

// The collection maps a README to the ID of its directory.
// See src/content.config.ts.
function slugFor(absPath) {
  const rel = relative(DOCS_DIR, absPath).split(sep).join("/");
  const noExt = rel.replace(DOC_EXT, "");
  const noIndex = noExt.replace(/(^|\/)README$/, "");
  return noIndex.replace(/\/$/, "");
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

      if (DOC_EXT.test(pathPart) && isInside(DOCS_DIR, target)) {
        const slug = slugFor(target);
        node.url = `${prefix}/${slug ? `docs/${slug}` : "docs"}/${suffix}`;
        return;
      }

      if (isInside(REPO_DIR, target)) {
        const rel = relative(REPO_DIR, target).split(sep).join("/");
        node.url = `${repo}/blob/${branch}/${rel}${suffix}`;
      }
    });
  };
}
