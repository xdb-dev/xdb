// Vitest through Astro's Vite config, so tests resolve the same aliases
// and plugins as the site build.
/// <reference types="vitest/config" />
import { getViteConfig } from "astro/config";

export default getViteConfig({
  test: {
    include: ["src/**/*.test.{ts,js}"],
  },
});
