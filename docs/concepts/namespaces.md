---
title: Namespaces
description: Logical grouping of schemas by domain, application, or tenant.
package: core
---

# Namespaces

A **Namespace** (NS) groups one or more [Schemas](schemas.md) together. Namespaces provide logical organization for your data, typically by domain, application, or tenant.

From the [CLI](../../cmd/xdb/cli/CONTEXT.md): `xdb namespaces list` enumerates namespaces; `xdb describe --uri xdb://ns` describes one. Namespaces are implicit — created on first schema write — so the CLI does not expose `create`/`delete` actions for them today (`xdb describe --actions` is authoritative).

## Structure

A namespace _is_ its name — a plain string, the shortest level of a [URI](uris.md).

```
┌──────────────────────────────────────────────┐
│            Namespace: com.example              │
├──────────────────────────────────────────────┤
│  Schema: posts                                │
│  Schema: users                                │
│  Schema: comments                             │
└──────────────────────────────────────────────┘
```

## Naming Rules

Namespace names must match: `[a-zA-Z0-9._-]` (no `/` — unlike a record ID, a
namespace is a single URI component).

| Valid         | Invalid           |
| ------------- | ----------------- |
| `com.example` | `` (empty)        |
| `acme-inc`    | `my namespace`    |
| `tenant_123`  | `ns@special`      |
| `io.myapp`    | `org/team`        |

Conventions:
- **Reverse domain** — `com.example`, `io.myapp` — good for public or multi-tenant systems.
- **Simple names** — `myapp`, `analytics` — fine for single-application use.

## Creating Namespaces

Namespaces are created implicitly when you create a schema within them:

```bash
# Creates the "com.example" namespace and "posts" schema
xdb make-schema xdb://com.example/posts --schema posts.json
```

In Go code a namespace is just the `NS` component of a [URI](uris.md):

```go
uri := core.MustNewURI("com.example")  // xdb://com.example
ns := uri.NS()                         // "com.example"
```

## URI Representation

A namespace URI uses the `xdb://` scheme with just the namespace component (e.g., `xdb://com.example`). This is the shortest valid XDB [URI](uris.md). All other resources extend from the namespace.

## Related Concepts

- [Schemas](schemas.md) — Grouped within namespaces
- [URIs](uris.md) — How namespaces are addressed
- [Stores](stores.md) — Where namespace data is persisted
