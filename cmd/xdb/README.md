# XDB CLI

Simple, S3-like CLI for managing your XDB data repositories.

## Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

## Quick Start

```bash
# Create repository
xdb make-repo posts

# Put record
xdb put xdb://posts/post-123 --file post.json

# Get record
xdb get xdb://posts/post-123

# List records
xdb ls xdb://posts

# Remove record
xdb rm xdb://posts/post-123
```

## URI Format

```
xdb://REPOSITORY[/RECORD][#ATTRIBUTE]
```

Examples:

```bash
xdb://posts                        # Repository
xdb://posts/post-123               # Record
xdb://posts/post-123#title         # Attribute
xdb://posts/post-123#author.name   # Nested attribute
```

## Commands

### Make Repository

```bash
# Create repository with schema
xdb make-repo posts --schema ./posts.json

# Create repository without schema
xdb make-repo users
```

`make-repo` will create a new repository with the given name and schema. Once the repository is created, you can update the schema later by calling `make-repo` again with the same name and a new schema. All tuple and record operations are automatically validated against the schema.

If no schema is provided, the repository will be created without a schema. This is suitable for storing key-value/NoSQL data.

**Note:** `make-repo` does not support switching between schema-based and key-value storage.

### List

```bash
# List repositories
xdb ls

# List repositories with pattern
xdb ls xdb://posts-*

# List records in repository
xdb ls xdb://posts

# List records in repository with pattern
xdb ls xdb://posts/2024-*
```

`ls` will list all records in the given repository URI. If no URI is provided, it will list all repositories.

### Get

```bash
# Get Repository
xdb get xdb://posts

# Get Record
xdb get xdb://posts/post-123

# Get Attribute
xdb get xdb://posts/post-123#title
```

`get` will retrieve the repository, record or attribute from the given URI.

### Put

```bash
# Put Record
xdb put xdb://posts/post-123 --data '{"title":"Hello"}'

# Put Record with data from file
xdb put xdb://posts/post-123 --file post.json

# Put Attribute
xdb put xdb://posts/post-123#title --value "New Title"
```

`put` will store the record or attribute in the given URI.

### Bulk Operations (Batch Mode)

**Get (URIs from stdin):**

```bash
# Basic pattern
xdb ls <pattern> | xdb get --batch

# Examples
xdb ls xdb://posts | xdb get --batch
cat uris.txt | xdb get --batch
```

**Put (JSONL from stdin):**

```bash
# Basic pattern
cat data.jsonl | xdb put --batch

# JSONL format (one per line)
{"uri": "xdb://posts/post-1", "data": {"title": "First", "content": "..."}}
{"uri": "xdb://posts/post-2", "data": {"title": "Second", "content": "..."}}
```
