# Pitches of similar projects

Date: 2026-09-11
Method: sub-agents read the homepage or README of 36 projects with WebFetch. They quoted the hero line and paraphrased the rest. Homepage copy changes often, so these tables are a snapshot of 2026-09-11.

## Verdict

Only 2 of the 36 heroes name their unit of data: TerminusDB ("document") and Quadstore ("quads"). None of the tuple and fact databases puts pluggable storage in its hero, although several have it. The storage libraries sell portability and name no data model. A hero that names the tuple and promises any storage has no direct match in this sample.

## Tuple, triple, and fact databases

| Project | Hero | Unit of data on the homepage | Leads with | Status |
| ------- | ---- | ---------------------------- | ---------- | ------ |
| [Datomic](https://www.datomic.com/) | "The fully transactional, cloud-ready, distributed database." | "Facts", lower down. The word "datom" does not appear. | Transactions, then history | Active (2026-07) |
| [XTDB](https://xtdb.com/) | "The database for our time" | None | Time travel, audit | Active (2026-08) |
| [DataScript](https://github.com/tonsky/datascript) | "Immutable database and Datalog query engine for Clojure, ClojureScript and JS" | "Datoms", only in a list near the end | A database as cheap as a hashmap | Active (2026-08) |
| [Datalevin](https://datalevin.org/) | "The database that thinks" | None | AI and reasoning | Active (2026-09) |
| [Datahike](https://datahike.io/) | "The memory model for intelligence" | "Facts", in the paragraph below the hero | Immutability and branches, as memory for AI agents | Active (2026-09) |
| [InstantDB](https://www.instantdb.com/) | "The best backend for AI-coded apps" | None. Triples appear only in the GitHub README. | A full backend for AI-built apps | Shutting down. Services end on 2027-08-31. |
| [Triplit](https://github.com/aspen-cloud/triplit) | Not verified. The site renders only in the browser. The README describes a database that syncs in real time. | None | Real-time sync | Dormant. Supabase acquired it in 2025-10. |
| [Dgraph](https://docs.dgraph.io/) | The site is now docs only. The pitch line claims terabyte-scale, real-time graph use. | None | Scale | Active (2026-08) |
| [Cayley](https://cayley.io/) | "CayleyGraph" | "Quads", lower down | Google Knowledge Graph lineage, then "runs on your existing database" | Dormant (last release 2019) |
| [TerminusDB](https://terminusdb.org/) | "Document Graph Database with Git-for-Data on JSON" | "Document", in the hero | Version control and the data model | Active (2026-08) |
| [FoundationDB](https://www.foundationdb.org/) | "FoundationDB gives you the power of ACID transactions in a distributed database." | "Key-value store", lower down. The tuple layer is in the docs only. | Transactions and scale | Active (2026-09) |
| [Fluree](https://flur.ee/) | "The Unified Intelligence Platform" | None | Enterprise AI | Active (2026-09) |
| [Quadstore](https://github.com/quadstorejs/quadstore) | The README calls it an RDF graph database and triplestore on LevelDB. | "Quads", in the first line | The data model and RDF standards | Low activity (2023) |

FoundationDB pitches the opposite of XDB. It has one store under many data models. XDB has one data model over many stores.

## Storage abstraction libraries

| Project | Hero | Words for the portability promise | Sells a data model? | Status |
| ------- | ---- | --------------------------------- | ------------------- | ------ |
| [Apache OpenDAL](https://opendal.apache.org/) | "One Layer, All Storage." | one, all, unified, same interface | Yes, "one mental model", but for file and object access | Active (2026-09) |
| [Go CDK](https://gocloud.dev/) | "Write once, run on any cloud" | vendor-neutral, portable, any cloud | No. It compares itself to `database/sql`. | Active (2026-06) |
| [gokv](https://github.com/philippgille/gokv) | "Simple key-value store abstraction and implementations for Go" | simple, abstraction | No. It shows the `Store` interface first. | Slow (2024-01) |
| [valkeyrie](https://github.com/kvtools/valkeyrie) | "Distributed Key/Value Store Abstraction Library written in Go." | abstract, multiple backends | No | Low activity (2022) |
| [Dapr state](https://docs.dapr.io/developing-applications/building-blocks/state-management/state-management-overview/) | Data stores are components that you swap without a change to the service code. | pluggable, swapped, portability | Key/value pairs | Active (2026-09) |
| [upper/db](https://upper.io/) | "A productive data access layer for Go" | database agnostic, consistent API | No. It sells productivity. | Slow (2025-03) |
| [Unstorage](https://unstorage.unjs.io/) | "Universal Key-Value." | universal, unified, drivers | Partly. Storages mount like Unix file systems. | Active |
| [Keyv](https://keyv.org/) | "Simple key-value storage with support for multiple backends" | simple, adapters, consistent | No | Active (2026-08) |
| [Flysystem](https://flysystem.thephpleague.com/) | "It provides one interface to interact with many types of filesystems." | one interface, many, vendor lock-in | The file system | Active (2026-09) |
| [fsspec](https://filesystem-spec.readthedocs.io/) | "a unified pythonic interface to local, remote and embedded file systems and bytes storage" | unified, uniform | Names that map to bytes | Active (2026-07) |
| [Mem0](https://mem0.ai/) | "AI memory that persists across sessions and agents" | None on the homepage. Backends are in the config docs. | No. It sells recall and token cost. | Very active |
| [Cognee](https://www.cognee.ai/) | "Any agent. One memory API." | "One API" faces many agents, not many stores. | A knowledge graph | Active (2026-09) |

## ORMs and app backends

| Project | Hero | Leads with | Multi-database support | Agents in the hero |
| ------- | ---- | ---------- | ---------------------- | ------------------ |
| [Ent](https://entgo.io/) | "An entity framework for Go" | The data model: schema as code, graph traversal | README only | No |
| [GORM](https://gorm.io/) | "The fantastic ORM library for Golang" | A feature list | Not on the homepage | No |
| [Bun](https://bun.uptrace.dev/) | "Write elegant SQL queries with type safety and performance." | SQL first | Third tile: four SQL databases | No |
| [sqlc](https://sqlc.dev/) | "Compile SQL to type-safe code; catch failures before they happen." | SQL first, type safety | A link only | No |
| [Prisma](https://www.prisma.io/) | "Your TypeScript app, from prompt to production" | Agents and one platform | Postgres only on the homepage | Yes. It offers a CLI that your agent drives, for build and deploy. |
| [Drizzle](https://orm.drizzle.team/) | "Headless TypeScript ORM with a head." | Personality, then performance | Very prominent: dialect tabs and a logo wall | No |
| [Kysely](https://kysely.dev/) | "Type-safe SQL query builder" | Type safety | Last tile | Second audience for the types |
| [Gel](https://www.geldata.com/) | "Postgres unchained" | Graph-relational data model | Postgres only | AI is a feature |
| [Convex](https://www.convex.dev/) | "All gas no breakages" | Agents and speed | Own database only | Yes |
| [Supabase](https://supabase.com/) | "Build in a weekend. Scale to millions." | Speed to ship | Postgres only | No |
| [TypeORM](https://typeorm.io/) | "Code with Confidence. Query with Power." | Type safety | Third tile: 10 databases, some not SQL | No |

## Patterns

Tuple and fact databases rarely name their unit of data. Datomic, the closest relative of XDB, never says "datom" on its homepage and calls its data "facts" lower on the page. InstantDB stores triples in Postgres and mentions them only in its GitHub README. These databases lead with time instead: history, immutability, time travel, and branches.

Pluggable storage is never the headline of a data-model project. Cayley, Datahike, Datomic, Quadstore, and Triplit all have it, and all put it lower on the page.

Storage libraries describe plumbing. Their words are "unified", "universal", "one API", "agnostic", "consistent interface", and "vendor lock-in". None of them says what the developer thinks in. OpenDAL comes closest with "one mental model", but its model is file access.

ORMs sell type safety, SQL first, or speed to ship. Multi-database support is a feature tile and always means SQL dialects.

Prisma, Convex, InstantDB, Fluree, Datahike, Datalevin, Mem0, Cognee, and Dapr all pitch to agents or to AI. Each of these pitches is about agents that write code against a stack, or agents that remember. None is a data layer that an agent operates directly through a CLI. Prisma's agent CLI is the nearest claim, and it builds and deploys apps.

## Open positions for XDB

- Name the tuple in the hero. No project in the sample does.
- Put memory, files, a key-value store, and SQL behind one model. No hero promises backends beyond SQL dialects under a data model.
- Give an agent the data layer itself at runtime, through the CLI. The current agent pitches are about writing code.

Words to avoid, because the sample already owns them: "unified", "universal", "one API", "agnostic", "any database", "memory", "intelligence", "for AI", and the "The database for X" shape.

## The current candidates

"Think in tuples." takes the first open position directly.

"Store anywhere." is close to Go CDK's "run on any cloud" and to the "anywhere" and "any" wording of the storage libraries. "Storage is a detail." has no match in the sample. "Plug in any store." is close to Dapr's "pluggable". "One API, any store." uses the most common wording in the sample, the same as OpenDAL, Flysystem, and Cognee.

The eyebrow "An agent-first data layer" competes with the Prisma, Convex, and Mem0 pitches. An eyebrow about what the agent does with the data, through the CLI, has no match.

## Sources

- Tuple and fact databases: [Datomic](https://www.datomic.com/), [Datomic Pro releases](https://docs.datomic.com/releases-pro.html), [XTDB](https://xtdb.com/), [DataScript](https://github.com/tonsky/datascript), [Datalevin](https://datalevin.org/), [Datahike](https://datahike.io/), [InstantDB](https://www.instantdb.com/), [Instant repo](https://github.com/instantdb/instant), [Triplit repo](https://github.com/aspen-cloud/triplit), [Triplit joins Supabase](https://supabase.com/blog/triplit-joins-supabase), [Dgraph docs](https://docs.dgraph.io/), [Cayley](https://cayley.io/), [TerminusDB](https://terminusdb.org/), [FoundationDB](https://www.foundationdb.org/), [FoundationDB layer concept](https://apple.github.io/foundationdb/layer-concept.html), [Fluree](https://flur.ee/), [Quadstore](https://github.com/quadstorejs/quadstore)
- Storage abstraction: [OpenDAL](https://opendal.apache.org/), [Go CDK README](https://github.com/google/go-cloud), [gokv](https://github.com/philippgille/gokv), [valkeyrie](https://github.com/kvtools/valkeyrie), [Dapr state overview](https://docs.dapr.io/developing-applications/building-blocks/state-management/state-management-overview/), [upper.io](https://upper.io/), [Unstorage](https://unstorage.unjs.io/), [Keyv](https://keyv.org/), [Flysystem](https://flysystem.thephpleague.com/docs/), [fsspec](https://filesystem-spec.readthedocs.io/en/latest/), [Mem0](https://mem0.ai/), [Cognee](https://www.cognee.ai/)
- ORMs and backends: [Ent](https://entgo.io/), [GORM](https://gorm.io/), [Bun](https://bun.uptrace.dev/), [sqlc](https://sqlc.dev/), [Prisma](https://www.prisma.io/), [Drizzle](https://orm.drizzle.team/), [Kysely](https://kysely.dev/), [Gel](https://www.geldata.com/), [Convex](https://www.convex.dev/), [Supabase](https://supabase.com/), [TypeORM](https://typeorm.io/)
