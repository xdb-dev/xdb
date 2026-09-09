---
title: Namespaces
description: Logical grouping of schemas by domain, application, or tenant.
package: core
---

# Namespaces

A namespace (NS) groups [schemas](schemas.md), usually by domain, application, or tenant.

From the [CLI](../../cmd/xdb/cli/CONTEXT.md), `xdb namespaces list` lists the namespaces, and `xdb namespaces get xdb://ns` shows one namespace with its schemas. Namespaces are implicit. XDB creates a namespace on the first schema write in it. As a result, namespaces support only the `list` and `get` actions. Run `xdb describe --actions` for the live list.

## Structure

A namespace is its name: a plain string, and the shortest level of a [URI](uris.md).

```
┌──────────────────────────────┐
│  Namespace: com.example      │
├──────────────────────────────┤
│  Schema: posts               │
│  Schema: users               │
│  Schema: comments            │
└──────────────────────────────┘
```

## Naming Rules

A namespace name must match `[a-zA-Z0-9._-]`. It cannot contain `/`. Unlike a
record ID, a namespace is a single URI component.

| Valid         | Invalid           |
| ------------- | ----------------- |
| `com.example` | `` (empty)        |
| `acme-inc`    | `my namespace`    |
| `tenant_123`  | `ns@special`      |
| `io.myapp`    | `org/team`        |

Conventions:

- Reverse domain: `com.example`, `io.myapp`. Good for public or multi-tenant systems.

- Simple names: `myapp`, `analytics`. Good for a single application.

## Creating Namespaces

XDB creates a namespace when you create the first schema in it:

```bash
# Creates the "com.example" namespace and the "posts" schema
xdb schemas create xdb://com.example/posts --file posts.json
```

In Go code, a namespace is the `NS` component of a [URI](uris.md):

```go
uri := core.MustNewURI("com.example")  // xdb://com.example
ns := uri.NS()                         // "com.example"
```

## URI Representation

A namespace URI has the `xdb://` scheme and only the namespace component, for example `xdb://com.example`. This is the shortest valid XDB [URI](uris.md). All other resources extend the namespace URI.

## Related Concepts

- [Schemas](schemas.md): Grouped in namespaces

- [URIs](uris.md): How namespaces are addressed

- [Stores](stores.md): Where namespace data is persisted
