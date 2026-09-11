// Records the terminal replay on the landing page. Each session comes from
// one of the agent task evals in internal/evals/tasks and uses the fixture
// data of that task. The script runs the commands of each session against
// a new daemon in a temporary HOME, then writes the commands, their
// output, and their exit codes to site/public/demo.json.
//
//   node site/scripts/record-demo.mjs <path to xdb>
//
// `make site-demo` builds the binary and runs this script. Run it again
// after a change to the output of the CLI or to a fixture.

import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";

if (!process.argv[2]) {
  console.error("usage: node site/scripts/record-demo.mjs <path to xdb>");
  process.exit(2);
}

const xdb = resolve(process.argv[2]);
const out = new URL("../public/demo.json", import.meta.url);
const tasks = new URL("../../internal/evals/tasks/", import.meta.url);

// ---------- fixture data ----------

const fixture = (task, file) =>
  readFileSync(new URL(`${task}/fixtures/${file}`, tasks), "utf8");

// The fixtures have no quoted fields, so a split on commas is enough.
// Some fixtures end their lines with CRLF.
function csv(text) {
  const [head, ...rows] = text.trim().split(/\r?\n/);
  const keys = head.split(",");
  return rows.map((row) =>
    Object.fromEntries(row.split(",").map((value, i) => [keys[i], value])),
  );
}

const ndjson = (rows) => rows.map((row) => JSON.stringify(row)).join("\n") + "\n";
const paise = (rupees) => Math.round(Number(rupees) * 100);

function products() {
  return csv(fixture("ecommerce-store", "catalog.csv")).map((p) => ({
    _id: p.sku,
    title: p.title,
    price_paise: paise(p.price_inr),
    stock: Number(p.stock),
    category: p.category,
  }));
}

// The categories that the household-ledger task asks the agent to use.
const CATEGORIES = [
  [/SALARY/, "salary"],
  [/BIGBASKET/, "groceries"],
  [/RENT/, "rent"],
  [/SWIGGY/, "food-delivery"],
  [/ATM/, "cash"],
  [/ELECTRICITY/, "utilities"],
  [/CREDIT CARD/, "card-payment"],
  [/AMAZON/, "shopping"],
  [/UBER/, "transport"],
  [/INDIGO/, "travel"],
];

const categoryOf = (text) => CATEGORIES.find(([pattern]) => pattern.test(text))[1];

function transactions() {
  // An id is the account and the date, with a counter for a second
  // transaction on the same day.
  const count = {};
  const idFor = (account, date) => {
    const key = `${account}-${date.replaceAll("-", "")}`;
    count[key] = (count[key] ?? 0) + 1;
    return `${key}-${count[key]}`;
  };

  const hdfc = csv(fixture("household-ledger", "hdfc-savings.csv")).map((row) => {
    const credit = row.deposit !== "";
    return {
      _id: idFor("hdfc", row.date),
      account: "hdfc-savings",
      date: `${row.date}T00:00:00Z`,
      description: row.description,
      amount_paise: paise(credit ? row.deposit : row.withdrawal),
      direction: credit ? "credit" : "debit",
      category: categoryOf(row.description),
    };
  });

  const icici = csv(fixture("household-ledger", "icici-credit-card.csv")).map((row) => ({
    _id: idFor("icici", row.date),
    account: "icici-card",
    date: `${row.date}T00:00:00+05:30`,
    description: row.merchant,
    amount_paise: paise(row.amount),
    direction: "debit",
    category: categoryOf(row.merchant),
  }));

  return [...hdfc, ...icici];
}

function issues() {
  return JSON.parse(fixture("issue-tracker", "issues.json")).map((issue) => ({
    _id: String(issue.number),
    title: issue.title,
    state: issue.state,
    priority: issue.priority,
    labels: issue.labels,
    // An unassigned issue has no assignee attribute, so `!has(assignee)`
    // finds it.
    ...(issue.assignee ? { assignee: issue.assignee } : {}),
    created_at: issue.created_at,
  }));
}

// ---------- sessions ----------

