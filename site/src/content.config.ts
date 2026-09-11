import { defineCollection } from "astro:content";
import { z } from "astro/zod";
import { glob } from "astro/loaders";
import { docsSchema } from "@astrojs/starlight/schema";

// The pages come from the /docs directory of the repository. GitHub shows
// the same files, so one change updates both.
//
// Plans and research are internal notes. The pattern does not include them.
//
// Each ID starts with "docs/", so "/" stays free for the landing page.
// A README.md file is the index of its directory.
export const collections = {
  docs: defineCollection({
    loader: glob({
      base: "../docs",
      pattern: ["README.md", "concepts/**/*.{md,mdx}", "howto/**/*.{md,mdx}"],
      generateId: ({ entry }) => {
        const path = entry
          .replace(/\.mdx?$/, "")
          .replace(/(^|\/)README$/, "")
          .replace(/\/$/, "");
        return path ? `docs/${path}` : "docs";
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
