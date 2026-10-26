// The parts of the figures that more than one page draws. The landing page
// and the guides import them, so the two pages show the same figure.
import type { ComponentProps } from "astro/types";
import type Anatomy from "../components/figures/Anatomy.astro";

type Parts = ComponentProps<typeof Anatomy>["parts"];

// A tuple: the URI of a record, an attribute, and a typed value.
export const tuple: { parts: Parts; groups: ComponentProps<typeof Anatomy>["groups"] } = {
  groups: [{ id: "path", label: "path = the record's address" }],
  parts: [
    { text: "xdb://", hue: "muted" },
    { text: "com.example", hue: "ns", label: "namespace", sub: "a domain, app, or tenant", group: "path" },
    { text: "/", hue: "muted", group: "path" },
    { text: "posts", hue: "schema", label: "schema", sub: "strict, flexible, or dynamic", row: 1, group: "path" },
    { text: "/", hue: "muted", group: "path" },
    { text: "p-1", hue: "id", label: "id", sub: "one record", group: "path" },
    { text: "#", hue: "muted" },
    { text: "title", hue: "attr", label: "attr", sub: "dot.nested allowed", row: 1 },
    { text: " = ", hue: "muted" },
    { text: '"Hello"', hue: "val", label: "value", sub: "typed: string, int, time, json", over: "immutable once written" },
  ],
};

// The grammar of a resource command of the CLI.
export const grammar: Parts = [
  { text: "xdb " },
  { text: "<resource>", hue: "accent", label: "the noun", sub: "records · schemas · namespaces" },
  { text: " " },
  { text: "<action>", hue: "attr", label: "closed set", sub: "actions vary by resource", row: 1 },
  { text: " " },
  { text: "<URI>", hue: "schema", label: "resource URI", sub: "depth picks the resource" },
  { text: " " },
  { text: "[--filter CEL]", hue: "ns", label: "CEL predicate", row: 1 },
  { text: " " },
  { text: "[--fields MASK]", hue: "ns", label: "projection" },
  { text: " " },
  { text: "[--json | -]", hue: "val", label: "payload", sub: "'-' reads stdin", row: 1 },
  { text: " " },
  { text: "[-o FMT]", hue: "muted", label: "output", sub: "table on tty, json on a pipe" },
];
