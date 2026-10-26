// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import { unified } from "@astrojs/markdown-remark";

import { renderMermaid } from "./src/remark/render-mermaid.mjs";
import { stripTitleHeading } from "./src/remark/strip-title-heading.mjs";
import { rewriteDocLinks } from "./src/remark/rewrite-doc-links.mjs";
import { sidebar } from "./src/sidebar.mjs";

const BASE = "/";
const REPO = "https://github.com/xdb-dev/xdb";

export default defineConfig({
  site: "https://xdb.dev",
  base: BASE,
  trailingSlash: "always",
  // The guides were in docs/howto before the site moved them to its own
  // pages. The redirects keep the old links working.
  redirects: {
    "/docs/howto/get-started": "/docs/start/",
    "/docs/howto/define-a-schema": "/docs/guides/define-a-schema/",
    "/docs/howto/import-types": "/docs/guides/import-types/",
    "/docs/howto/read-and-write": "/docs/guides/read-and-write/",
    "/docs/howto/choose-a-backend": "/docs/guides/choose-a-backend/",
    "/docs/howto/use-with-agents": "/docs/guides/use-with-agents/",
    "/docs/howto/embed-in-go": "/docs/guides/embed-in-go/",
  },
  markdown: {
    // MDX gets the same plugins from this processor.
    // stripTitleHeading looks only at the first node after the frontmatter.
    processor: unified({
      remarkPlugins: [
        stripTitleHeading,
        [
          renderMermaid,
          {
            // The fonts of tokens.css. The diagram labels are measured in them.
            fonts: {
              regular: "@fontsource/inter/files/inter-latin-400-normal.woff",
              bold: "@fontsource/inter/files/inter-latin-700-normal.woff",
              mono: "@fontsource/jetbrains-mono/files/jetbrains-mono-latin-400-normal.woff",
              monoBold: "@fontsource/jetbrains-mono/files/jetbrains-mono-latin-700-normal.woff",
            },
            // The content column inside figure.frame.
            maxWidth: 580,
          },
        ],
        [rewriteDocLinks, { base: BASE, repo: REPO }],
      ],
    }),
  },
  integrations: [
    starlight({
      title: "xdb",
      favicon: "/favicon.svg",
      logo: {
        src: "./public/logo.svg",
        alt: "XDB",
        replacesTitle: true,
      },
      description:
        "XDB stores data as tuples: a path, an attribute, and a typed value. Read and write them from Go or the CLI. Keep them in memory, files, Redis, or SQLite.",
      // tokens.css is in public/ because the landing page uses it too.
      // Starlight bundles only the files in customCss.
      customCss: ["./src/styles/starlight.css"],
      head: [
        {
          tag: "meta",
          attrs: { property: "og:image", content: "https://xdb.dev/og.png" },
        },
        {
          tag: "meta",
          attrs: { name: "twitter:image", content: "https://xdb.dev/og.png" },
        },
        {
          tag: "link",
          attrs: { rel: "stylesheet", href: `${BASE}tokens.css` },
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
      // An entry path is relative to site/. A concept page starts with
      // "../docs/", which resolves to the docs directory of the repository.
      editLink: {
        baseUrl: `${REPO}/edit/main/site/`,
      },
      // The Expressive Code options are in ec.config.mjs.
      sidebar,
    }),
  ],
});
