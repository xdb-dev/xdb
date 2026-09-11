// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import { unified } from "@astrojs/markdown-remark";

import { stripTitleHeading } from "./src/remark/strip-title-heading.mjs";
import { rewriteDocLinks } from "./src/remark/rewrite-doc-links.mjs";
import { sidebar } from "./src/sidebar.mjs";

const BASE = "/";
const REPO = "https://github.com/xdb-dev/xdb";

// The build copies public/index.html to dist/index.html, so "/" is the
// landing page on the live site. The dev server does not serve an
// index.html from public/ at "/". This integration sends "/" to it.
/** @type {import("astro").AstroIntegration} */
const landingInDev = {
  name: "landing-in-dev",
  hooks: {
    "astro:server:setup": ({ server }) => {
      server.middlewares.use((req, _res, next) => {
        const [path, query] = (req.url ?? "").split("?");
        if (path === BASE) {
          req.url = `${BASE}index.html${query ? `?${query}` : ""}`;
        }
        next();
      });
    },
  },
};

export default defineConfig({
  site: "https://xdb.dev",
  base: BASE,
  trailingSlash: "always",
  markdown: {
    // MDX gets the same plugins from this processor.
    // stripTitleHeading looks only at the first node after the frontmatter.
    processor: unified({
      remarkPlugins: [
        stripTitleHeading,
        [rewriteDocLinks, { base: BASE, repo: REPO }],
      ],
    }),
  },
  integrations: [
    landingInDev,
    starlight({
      title: "xdb",
      description:
        "XDB stores data as tuples: a path, an attribute, and a typed value. Read and write them from Go or the CLI. Keep them in memory, files, Redis, or SQLite.",
      // tokens.css is in public/ because the landing page uses it too.
      // Starlight bundles only the files in customCss.
      customCss: ["./src/styles/starlight.css"],
      head: [
        {
          tag: "link",
          attrs: { rel: "stylesheet", href: `${BASE}tokens.css` },
        },
        // The figures in the docs come from the same script as the figures
        // on the landing page. The script draws only the figures on the page.
        {
          tag: "script",
          attrs: { src: `${BASE}vendor/rough.js`, defer: true },
        },
        {
          tag: "script",
          attrs: { src: `${BASE}site.js`, defer: true },
        },
        {
          tag: "link",
          attrs: { rel: "preconnect", href: "https://fonts.googleapis.com" },
        },
        {
          tag: "link",
          attrs: {
            rel: "stylesheet",
            href: "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap",
          },
        },
      ],
      components: {
        // The eyebrow, the lede, and the package links.
        PageTitle: "./src/components/PageTitle.astro",
        // The edit and issue links above the previous and next pages.
        Footer: "./src/components/Footer.astro",
        // The page title before the direction.
        Pagination: "./src/components/Pagination.astro",
        // The landing page has only a light design, so the docs do too.
        ThemeProvider: "./src/components/ThemeProvider.astro",
        ThemeSelect: "./src/components/ThemeSelect.astro",
      },
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: REPO,
        },
      ],
      editLink: {
        baseUrl: `${REPO}/edit/main/`,
      },
      // The Expressive Code options are in ec.config.mjs.
      sidebar,
    }),
  ],
});
