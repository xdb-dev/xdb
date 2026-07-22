---
name: schema-evolution
description: "Evolve schemas safely with patch updates and revision CAS"
category: recipe
---

# Schema Evolution

Schemas update with patch semantics: fields are added or replaced,
never removed. Every update bumps the schema's `revision`.

## Inspect the current definition

```bash
xdb schemas get --uri xdb://myapp/todos
xdb describe --uri xdb://myapp/todos
```

## Add a field

```bash
xdb schemas update --uri xdb://myapp/todos --json '{
  "fields": {
    "priority": {"type": "integer"}
  }
}'
```

Existing records stay readable; the new field is simply absent until
records set it. Add fields as optional first — marking a field
`"required": true` breaks writes for records that omit it.

## Safe rollout order

1. `schemas update` adds the new optional field.
2. Backfill existing records (`xdb records update` or a bulk import).
3. Tighten to required only after every record carries the field.

## Concurrency: revision CAS

`schemas get` returns the current `revision`. Pass it in an update to
fail with CONFLICT if someone else evolved the schema in between
(omit it for an unconditional update):

```bash
xdb schemas update --uri xdb://myapp/todos --json '{
  "revision": 3,
  "fields": {"priority": {"type": "integer"}}
}'
```

## Conflicts on create

Re-creating a schema with the identical definition is an idempotent
success; a different definition fails with CONFLICT — use
`schemas update` to evolve instead.

## Modes

`xdb describe --schema-format` documents the three validation modes:
strict (only declared fields), flexible (undeclared fields pass
through), dynamic (undeclared fields evolve the schema automatically).
