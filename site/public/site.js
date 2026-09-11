/* Hand-drawn figures for the XDB landing page and docs, rendered with
   rough.js. Every figure is deterministic (fixed seed) so the page looks
   the same on every load. A page draws only the figures whose svg id it
   contains. Colours match the CSS variables in tokens.css. */
(function () {
  const C = {
    accent: "#2b4ee6",
    ink: "#15171c",
    muted: "#6b7280",
    line: "#c9c7bf",
    ns: "#7c3aed",
    schema: "#0369a1",
    id: "#b45309",
    attr: "#047857",
    val: "#be123c",
    fill: "#e8ecfb",
  };
  const NS = "http://www.w3.org/2000/svg";

  function figure(id, w, h, draw) {
    const svg = document.getElementById(id);
    if (!svg || typeof rough === "undefined") return;
    svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
    const rc = rough.svg(svg);
    let seed = 7;
    const base = { roughness: 1.1, strokeWidth: 1.4, bowing: 1.2 };
    const opt = (o) => Object.assign({}, base, { seed: seed++ }, o);

    const api = {
      rect(x, y, w, h, o) { svg.appendChild(rc.rectangle(x, y, w, h, opt(Object.assign({ stroke: C.ink }, o)))); },
      line(x1, y1, x2, y2, o) { svg.appendChild(rc.line(x1, y1, x2, y2, opt(Object.assign({ stroke: C.ink }, o)))); },
      path(d, o) { svg.appendChild(rc.path(d, opt(Object.assign({ stroke: C.ink, fill: "none" }, o)))); },
      ellipse(x, y, w, h, o) { svg.appendChild(rc.ellipse(x, y, w, h, opt(Object.assign({ stroke: C.ink }, o)))); },
      arrow(x1, y1, x2, y2, o) {
        const stroke = (o && o.stroke) || C.accent;
        api.line(x1, y1, x2, y2, Object.assign({ stroke }, o));
        const a = Math.atan2(y2 - y1, x2 - x1);
        const L = 11;
        const p1 = [x2 - L * Math.cos(a - 0.45), y2 - L * Math.sin(a - 0.45)];
        const p2 = [x2 - L * Math.cos(a + 0.45), y2 - L * Math.sin(a + 0.45)];
        api.line(x2, y2, p1[0], p1[1], { stroke, roughness: 0.6 });
        api.line(x2, y2, p2[0], p2[1], { stroke, roughness: 0.6 });
      },
      curve(d, x2, y2, angle, o) {
        const stroke = (o && o.stroke) || C.accent;
        api.path(d, Object.assign({ stroke }, o));
        const L = 11;
        const p1 = [x2 - L * Math.cos(angle - 0.45), y2 - L * Math.sin(angle - 0.45)];
        const p2 = [x2 - L * Math.cos(angle + 0.45), y2 - L * Math.sin(angle + 0.45)];
        api.line(x2, y2, p1[0], p1[1], { stroke, roughness: 0.6 });
        api.line(x2, y2, p2[0], p2[1], { stroke, roughness: 0.6 });
      },
      text(x, y, str, o) {
        o = o || {};
        const t = document.createElementNS(NS, "text");
        t.setAttribute("x", x);
        t.setAttribute("y", y);
        t.setAttribute("class", o.mono ? "mono" : "hand");
        t.setAttribute("font-size", o.size || 15);
        t.setAttribute("fill", o.fill || C.ink);
        t.setAttribute("text-anchor", o.anchor || "start");
        if (o.length) {
          t.setAttribute("textLength", o.length);
          t.setAttribute("lengthAdjust", "spacingAndGlyphs");
        }
        if (Array.isArray(str)) {
          str.forEach((part) => {
            const ts = document.createElementNS(NS, "tspan");
            ts.textContent = part[0];
            if (part[1]) ts.setAttribute("fill", part[1]);
            t.appendChild(ts);
          });
        } else {
          t.textContent = str;
        }
        svg.appendChild(t);
        return t;
      },
      // A labelled box: rough rect + centred hand text.
      box(x, y, w, h, label, o) {
        o = o || {};
        api.rect(x, y, w, h, o);
        if (label) api.text(x + w / 2, y + h / 2 + 5, label, { anchor: "middle", size: o.size || 15, fill: o.color || C.ink, mono: o.mono });
      },
      // Storage sketches, reused in the write figure and the backends figure.
      memory(x, y, s) {
        s = s || 1;
        api.rect(x, y, 120 * s, 84 * s, { stroke: C.accent });
        api.rect(x + 14 * s, y + 26 * s, 92 * s, 44 * s, { stroke: C.accent, fill: C.fill, fillStyle: "hachure", hachureGap: 7, fillWeight: 0.8 });
        api.text(x + 10 * s, y + 18 * s, "map[path]", { size: 11 * s, fill: C.accent });
        api.text(x + 20 * s, y + 43 * s, "map[attr]", { size: 11 * s, fill: C.accent });
        api.text(x + 20 * s, y + 62 * s, "-> value", { size: 11 * s, fill: C.ink });
      },
      folder(x, y, s) {
        s = s || 1;
        api.path(`M${x} ${y + 14 * s} L${x + 34 * s} ${y + 14 * s} L${x + 42 * s} ${y + 6 * s} L${x + 66 * s} ${y + 6 * s} L${x + 66 * s} ${y + 14 * s} L${x + 120 * s} ${y + 14 * s} L${x + 120 * s} ${y + 84 * s} L${x} ${y + 84 * s} Z`, { stroke: C.accent });
        api.line(x + 20 * s, y + 30 * s, x + 20 * s, y + 74 * s, { stroke: C.accent, roughness: 0.8 });
        api.line(x + 20 * s, y + 42 * s, x + 32 * s, y + 42 * s, { stroke: C.accent, roughness: 0.8 });
        api.line(x + 20 * s, y + 58 * s, x + 32 * s, y + 58 * s, { stroke: C.accent, roughness: 0.8 });
        api.line(x + 20 * s, y + 74 * s, x + 32 * s, y + 74 * s, { stroke: C.accent, roughness: 0.8 });
        api.text(x + 14 * s, y + 30 * s, "com.example/", { size: 11 * s, fill: C.ns });
        api.text(x + 36 * s, y + 46 * s, "posts/", { size: 11 * s, fill: C.schema });
        api.text(x + 36 * s, y + 62 * s, "p-1.json", { size: 11 * s, fill: C.id });
        api.text(x + 36 * s, y + 78 * s, "{ title: .. }", { size: 10 * s, fill: C.attr });
      },
      hash(x, y, s) {
        s = s || 1;
        api.rect(x, y + 28 * s, 46 * s, 22 * s, { stroke: C.accent, fill: C.fill, fillStyle: "solid" });
        api.text(x + 5 * s, y + 43 * s, "key", { size: 11 * s, fill: C.accent });
        api.arrow(x + 48 * s, y + 39 * s, x + 62 * s, y + 39 * s);
        api.rect(x + 64 * s, y, 56 * s, 84 * s, { stroke: C.accent });
        api.line(x + 64 * s, y + 28 * s, x + 120 * s, y + 28 * s, { stroke: C.accent, roughness: 0.8 });
        api.line(x + 64 * s, y + 56 * s, x + 120 * s, y + 56 * s, { stroke: C.accent, roughness: 0.8 });
        api.text(x + 69 * s, y + 18 * s, "title", { size: 10 * s, fill: C.attr });
        api.text(x + 69 * s, y + 46 * s, "views", { size: 10 * s, fill: C.attr });
        api.text(x + 69 * s, y + 74 * s, "_ver..", { size: 10 * s, fill: C.attr });
      },
      table(x, y, s) {
        s = s || 1;
        api.rect(x, y, 120 * s, 84 * s, { stroke: C.accent });
        api.rect(x, y, 120 * s, 24 * s, { stroke: C.accent, fill: C.fill, fillStyle: "hachure", hachureGap: 6, fillWeight: 0.7 });
        api.line(x, y + 54 * s, x + 120 * s, y + 54 * s, { stroke: C.accent, roughness: 0.8 });
        api.line(x + 40 * s, y, x + 40 * s, y + 84 * s, { stroke: C.accent, roughness: 0.8 });
        api.line(x + 80 * s, y, x + 80 * s, y + 84 * s, { stroke: C.accent, roughness: 0.8 });
        api.text(x + 8 * s, y + 17 * s, "_id", { size: 10 * s });
        api.text(x + 46 * s, y + 17 * s, "title", { size: 10 * s, fill: C.attr });
        api.text(x + 86 * s, y + 17 * s, "views", { size: 10 * s, fill: C.attr });
        api.text(x + 8 * s, y + 43 * s, "p-1", { size: 10 * s, fill: C.id });
        api.text(x + 46 * s, y + 43 * s, "Hello", { size: 10 * s, fill: C.val });
        api.text(x + 86 * s, y + 43 * s, "42", { size: 10 * s, fill: C.val });
        api.text(x + 8 * s, y + 72 * s, "p-2", { size: 10 * s, fill: C.id });
        api.text(x + 46 * s, y + 72 * s, "..", { size: 10 * s, fill: C.muted });
      },
      yours(x, y, s) {
        s = s || 1;
        api.rect(x, y, 120 * s, 84 * s, { stroke: C.muted, strokeLineDash: [6, 5], roughness: 1.4 });
        api.text(x + 60 * s, y + 38 * s, "your db?", { anchor: "middle", size: 15 * s, fill: C.muted });
        api.text(x + 60 * s, y + 58 * s, "implement Driver", { anchor: "middle", size: 11 * s, fill: C.muted });
      },
    };
    draw(api);
  }

  /* ---------- fig 1: formats in, tuples in the middle, backends out ---------- */
  figure("fig-write", 960, 450, (f) => {
    // Inputs on the left. Each format converts to tuples through its encoder.
    // SVG collapses leading spaces, so indented lines set `indent` instead.
    const inputs = [
      { y: 40, name: "json", lines: [
        { parts: [["{ ", C.muted], ['"title"', C.attr], [": ", C.muted], ['"Hello"', C.val], [",", C.muted]] },
        { indent: 1, parts: [['"views"', C.attr], [": ", C.muted], ["42", C.val], [" }", C.muted]] },
      ] },
      { y: 178, name: "protobuf", lines: [
        { parts: [["message ", C.accent], ["Post {", C.ink]] },
        { indent: 1, parts: [["string ", C.muted], ["title", C.attr], [" = 1;", C.muted]] },
        { parts: [["}", C.ink]] },
      ] },
      { y: 316, name: "go struct", lines: [
        { parts: [["type ", C.accent], ["Post ", C.ink], ["struct", C.accent], [" {", C.ink]] },
        { indent: 1, parts: [["Title ", C.attr], ["string", C.muted]] },
        { parts: [["}", C.ink]] },
      ] },
    ];
    inputs.forEach((t, i) => {
      f.rect(40, t.y, 170, 84);
      const first = t.y + 46 - (t.lines.length - 1) * 9;
      t.lines.forEach((line, j) => {
        f.text(52 + (line.indent || 0) * 14, first + j * 18, line.parts, { mono: true, size: 12 });
      });
      f.text(40, t.y + 104, t.name, { size: 14, fill: C.muted });
      f.arrow(214, t.y + 42, 384, 212 + i * 8);
    });
    f.text(300, 250, "encode", { anchor: "middle", size: 15, fill: C.accent });

    // The middle of the X: every format becomes the same tuples.
    f.rect(390, 120, 180, 200, { fill: C.fill, fillStyle: "hachure", hachureGap: 8, fillWeight: 0.7 });
    f.text(480, 152, "tuples", { anchor: "middle", size: 21 });
    const rows = [
      [["p-1", C.id], ["#", C.muted], ["title", C.attr], [" = ", C.muted], ['"Hello"', C.val]],
      [["p-1", C.id], ["#", C.muted], ["views", C.attr], [" = ", C.muted], ["42", C.val]],
      [["p-2", C.id], ["#", C.muted], ["title", C.attr], [" = ", C.muted], ['"Hi"', C.val]],
    ];
    rows.forEach((row, i) => {
      f.rect(405, 168 + i * 36, 150, 26, { fill: "#fff", fillStyle: "solid", roughness: 0.8 });
      f.text(413, 186 + i * 36, row, { mono: true, size: 11.5 });
    });
    f.text(480, 300, "validate · version", { anchor: "middle", size: 14, fill: C.muted });

    // Out to any backend through its driver.
    const targets = [
      { y: 6, name: "memory", draw: f.memory },
      { y: 116, name: "filesystem", draw: f.folder },
      { y: 226, name: "redis", draw: f.hash },
      { y: 336, name: "sqlite", draw: f.table },
    ];
    targets.forEach((t) => {
      f.arrow(576, 220, 774, t.y + 42);
      t.draw(780, t.y, 1);
      f.text(780, t.y + 100, t.name, { size: 14, fill: C.muted });
    });
    f.text(660, 224, "driver", { anchor: "middle", size: 15, fill: C.accent });
  });

  /* ---------- fig 0: anatomy of a tuple, in the hero ---------- */
  figure("fig-tuple", 900, 270, (f) => {
    const cw = 18; // char width at 30px JetBrains Mono (0.6em)
    const x0 = 60;
    const str = 'xdb://com.example/posts/p-1#title = "Hello"';
    f.text(x0, 120, [
      ["xdb://", C.muted],
      ["com.example", C.ns],
      ["/", C.muted],
      ["posts", C.schema],
      ["/", C.muted],
      ["p-1", C.id],
      ["#", C.muted],
      ["title", C.attr],
      [" = ", C.muted],
      ['"Hello"', C.val],
    ], { mono: true, size: 30, length: str.length * cw });

    const seg = (from, len) => [x0 + from * cw, x0 + (from + len) * cw];
    const parts = [
      { r: seg(6, 11), c: C.ns, top: "namespace", sub: "a domain, app or tenant", row: 0 },
      { r: seg(18, 5), c: C.schema, top: "schema", sub: "strict, flexible or dynamic", row: 1 },
      { r: seg(24, 3), c: C.id, top: "id", sub: "one record", row: 0 },
      { r: seg(28, 5), c: C.attr, top: "attr", sub: "dot.nested allowed", row: 1 },
      { r: seg(36, 7), c: C.val, top: "value", sub: "typed: string, int, time, json ...", row: 0 },
    ];
    parts.forEach((p) => {
      const mid = (p.r[0] + p.r[1]) / 2;
      const y = p.row === 0 ? 192 : 240;
      f.line(p.r[0] + 2, 134, p.r[1] - 2, 134, { stroke: p.c, roughness: 0.9 });
      f.curve(`M${mid} ${y - 24} Q ${mid + 4} ${y - 40} ${mid} ${142}`, mid, 142, -Math.PI / 2, { stroke: p.c });
      f.text(mid, y - 8, p.top, { anchor: "middle", size: 17, fill: p.c });
      f.text(mid, y + 10, p.sub, { anchor: "middle", size: 12, fill: C.muted });
    });

    // the path bracket on top
    const [px1] = seg(6, 11);
    const [, px2] = seg(24, 3);
    f.path(`M${px1} 82 L${px1} 70 L${px2} 70 L${px2} 82`, { stroke: C.ink, roughness: 0.9 });
    f.text((px1 + px2) / 2, 60, "path = the record's address", { anchor: "middle", size: 15 });
    f.text(x0 + 36 * cw + 3.5 * cw, 60, "immutable once written", { anchor: "middle", size: 12, fill: C.muted });
  });

  /* ---------- fig 2: tuples nest into records, schemas, namespaces ---------- */
  figure("fig-nest", 480, 360, (f) => {
    f.rect(10, 34, 460, 316, { stroke: C.ns, roughness: 1.3 });
    f.text(22, 26, "namespace  com.example", { size: 14, fill: C.ns });

    f.rect(40, 84, 400, 246, { stroke: C.schema, roughness: 1.3 });
    f.text(52, 76, "schema  posts", { size: 14, fill: C.schema });
    f.text(300, 76, "strict | flexible | dynamic", { size: 11, fill: C.muted });

    f.rect(70, 134, 340, 176, { stroke: C.id, roughness: 1.2, fill: "#fff", fillStyle: "solid" });
    f.text(82, 126, "record  p-1", { size: 14, fill: C.id });

    const rows = [
      ["title", '"Hello"'],
      ["author.name", '"Ravi"'],
      ["views", "42"],
    ];
    rows.forEach((r, i) => {
      const y = 152 + i * 46;
      f.rect(90, y, 300, 34, { stroke: C.ink, roughness: 0.9 });
      f.text(104, y + 22, [[r[0], C.attr], ["  =  ", C.muted], [r[1], C.val]], { mono: true, size: 13 });
    });
    f.text(240, 300, "a record groups tuples at one path", { anchor: "middle", size: 13, fill: C.muted });
  });

  /* ---------- fig 3: where it lives ---------- */
  figure("fig-backends", 900, 210, (f) => {
    const cols = [
      { x: 25, name: "memory", note: "map[path][attr]", draw: f.memory },
      { x: 205, name: "filesystem", note: "one json file per record", draw: f.folder },
      { x: 385, name: "redis", note: "one hash per record", draw: f.hash },
      { x: 565, name: "sqlite", note: "a table per schema", draw: f.table },
      { x: 745, name: "yours", note: "pass the conformance suite", draw: f.yours },
    ];
    // one tuple comes in from the top
    f.rect(290, 6, 320, 30, { stroke: C.ink, fill: "#fff", fillStyle: "solid" });
    f.text(450, 26, [["com.example/posts/p-1", C.ns], [" # ", C.muted], ["title", C.attr], [" = ", C.muted], ['"Hello"', C.val]], { mono: true, size: 11, anchor: "middle" });
    cols.forEach((c) => {
      const cx = c.x + 60;
      f.curve(`M450 40 Q 450 60 ${cx} 76`, cx, 76, Math.atan2(36, cx - 450), { stroke: C.line });
      c.draw(c.x, 80, 1);
      f.text(cx, 186, c.name, { anchor: "middle", size: 15 });
      f.text(cx, 203, c.note, { anchor: "middle", size: 11, fill: C.muted });
    });
  });

  /* ---------- fig 6: clients share a store ---------- */
  figure("fig-doors", 600, 180, (f) => {
    const doors = [
      { y: 14, label: "Go", sub: "store.Store" },
      { y: 68, label: "JSON-RPC", sub: "records.create" },
      { y: 122, label: "CLI", sub: "xdb records create" },
    ];
    doors.forEach((d) => {
      f.rect(20, d.y, 150, 44, { stroke: C.ink });
      f.text(95, d.y + 19, d.label, { anchor: "middle", size: 15 });
      f.text(95, d.y + 35, d.sub, { anchor: "middle", size: 10, mono: true, fill: C.muted });
      f.curve(`M174 ${d.y + 22} Q 230 ${d.y + 22} 268 90`, 268, 90, Math.atan2(90 - d.y - 22, 94));
    });
    f.rect(272, 50, 150, 80, { fill: C.fill, fillStyle: "hachure", hachureGap: 8, fillWeight: 0.7 });
    f.text(347, 82, "one store", { anchor: "middle", size: 18 });
    f.text(347, 102, "shared validation", { anchor: "middle", size: 11, fill: C.muted });
    f.text(347, 118, "same _version", { anchor: "middle", size: 11, fill: C.muted });
    f.arrow(426, 90, 470, 90);
    f.rect(474, 62, 110, 56, { stroke: C.accent });
    f.text(529, 86, "driver", { anchor: "middle", size: 16, fill: C.accent });
    f.text(529, 104, "any backend", { anchor: "middle", size: 11, fill: C.muted });
    f.text(300, 24, "the CLI calls the daemon over JSON-RPC", { size: 11, fill: C.muted });
  });

  /* ---------- fig 5: the grammar ---------- */
  figure("fig-grammar", 920, 150, (f) => {
    const size = 18;
    const cw = size * 0.6;
    const x0 = 10;
    const str = "xdb <resource> <action> <URI> [--filter CEL] [--fields MASK] [--json | -] [-o FMT]";
    f.text(x0, 120, [
      ["xdb ", C.ink],
      ["<resource>", C.accent],
      [" ", C.ink],
      ["<action>", C.attr],
      [" ", C.ink],
      ["<URI>", C.schema],
      [" ", C.ink],
      ["[--filter CEL]", C.ns],
      [" ", C.ink],
      ["[--fields MASK]", C.ns],
      [" ", C.ink],
      ["[--json | -]", C.val],
      [" ", C.ink],
      ["[-o FMT]", C.muted],
    ], { mono: true, size, length: str.length * cw });
    const mid = (from, len) => x0 + (from + len / 2) * cw;
    const notes = [
      { x: mid(4, 10), t: "the noun", s: "records · schemas · namespaces", c: C.accent, row: 0 },
      { x: mid(15, 8), t: "closed set", s: "actions vary by resource", c: C.attr, row: 1 },
      { x: mid(24, 5), t: "resource URI", s: "depth picks the resource", c: C.schema, row: 0 },
      { x: mid(30, 14), t: "CEL predicate", s: "", c: C.ns, row: 1 },
      { x: mid(45, 15), t: "projection", s: "", c: C.ns, row: 0 },
      { x: mid(61, 12), t: "payload", s: "'-' reads stdin", c: C.val, row: 1 },
      { x: mid(74, 8), t: "output", s: "table on tty, json on a pipe", c: C.muted, row: 0 },
    ];
    notes.forEach((n) => {
      const y = n.row === 0 ? 28 : 64;
      f.text(n.x, y, n.t, { anchor: "middle", size: 15, fill: n.c });
      if (n.s) f.text(n.x, y + 15, n.s, { anchor: "middle", size: 10, fill: C.muted });
      f.curve(`M${n.x} ${y + (n.s ? 20 : 6)} Q ${n.x + 3} ${y + 50} ${n.x} 100`, n.x, 100, Math.PI / 2, { stroke: n.c });
    });
  });

  /* ---------- fig 6: pipes ---------- */
  figure("fig-pipe", 300, 200, (f) => {
    f.box(60, 8, 180, 40, "echo '{\"title\":\"t\"}'", { mono: true, size: 11 });
    f.arrow(150, 52, 150, 76);
    f.text(160, 68, "stdin as -", { size: 12, fill: C.accent });
    f.box(40, 80, 220, 40, "xdb records create <uri> -", { mono: true, size: 11, fill: C.fill, fillStyle: "solid" });
    f.arrow(150, 124, 150, 148);
    f.text(160, 140, "json on a pipe", { size: 12, fill: C.accent });
    f.box(60, 152, 180, 40, "jq  |  xdb batch -", { mono: true, size: 11 });
  });

  /* ---------- fig 8: bring your own types ---------- */
  figure("fig-byot", 520, 270, (f) => {
    const src = [
      { y: 30, t: "user.proto", s: "protobuf" },
      { y: 100, t: "user.schema.json", s: "JSON Schema" },
      { y: 170, t: "type User struct", s: "Go, xdb tags" },
    ];
    src.forEach((d) => {
      f.rect(20, d.y, 150, 50, { stroke: C.ink });
      f.text(95, d.y + 22, d.t, { anchor: "middle", size: 11, mono: true });
      f.text(95, d.y + 40, d.s, { anchor: "middle", size: 11, fill: C.muted });
      f.curve(`M174 ${d.y + 25} Q 220 ${d.y + 25} 250 125`, 250, 125, Math.atan2(125 - d.y - 25, 76));
    });
    f.rect(254, 92, 120, 66, { fill: C.fill, fillStyle: "hachure", hachureGap: 8, fillWeight: 0.7 });
    f.text(314, 120, "schema.Def", { anchor: "middle", size: 14, mono: true });
    f.text(314, 142, "one definition", { anchor: "middle", size: 11, fill: C.muted });
    f.arrow(378, 125, 420, 125);
    f.rect(424, 96, 80, 58, { stroke: C.accent });
    f.text(464, 122, "store", { anchor: "middle", size: 15, fill: C.accent });
    f.text(464, 140, "any backend", { anchor: "middle", size: 10, fill: C.muted });
    // the drift loop
    f.curve("M464 158 Q 464 236 200 236 Q 95 236 95 226", 95, 226, -Math.PI / 2, { stroke: C.muted, strokeLineDash: [5, 5] });
    f.text(300, 258, "xdb schemas diff --check  (CI fails on drift)", { anchor: "middle", size: 12, fill: C.muted });
  });

  /* ---------- fig 9: version checks on transactional backends ---------- */
  figure("fig-cas", 640, 300, (f) => {
    const A = 90, B = 230, R = 520;
    f.text(A, 24, "agent A", { anchor: "middle", size: 15 });
    f.text(B, 24, "agent B", { anchor: "middle", size: 15 });
    f.rect(R - 70, 6, 140, 30, { stroke: C.ink, fill: "#fff", fillStyle: "solid" });
    f.text(R, 26, "posts/p-1", { anchor: "middle", size: 12, mono: true });
    [A, B, R].forEach((x) => f.line(x, 40, x, 290, { stroke: C.line, roughness: 0.6, strokeLineDash: [4, 6] }));

    const row = (y, from, to, label, color, back) => {
      f.arrow(from, y, to, y, { stroke: color });
      f.text((from + to) / 2, y - 6, label, { anchor: "middle", size: 12, fill: color });
    };
    row(64, R, A, "read  ·  _version 3", C.muted);
    row(96, R, B, "read  ·  _version 3", C.muted);
    row(132, A, R, 'write {_version: 3, title: "Hi"}', C.accent);
    f.text(R + 12, 150, "ok -> 4", { size: 12, fill: C.attr });
    row(178, B, R, 'write {_version: 3, title: "Yo"}', C.accent);
    f.text(R + 12, 196, "CONFLICT", { size: 12, fill: C.val });
    row(222, R, B, "re-read  ·  _version 4", C.muted);
    row(258, B, R, "write {_version: 4, ...}", C.accent);
    f.text(R + 12, 276, "ok -> 5", { size: 12, fill: C.attr });
    f.text(B, 292, "retry uses the current version", { anchor: "middle", size: 12, fill: C.muted });
  });

  /* ---------- fig 5: a driver is a composition of capabilities ---------- */
  figure("fig-driver", 560, 290, (f) => {
    // required: four roles stacked, bracketed into one Driver
    f.text(20, 18, "required driver interfaces", { size: 12, fill: C.muted });
    const roles = [
      ["TupleReader", "GetTuples · ScanTuples"],
      ["TupleWriter", "Apply(mutation)"],
      ["SchemaReader", "GetSchema · ScanSchemas"],
      ["SchemaWriter", "Create · Put · Delete · DropRecords"],
    ];
    roles.forEach((r, i) => {
      const y = 30 + i * 52;
      f.rect(20, y, 250, 44, { stroke: C.accent, fill: "#fff", fillStyle: "solid" });
      f.text(32, y + 19, r[0], { size: 14, fill: C.accent });
      f.text(32, y + 36, r[1], { size: 9.5, mono: true, fill: C.muted });
    });
    f.path("M278 32 L286 32 L286 232 L278 232", { stroke: C.ink, roughness: 0.9 });
    f.text(294, 137, "= Driver", { size: 16 });

    // optional: detected by store.New, the facade fills the gap
    f.text(380, 18, "detected by store.New", { size: 12, fill: C.muted });
    const caps = [
      { y: 30, name: "TxDriver", m: "Tx(fn)", have: "memory · sqlite", miss: "otherwise: sequential writes" },
      { y: 128, name: "QueryDriver", m: "QueryTuples(q)", have: "sqlite  (CEL → SQL)", miss: "otherwise: filter a scan" },
    ];
    caps.forEach((c) => {
      f.rect(380, c.y, 160, 60, { stroke: C.ink, strokeLineDash: [6, 5], roughness: 1.3 });
      f.text(392, c.y + 20, c.name, { size: 14 });
      f.text(392, c.y + 36, c.m, { size: 9.5, mono: true, fill: C.muted });
      f.text(392, c.y + 52, c.have, { size: 10, fill: C.attr });
      f.text(380, c.y + 78, c.miss, { size: 11, fill: C.muted });
    });

    // the line no Record crosses
    f.line(20, 250, 540, 250, { stroke: C.line, strokeLineDash: [4, 6], roughness: 0.6 });
    f.text(280, 272, "the facade assembles records from tuple reads", { anchor: "middle", size: 12, fill: C.muted });
  });
})();
