import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { beforeAll, describe, expect, it } from "vitest";

import Anatomy from "./Anatomy.astro";
import Flow from "./Flow.astro";
import Sequence from "./Sequence.astro";
import { grammar, tuple } from "../../figures/parts";

let container: AstroContainer;

beforeAll(async () => {
  container = await AstroContainer.create();
});

const count = (html: string, pattern: RegExp) => html.match(pattern)?.length ?? 0;

describe("Flow", () => {
  it("draws one edge for each source and each target", async () => {
    const html = await container.renderToString(Flow, {
      props: {
        alt: "a flow",
        from: [{ title: "a" }, { title: "b" }, { title: "c" }],
        center: { title: "hub" },
        to: [{ title: "x" }, { title: "y" }],
      },
    });
    expect(count(html, /<path d="M0 /g)).toBe(5);
    expect(count(html, /class="fl-card/g)).toBe(5);
  });

  it("leaves out the side that has no cards", async () => {
    const html = await container.renderToString(Flow, {
      props: { alt: "a fan", center: { title: "hub" }, to: [{ title: "x" }] },
    });
    expect(html).not.toContain('class="fl-center in');
    expect(count(html, /class="fl-edge/g)).toBe(1);
  });
});

describe("Anatomy", () => {
  it("lists each callout and each group in the legend", async () => {
    const html = await container.renderToString(Anatomy, {
      props: { alt: "a tuple", parts: tuple.parts, groups: tuple.groups },
    });
    const called = tuple.parts.filter((p) => p.label).length;
    expect(count(html, /<li /g)).toBe(called + (tuple.groups?.length ?? 0));
    expect(count(html, /class="an-group/g)).toBe(1);
  });

  it("keeps each part of the line on one line of markup", async () => {
    const html = await container.renderToString(Anatomy, {
      props: { alt: "the grammar", parts: grammar },
    });
    const line = html.slice(html.indexOf('class="an-line'), html.indexOf('class="an-legend'));
    expect(line).not.toMatch(/>\s*\n\s*</);
  });
});

describe("Sequence", () => {
  it("points each arrow from its sender to its target", async () => {
    const html = await container.renderToString(Sequence, {
      props: {
        alt: "a race",
        actors: [{ name: "A" }, { name: "B" }],
        messages: [
          { from: 0, to: 1, label: "write" },
          { from: 1, to: 0, label: "read" },
        ],
      },
    });
    expect(count(html, /sq-arrow forth/g)).toBe(1);
    expect(count(html, /sq-arrow back/g)).toBe(1);
    expect(html).toContain("left:25%;width:50%");
  });
});
