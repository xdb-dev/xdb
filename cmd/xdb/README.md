# XDB CLI

Simple, S3-like CLI for managing your XDB data.

## Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

## Quick Start

```bash
# Create schema
xdb make-schema xdb://com.example/posts --schema posts.json

# List schemas
xdb ls xdb://com.example

# Get schema
xdb get xdb://com.example/posts

# Remove schema
xdb rm xdb://com.example/posts

# Put record
xdb put xdb://com.example/posts/post-123 --file post.json

# Get record
xdb get xdb://com.example/posts/post-123

# List records
xdb ls xdb://com.example/posts

# Remove record
xdb rm xdb://com.example/posts/post-123
```

## URI Format

```
xdb://NS[/SCHEMA][/ID][#ATTRIBUTE]
```

Examples:

```bash
xdb://com.example                        # Namespace (NS)
xdb://com.example/posts                  # Schema (NS + Schema)
xdb://com.example/posts/post-123         # Record (NS + Schema + ID)
xdb://com.example/posts/post-123#title   # Attribute (Record + Attribute)
xdb://com.example/posts/post-123#author.name   # Nested attribute
```

## Commands

### Make Schema

```bash
# Create schema from definition file
xdb make-schema xdb://com.example/posts --schema ./posts.json

# Create flexible schema (no definition file)
xdb make-schema xdb://com.example/users
```

`make-schema` creates or updates a schema at the given URI. The URI must include both NS and Schema components (e.g., `xdb://com.example/posts`).

If no schema definition file is provided, a flexible schema is created. Flexible schemas allow arbitrary data without validation. Schema definitions enforce structure and types on all Tuple and Record operations.

**Note:** You can update a schema by calling `make-schema` again with the same URI and a new schema definition.

### List

```bash
# List namespaces
xdb ls

# List namespaces matching pattern
xdb ls xdb://com.example*

# List schemas in namespace
xdb ls xdb://com.example

# List records in schema
xdb ls xdb://com.example/posts

# List records matching pattern
xdb ls xdb://com.example/posts/2024-*
```

`ls` lists namespaces, schemas, or records from the given URI. If no URI is provided, it lists all namespaces.

### Get

```bash
# Get Namespace
xdb get xdb://com.example

# Get Schema
xdb get xdb://com.example/posts

# Get Record
xdb get xdb://com.example/posts/post-123

# Get Attribute
xdb get xdb://com.example/posts/post-123#title
```

`get` retrieves a Namespace, Schema, Record, or Attribute from the given URI.

### Put

```bash
# Put Record
xdb put xdb://com.example/posts/post-123 --data '{"title":"Hello"}'

# Put Record with data from file
xdb put xdb://com.example/posts/post-123 --file post.json

# Put Attribute
xdb put xdb://com.example/posts/post-123#title --value "New Title"
```

`put` stores a Record or Attribute at the given URI.

### Bulk Operations (Batch Mode)

**Get (URIs from stdin):**

```bash
# Basic pattern
xdb ls <pattern> | xdb get --batch

# Examples
xdb ls xdb://com.example/posts | xdb get --batch
cat uris.txt | xdb get --batch
```

**Put (JSONL from stdin):**

```bash
# Basic pattern
cat data.jsonl | xdb put --batch

# JSONL format (one per line)
{"uri": "xdb://com.example/posts/post-1", "data": {"title": "First", "content": "..."}}
{"uri": "xdb://com.example/posts/post-2", "data": {"title": "Second", "content": "..."}}
```
