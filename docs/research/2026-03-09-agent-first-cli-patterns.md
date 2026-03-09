# Agent-First CLI Patterns

**Date:** 2026-03-09
**Source:** [Rewrite Your CLI for AI Agents](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)
**Reference Implementation:** [Google Workspace CLI](https://github.com/googleworkspace/cli) (Rust, Apache 2.0)

---

## Core Thesis

Human DX optimizes for discoverability and forgiveness. Agent DX optimizes for predictability and defense-in-depth. These are different enough that retrofitting a human-first CLI for agents doesn't work — you need to design for agents from the start.

---

## The 7 Principles

### 1. Raw JSON Payloads > Bespoke Flags

Humans dislike typing JSON in terminals. Agents are great at constructing it. Instead of flattening complex structures into dozens of `--flags`, accept the full payload via `--json`:

```bash
# Human-friendly (10 flags for a nested structure)
gws sheets create --title "Budget" --sheet-title "Q1" --frozen-rows 1

# Agent-friendly (full API payload, zero translation loss)
gws sheets spreadsheets create --json '{
  "properties": {"title": "Budget"},
  "sheets": [{"properties": {"title": "Q1", "gridProperties": {"frozenRowCount": 1}}}]
}'
```

Support both paths. TTY detection or explicit flags let you serve both audiences from one binary.

### 2. Schema Introspection Replaces Documentation

Agents can't affordably read docs (token cost, staleness). Make the CLI itself queryable at runtime:

```bash
# Type definition
gws schema drive.File

# Method signature — params, request body, response types, required scopes
gws schema sheets.spreadsheets.create
```

**gwscli implementation** (`src/schema.rs`):
- `gws schema service.SchemaName` → returns type definition as JSON
- `gws schema service.resource.method` → returns method details (HTTP method, params, request/response schemas)
- Recursively walks Discovery Document resource tree
- Resolves `$ref` references with cycle detection
- The CLI becomes the canonical, always-current API reference

### 3. Context Window Discipline

API responses are huge. Agents pay per token and lose reasoning with noise.

**Field masks** restrict returned fields:
```bash
gws drive files list --params '{"fields": "files(id,name,mimeType)"}'
```

**NDJSON pagination** with `--page-all` emits one JSON object per line for streaming:
```bash
gws drive files list --page-all  # streams pages as NDJSON
```

**gwscli implementation** (`src/formatter.rs`):
- JSON: pretty for single responses, compact NDJSON for paginated
- Table: flattened dot-notation columns, 60-char max width
- CSV: proper escaping, headers only on first page
- YAML: double-quoted strings, block scalars for multiline, `---` separators between pages

### 4. Input Hardening Against Hallucinations

Agent mistakes are structurally different from human typos. Agents hallucinate path traversals (`../../.ssh`), embed query params in resource IDs, and double-encode strings.

**Treat the agent as an untrusted operator.** Defense-in-depth:

| Input Surface | Validator | Rejects |
|---------------|-----------|---------|
| File paths (write) | `validate_safe_output_dir()` | Absolute paths, `../` traversal, symlinks outside CWD, control chars |
| File paths (read) | `validate_safe_dir_path()` | Same as above |
| Resource names | `validate_resource_name()` | `..`, control chars, `?`, `#`, `%` |
| API identifiers | `validate_api_identifier()` | Non-alphanumeric (except `-`, `_`, `.`) |
| URL path segments | `encode_path_segment()` | Percent-encodes non-alphanumeric chars |
| Enum flags | clap `value_parser` | Anything not in allowlist |

**gwscli implementation** (`src/validate.rs`):
- Separate validation functions per input surface
- Tests for both happy path AND rejection path
- Environment variables are trusted (set by user, not agent)
- Query parameters use reqwest's `.query()` builder (auto-encodes)

**Error messages must be machine-parseable:**
```json
{
  "error": {
    "code": 400,
    "message": "Resource name contains path traversal sequence '..'",
    "reason": "invalidArgument"
  }
}
```

### 5. Ship Agent Skills, Not Just Commands

Humans learn from `--help` and blog posts. Agents learn from injected context at conversation start. Ship structured `SKILL.md` files that encode invariants agents can't intuit.

See [Skills](#skills) section below.

### 6. Multi-Surface: MCP, Extensions, Env Vars

One binary, multiple agent surfaces:

- **MCP** (JSON-RPC over stdio) — eliminates shell escaping ambiguity
- **Gemini CLI Extension** — `gemini extensions install` makes CLI a native capability
- **Environment variables** — for auth without browser redirects

**gwscli auth priority chain** (`src/auth.rs`):
1. `GOOGLE_WORKSPACE_CLI_TOKEN` — raw access token (highest priority)
2. `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` — explicit credentials file (fail-fast if missing)
3. `GOOGLE_APPLICATION_CREDENTIALS` — ADC path (fail-fast if set but missing)
4. `~/.config/gcloud/application_default_credentials.json` — well-known path (silently skip if absent)

**Design rule:** Explicitly-specified env vars must resolve or error. Default/well-known paths silently fall through.

### 7. Safety Rails: Dry-Run + Response Sanitization

**`--dry-run`** validates locally without hitting the API:

```json
{
  "dry_run": true,
  "url": "https://www.googleapis.com/drive/v3/files?pageSize=10",
  "method": "GET",
  "query_params": {"pageSize": "10"},
  "body": null,
  "is_multipart_upload": false
}
```

Returns what *would* happen — URL, method, params, body — without executing.

**`--sanitize`** pipes responses through Google Model Armor to defend against prompt injection embedded in data (e.g., malicious email body telling agent to forward all mail).

---

## Skills

### Three Categories

| Category | Count | Purpose | Example |
|----------|-------|---------|---------|
| **Service skills** (`gws-*`) | 34 | One per API surface — commands, resources, methods | `gws-gmail`, `gws-drive` |
| **Personas** (`persona-*`) | 10 | Role-based bundles — who the agent is | `persona-exec-assistant` |
| **Recipes** (`recipe-*`) | 45 | Multi-step task playbooks | `recipe-backup-sheet-as-csv` |

### Skill Discovery Chain

```
Google Discovery Document
        ↓
  gws generate-skills          (build-time, from CLI metadata)
        ↓
  skills/gws-*/SKILL.md        (files in repo)
        ↓
  npx skills add / gemini extensions install   (user installs)
        ↓
  ~/.openclaw/skills/           (agent framework reads from disk)
        ↓
  Agent context injection       (runtime — framework injects into LLM prompt)
```

The CLI doesn't discover skills at runtime. The agent framework does. The CLI's job is to generate well-structured skill files that frameworks consume through their own discovery mechanisms.

### Installation Methods

```bash
# OpenClaw — install all or cherry-pick
npx skills add https://github.com/googleworkspace/cli
npx skills add https://github.com/googleworkspace/cli/tree/main/skills/gws-drive

# Gemini CLI — extension install
gemini extensions install https://github.com/googleworkspace/cli

# Manual — symlink for local dev
ln -s $(pwd)/skills/gws-* ~/.openclaw/skills/
```

### SKILL.md Format

Each skill is a directory with a single `SKILL.md` file:

```
skills/
  gws-gmail/SKILL.md
  gws-drive/SKILL.md
  persona-exec-assistant/SKILL.md
  recipe-label-and-archive-emails/SKILL.md
```

#### Service Skill Example

```markdown
---
name: gws-gmail
version: 1.0.0
description: "Gmail: Send, read, and manage email."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws gmail --help"
---

# gmail (v1)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules.

## Helper Commands
| Command | Description |
|---------|-------------|
| [`+send`](../gws-gmail-send/SKILL.md) | Send an email |
| [`+triage`](../gws-gmail-triage/SKILL.md) | Show unread inbox summary |

## API Resources
### users
  - `getProfile` — Gets the current user's Gmail profile.
  - `messages` — Operations on the 'messages' resource
  - `labels` — Operations on the 'labels' resource

## Discovering Commands
gws gmail --help
gws schema gmail.<resource>.<method>
```

#### Frontmatter Fields

- `name` — unique identifier, used as directory name
- `version` — semver
- `description` — trigger text for agent frameworks (concise, under 130 chars)
- `metadata.openclaw.category` — indexing/filtering
- `metadata.openclaw.requires.bins` — CLI binaries that must be on `$PATH`
- `metadata.openclaw.requires.skills` — dependent skills to auto-load (used by personas)

### Skill Generation

Skills are auto-generated by `gws generate-skills`, not hand-written.

**Source data:**
- Service skills → Google Discovery Documents + clap command metadata
- Personas → `registry/personas.yaml`
- Recipes → `registry/recipes.yaml`

**Process** (`src/generate_skills.rs`):
1. Iterate all registered services in `src/services.rs`
2. Fetch each service's Discovery Document
3. Build the clap command tree (same as runtime CLI)
4. Extract helper commands, API resources, methods, descriptions, flags
5. Render SKILL.md from templates
6. Write `docs/skills.md` index table

**Blocked methods** — destructive operations excluded from generated skills:
```rust
const BLOCKED_METHODS: &[(&str, &str, &str)] = &[
    ("drive", "files", "delete"),
    ("drive", "files", "emptyTrash"),
    ("drive", "drives", "delete"),
    ("people", "people", "deleteContact"),
];
```

---

## Personas

Personas are role-based skill bundles that tell an agent how to behave for a specific job function.

### Hierarchy

```
Persona  (who you are)       → "Executive Assistant"
  ├─ Services  (what you use)  → gmail, calendar, drive, chat
  ├─ Workflows (what you do)   → +standup-report, +meeting-prep
  ├─ Instructions (how)        → step-by-step guidance
  └─ Tips (watch out for)      → "always confirm before committing"
```

### Definition (registry/personas.yaml)

```yaml
- name: exec-assistant
  title: Executive Assistant
  description: "Manage an executive's schedule, inbox, and communications."
  services: [gmail, calendar, drive, chat]
  workflows: ["+standup-report", "+meeting-prep", "+weekly-digest"]
  instructions:
    - "Start each day with `gws workflow +standup-report`..."
    - "Triage the inbox with `gws gmail +triage --max 10`..."
  tips:
    - "Always confirm calendar changes with the executive before committing."
```

### Generated SKILL.md

```markdown
---
name: persona-exec-assistant
version: 1.0.0
description: "Manage an executive's schedule, inbox, and communications."
metadata:
  openclaw:
    category: "persona"
    requires:
      bins: ["gws"]
      skills: ["gws-gmail", "gws-calendar", "gws-drive", "gws-chat"]
---

# Executive Assistant

> **PREREQUISITE:** Load the following utility skills: `gws-gmail`, `gws-calendar`, `gws-drive`, `gws-chat`

## Relevant Workflows
- `gws workflow +standup-report`
- `gws workflow +meeting-prep`

## Instructions
- Start each day with `gws workflow +standup-report`...

## Tips
- Always confirm calendar changes before committing.
```

The `requires.skills` in frontmatter triggers cascading skill loading — loading `persona-exec-assistant` auto-loads `gws-gmail`, `gws-calendar`, `gws-drive`, and `gws-chat`.

### Purpose

Personas solve the "agent doesn't know what to do first" problem. Without a persona, an agent with 100+ commands has no starting point. With a persona, it has an opinionated playbook mirroring how a human in that role works.

### The 10 Shipped Personas

| Persona | Services | Focus |
|---------|----------|-------|
| exec-assistant | gmail, calendar, drive, chat | Schedule, inbox, communications |
| project-manager | drive, sheets, calendar, gmail, chat | Tasks, meetings, docs |
| hr-coordinator | gmail, calendar, drive, chat | Onboarding, announcements |
| sales-ops | gmail, calendar, sheets, drive | Deals, client comms |
| it-admin | gmail, drive, calendar | Security, configuration |
| content-creator | docs, drive, gmail, chat, slides | Content production |
| customer-support | gmail, sheets, chat, calendar | Tickets, escalation |
| event-coordinator | calendar, gmail, drive, chat, sheets | Planning, invitations |
| team-lead | calendar, gmail, chat, drive, sheets | Standups, 1:1s, OKRs |
| researcher | drive, docs, sheets, gmail | References, notes, collaboration |

---

## Workflows vs Recipes

These are two distinct multi-step patterns:

### Workflows (Compiled Code)

Workflows are **hardcoded helper commands** in `src/helpers/workflows.rs` that orchestrate multiple API calls in a single CLI invocation.

```bash
gws workflow +standup-report     # Calendar (today) + Tasks (open) → combined JSON
gws workflow +meeting-prep       # Calendar (next event) → attendees, description, links
gws workflow +weekly-digest      # Calendar (7 days) + Gmail (unread count)
gws workflow +email-to-task      # Gmail (read) → Tasks (create)
gws workflow +file-announce      # Drive (metadata) → Chat (send message)
```

**How they work internally:**
1. Authenticate with required OAuth scopes
2. Make multiple API calls in sequence
3. Combine results into a single structured JSON response
4. Output via standard formatter

The `workflow` service has `helper_only: true` — no Discovery Document, no generic API commands. It only exists as a container for cross-service helpers.

### Recipes (Declarative YAML, Agent-Executed)

Recipes are **not code**. They're YAML-defined step-by-step instructions rendered into SKILL.md files for agents to follow.

```yaml
# registry/recipes.yaml
- name: label-and-archive-emails
  title: Label and Archive Gmail Threads
  description: "Apply Gmail labels and archive to keep inbox clean."
  category: productivity
  services: [gmail]
  steps:
    - "Search: `gws gmail users messages list --params '{...}'`"
    - "Label: `gws gmail users messages modify --params '{...}' --json '{...}'`"
    - "Archive: `gws gmail users messages modify --params '{...}' --json '{...}'`"
  caution: null  # optional warning for destructive recipes
```

The agent reads the recipe skill, then executes each step as a separate CLI command. The recipe doesn't run as a single invocation — it's a playbook the agent follows.

### Comparison

| | Workflows | Recipes |
|--|----------|---------|
| **Where** | `src/helpers/workflows.rs` (Rust) | `registry/recipes.yaml` (YAML) |
| **Execution** | Single CLI command, multiple API calls | Agent runs each step separately |
| **Who orchestrates** | The CLI binary | The AI agent |
| **Atomicity** | One invocation, combined output | Multiple invocations, agent manages state |
| **When to use** | High-frequency, predictable compositions | Long-tail tasks with decision points |
| **Count** | 5 | 45+ |
| **Flexibility** | Fixed logic | Agent adapts steps based on intermediate results |

**When to make a workflow vs a recipe:**
- Workflow: the steps are always the same, the output is predictable, speed matters
- Recipe: the agent may need to make decisions between steps, or the task is too niche to justify compiled code

---

## gwscli Architecture

### Two-Phase Dynamic Parsing

The CLI uses no hardcoded command definitions for Google APIs:

1. Parse argv → extract service name (e.g., `drive`)
2. Fetch service's Discovery Document (cached 24h)
3. Build `clap::Command` tree from document's resources and methods
4. Re-parse remaining arguments
5. Authenticate, build HTTP request, execute

When Google adds an API endpoint, `gws` picks it up automatically.

### Source Layout

| File | Purpose |
|------|---------|
| `src/main.rs` | Entrypoint, two-phase parsing, method resolution |
| `src/discovery.rs` | Serde models for Discovery Document + fetch/cache |
| `src/services.rs` | Service alias → Discovery API name/version mapping |
| `src/commands.rs` | Recursive clap command builder from Discovery resources |
| `src/executor.rs` | HTTP request construction, response handling, `--dry-run` |
| `src/schema.rs` | `gws schema` command — introspect API method schemas |
| `src/auth.rs` | OAuth2 token acquisition (env vars, encrypted creds, ADC) |
| `src/credential_store.rs` | AES-256-GCM encryption/decryption |
| `src/formatter.rs` | Output formatting (JSON, NDJSON, table, CSV, YAML) |
| `src/validate.rs` | Input hardening (paths, resource names, identifiers) |
| `src/error.rs` | Structured JSON error output |
| `src/generate_skills.rs` | Skill file generation from CLI metadata |
| `src/helpers/*.rs` | Per-service helper commands (+upload, +send, +triage, etc.) |

### Helper Trait Pattern

```rust
pub trait Helper {
    fn inject_commands(&mut self, cmd: Command, doc: &RestDescription) -> Command;
    fn handle(&self, doc: &RestDescription, matches: &ArgMatches, ...) -> Result<bool>;
    fn helper_only(&self) -> bool { false }
}
```

Each service implements custom helper commands:
- Gmail: `+send`, `+triage`, `+watch`
- Drive: `+upload`
- Sheets: `+append`, `+read`
- Calendar: `+insert`, `+agenda`
- Docs: `+write`
- Chat: `+send`
- Workflow: `+standup-report`, `+meeting-prep`, `+email-to-task`, `+weekly-digest`, `+file-announce`

### Credential Storage

AES-256-GCM encryption with OS keyring fallback:
```
~/.config/gws/
├── credentials.enc       # Encrypted credentials
├── .encryption_key       # Fallback key (if no keyring)
└── discovery_cache/      # Cached Discovery Documents (24h TTL)
```

File permissions: `0o700` for directories, `0o600` for files. Atomic writes via temp files.

---

## Incremental Adoption Path

For retrofitting an existing CLI:

1. `--output json` for machine-readable output
2. Validate all inputs against adversarial patterns
3. Add `--describe` / schema command for runtime introspection
4. Support `--fields` to limit response size
5. Add `--dry-run` for mutation validation
6. Ship `CONTEXT.md` / skill files for invariants
7. Expose MCP surface for structured invocation

---

## Key Takeaways for XDB

### Adopt
- Raw JSON via `--json` flag
- `xdb describe` for schema introspection
- `--fields` for field masks
- `--dry-run` on all mutations
- Input hardening (URI validation, file path sandboxing)
- Structured JSON error envelope
- NDJSON streaming for lists
- Skill files for agent context
- Environment variables for all config

### Skip
- Response sanitization (`--sanitize`) — XDB stores user data, not untrusted third-party content
- OAuth / browser auth — local-first daemon, env vars suffice
- Dynamic Discovery Documents — XDB schemas are the discovery mechanism
- Personas — XDB is a data tool, not a multi-service productivity suite
- Recipes — XDB operations are atomic, not multi-step cross-service workflows
