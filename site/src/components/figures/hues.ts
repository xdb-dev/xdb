// The colors of a figure, as CSS variables from public/tokens.css. A part
// of a URI takes the color of its kind on the landing page and in the docs.
export type Hue = "ink" | "muted" | "accent" | "ns" | "schema" | "id" | "attr" | "val";

const VARS: Record<Hue, string> = {
  ink: "var(--ink)",
  muted: "var(--muted)",
  accent: "var(--accent)",
  ns: "var(--c-ns)",
  schema: "var(--c-schema)",
  id: "var(--c-id)",
  attr: "var(--c-attr)",
  val: "var(--c-val)",
};

export const color = (hue: Hue = "ink") => VARS[hue];

// A run of mono text in one color. A line of code is a list of runs.
export type Run = string | [string, Hue];
export type Line = Run[];

export const run = (r: Run): [string, Hue] => (typeof r === "string" ? [r, "ink"] : r);