// On a terminal, xdb prints tables. This script runs without a terminal,
// so a step with `tty` adds "-o table" to the last command of the step to
// get the output that a person sees. The replay shows the command without
// the flag.
//
// A step fails the recording if its exit code is not `exit` (default 0).
// Thus a change in the behavior of the CLI stops the script.
const EPISODES = [
  {
    id: "store",
    label: "Store",
    title: "Run a small store",
    task: "ecommerce-store",
    files: () => ({ "products.ndjson": ndjson(products()) }),
    steps: [
      {
        note: "Prices are integers in paise, so money never loses precision.",
        cmd: `xdb schemas create xdb://shop/products --json '{"fields":{"title":{"type":"string","required":true},"price_paise":{"type":"integer","required":true},"stock":{"type":"integer"},"category":{"type":"string","indexed":true}}}' --quiet`,
      },
      { cmd: "head -2 products.ndjson" },
      { cmd: "xdb import --uri xdb://shop/products -f products.ndjson", tty: true },
      {
        cmd: `xdb records list xdb://shop/products --filter 'category == "pantry"' --fields _id,title,price_paise,stock`,
        tty: true,
      },
      {
        note: "Two people edit KURTA-COTTON-M at once. Both read version 1.",
        cmd: "xdb records get xdb://shop/products/KURTA-COTTON-M --fields title,stock,_version",
        tty: true,
      },
      {
        cmd: `xdb records update xdb://shop/products/KURTA-COTTON-M --json '{"_version":1,"stock":9}'`,
        tty: true,
      },
      {
        note: "The second write still carries version 1, so it fails and does not overwrite the first.",
        cmd: `xdb records update xdb://shop/products/KURTA-COTTON-M --json '{"_version":1,"stock":8}' -o json`,
        exit: 1,
      },
    ],
  },
  {
    id: "ledger",
    label: "Ledger",
    title: "Track a household ledger",
    task: "household-ledger",
    files: () => ({ "august.ndjson": ndjson(transactions()) }),
    steps: [
      {
        note: "Two bank statements for August, one transaction on each line.",
        cmd: `xdb schemas create xdb://household/transactions --json '{"fields":{"account":{"type":"string","required":true},"date":{"type":"time","required":true},"description":{"type":"string"},"amount_paise":{"type":"integer","required":true},"direction":{"type":"string"},"category":{"type":"string","indexed":true}}}' --quiet`,
      },
      { cmd: "head -2 august.ndjson" },
      { cmd: "xdb import --uri xdb://household/transactions -f august.ndjson", tty: true },
      {
        note: "Groceries in August, in paise.",
        cmd: `xdb records list xdb://household/transactions --filter 'category == "groceries"' -o ndjson | jq -s 'map(.amount_paise) | add'`,
      },
      {
        note: "Every transaction over ₹5,000, in either direction.",
        cmd: `xdb records list xdb://household/transactions --filter 'amount_paise > 500000' --fields _id,description,amount_paise`,
        tty: true,
      },
      {
        note: "The BigBasket charge on 3 August was ₹2,340.50. Patch one field.",
        cmd: `xdb records update xdb://household/transactions/hdfc-20260803-1 --json '{"amount_paise":234050}'`,
        tty: true,
      },
      {
        note: "A rupee string where the schema wants paise fails before anything is written.",
        cmd: `xdb records update xdb://household/transactions/hdfc-20260803-1 --json '{"amount_paise":"2340.50"}' --dry-run -o json`,
        exit: 1,
      },
    ],
  },
  {
    id: "issues",
    label: "Issue tracker",
    title: "Triage an issue tracker",
    task: "issue-tracker",
    files: () => ({ "issues.ndjson": ndjson(issues()) }),
    steps: [
      {
        note: "Twelve issues from a Linear-style export.",
        cmd: `xdb schemas create xdb://tracker/issues --json '{"fields":{"title":{"type":"string","required":true},"state":{"type":"string","indexed":true},"priority":{"type":"string"},"labels":{"type":"array","elem_type":"string"},"assignee":{"type":"string"},"created_at":{"type":"time"}}}' --quiet`,
      },
      { cmd: "xdb import --uri xdb://tracker/issues -f issues.ndjson", tty: true },
      {
        note: "Open issues that nobody owns.",
        cmd: `xdb records list xdb://tracker/issues --filter 'state == "open" && !has(assignee)' --fields _id,priority,title`,
        tty: true,
      },
      {
        note: "Give all of them to rohan in one batch.",
        cmd: `xdb records list xdb://tracker/issues --filter 'state == "open" && !has(assignee)' -o ndjson | jq -c '{op: "records.update", uri: ("xdb://tracker/issues/" + ._id), data: {assignee: "rohan"}}' | xdb batch -`,
      },
      {
        note: "Raise #9 to P0.",
        cmd: `xdb records update xdb://tracker/issues/9 --json '{"priority":"P0"}'`,
        tty: true,
      },
      {
        note: "Open issues for priya.",
        cmd: `xdb records list xdb://tracker/issues --filter 'assignee == "priya" && state == "open"' --fields _id,priority,title`,
        tty: true,
      },
    ],
  },
];

// ---------- recording ----------

function record(episode) {
  // macOS limits the path of a Unix socket to 104 bytes, so the daemon
  // socket must be in a short directory.
  const home = mkdtempSync("/tmp/xdb-demo-");
  const env = {
    ...process.env,
    HOME: home,
    PATH: `${dirname(xdb)}:${process.env.PATH}`,
  };

  const sh = (cmd) => {
    const result = spawnSync("/bin/sh", ["-c", `${cmd} 2>&1`], {
      cwd: home,
      env,
      encoding: "utf8",
    });
    return { text: result.stdout, exit: result.status ?? 1 };
  };

  // The temporary HOME is a detail of the recording. The replay shows "~".
  const tidy = (text) => text.replaceAll(`/private${home}`, "~").replaceAll(home, "~");

  const steps = [];

  try {
    const init = sh("xdb init");
    if (init.exit !== 0) throw new Error(`xdb init failed:\n${init.text}`);

    for (const [name, content] of Object.entries(episode.files())) {
      writeFileSync(join(home, name), content);
    }

    for (const step of episode.steps) {
      const run = step.tty ? `${step.cmd} -o table` : step.cmd;
      const { text, exit } = sh(run);
      const want = step.exit ?? 0;
      if (exit !== want) {
        throw new Error(`${episode.id}: "${run}" exited with ${exit}, not ${want}:\n${text}`);
      }

      steps.push({
        ...(step.note ? { note: step.note } : {}),
        cmd: step.cmd,
        out: tidy(text),
        exit,
      });
    }
  } finally {
    spawnSync(xdb, ["daemon", "stop"], { env });
    rmSync(home, { recursive: true, force: true });
  }

  return {
    id: episode.id,
    label: episode.label,
    title: episode.title,
    task: episode.task,
    steps,
  };
}

const episodes = EPISODES.map(record);
writeFileSync(out, `${JSON.stringify({ episodes }, null, 2)}\n`);

const count = episodes.reduce((n, e) => n + e.steps.length, 0);
console.log(`wrote ${episodes.length} sessions, ${count} steps, to site/public/demo.json`);
