# XDB CLI

Simple, S3-like CLI for managing your XDB data repositories.

## Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

## Quick Start

```bash
# Start server
xdb server

# Create repository
xdb mr posts

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

### Repository Management

```bash
# Create repository with schema
xdb make-repo posts --schema ./posts.json

# Create repository without schema (stored as key-value pairs)
xdb make-repo users

# List repositories
xdb ls

# List records in repository
xdb ls xdb://posts

# Purge repository (includes all records recursively)
xdb purge xdb://posts --recursive
```

### Single Record Operations

```bash
# Get
xdb get xdb://posts/post-123
xdb get xdb://posts/post-123 --output post.json
xdb get xdb://posts/post-123#title

# Put
xdb put xdb://posts/post-123 --file post.json
xdb put xdb://posts/post-123 --data '{"title":"Hello"}'
xdb put xdb://posts/post-123#title --value "New Title"

# Remove
xdb rm xdb://posts/post-123
```

### Bulk Operations (Batch Mode)

**Get (URIs from stdin):**

```bash
# Basic pattern
xdb ls <pattern> | xdb get --batch

# Examples
xdb ls xdb://posts | xdb get --batch
xdb ls xdb://posts/2024-* | xdb get --batch --output ./backup/
xdb query xdb://posts --filter "published=true" | xdb get --batch
cat uris.txt | xdb get --batch

# Advanced
xdb ls xdb://posts | xdb get --batch --concurrency 50
xdb ls xdb://posts | xdb get --batch | gzip > backup.jsonl.gz
xdb ls xdb://users | sed 's/$/#email/' | xdb get --batch
```

**Put (JSONL from stdin):**

```bash
# Basic pattern
cat data.jsonl | xdb put --batch

# JSONL format (one per line)
{"uri": "xdb://posts/post-1", "data": {"title": "First", "content": "..."}}
{"uri": "xdb://posts/post-2", "data": {"title": "Second", "content": "..."}}
```

**Remove (URIs from stdin):**

```bash
xdb ls xdb://posts | xdb rm --batch
xdb query xdb://posts --filter "archived=true" | xdb rm --batch
```
