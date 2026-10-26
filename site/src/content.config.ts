import { defineCollection } from "astro:content";
import { z } from "astro/zod";
import { glob } from "astro/loaders";
import { docsSchema } from "@astrojs/starlight/schema";

// The site has two sources. The pages for the users of XDB live in
// src/content/docs. The concept pages live in the repository's
// /docs/concepts, so GitHub and the site show the same files. The rest of
// /docs is not part of the site.
//
// IDs get a "docs/" prefix to leave "/" free for the landing page. A page
// keeps its path under src/content/docs. A concept page keeps its path
// under /docs. index.md and README.md become the index of their directory.
const PAGES = "site/src/content/docs/";

export const collections = {
  docs: defineCollection({
    loader: glob({
      base: "..",
      pattern: [`${PAGES}**/*.{md,mdx}`, "docs/concepts/**/*.{md,mdx}"],
      generateId: ({ entry }) => {
        const path = entry
          .replace(PAGES, "docs/")
          .replace(/\.mdx?$/, "")
          .replace(/(^|\/)(index|README)$/, "")
          .replace(/\/$/, "");
        return path || "docs";
      },
    }),
    schema: (ctx) =>
      docsSchema({
        extend: z.object({
          // The Go packages that the page documents, separated by commas.
          package: z.string().optional(),
          // The tasks that the page supports.
          read_when: z.array(z.string()).optional(),
        }),
      })(ctx),
  }),
};
