# XDB CLI

Simple, S3-like CLI for managing your XDB data.

## Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

### Check Version

```bash
xdb --version
```

## Global Flags

These flags are available for all commands:

- `--output`, `-o`: Output format (json, table, yaml). Auto-detected by default (table for TTY, JSON for pipes)
- `--config`, `-c`: Path to config file (defaults to xdb.yaml or xdb.yml). Can also be set via `XDB_CONFIG` environment variable
- `--verbose`, `-v`: Enable verbose logging (INFO level)
- `--debug`: Enable debug logging with source locations

### Examples

```bash
# Use table format (default for terminal)
xdb ls xdb://com.example

# Force JSON output
xdb get xdb://com.example/posts/post-123 --output json

# Get JSON and pipe to jq
xdb get xdb://com.example/posts/post-123 | jq '.title'

# Enable verbose logging
xdb -v ls xdb://com.example

# Use a specific config file
xdb --config prod.yaml server
```

## Output Formats

XDB supports multiple output formats for displaying results:

### Automatic Format Selection

By default, XDB automatically selects the appropriate format:

- **Table format**: When output is a terminal (TTY) - human-readable tables
- **JSON format**: When output is piped or redirected - machine-parseable

### Available Formats

- `json`: Indented JSON output (machine-readable)
- `table`: Human-readable tables with borders
- `yaml`: YAML output

### Examples

```bash
# Uses table format (interactive terminal)
xdb get xdb://com.example/users/123

# Uses JSON format (piped to jq)
xdb get xdb://com.example/users/123 | jq '.name'

# Force JSON output even in terminal
xdb get xdb://com.example/users/123 --output json

# Force table output even when piping
xdb ls xdb://com.example/users --output table
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
# List schemas in namespace
xdb ls xdb://com.example

# List with pagination
xdb ls xdb://com.example/users --limit 10
xdb ls xdb://com.example/users --limit 10 --offset 20

# List all (default limit is 100)
xdb ls xdb://com.example
```

`ls` lists schemas in the given namespace. If no URI is provided, it lists all namespaces.

**Flags:**

- `--limit N`: Maximum number of results to return (default: 100)
- `--offset N`: Number of results to skip (default: 0)

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
# Put Record from JSON file
xdb put xdb://com.example/posts/post-123 --file post.json

# Put Record from stdin (JSON)
echo '{"title":"Hello","content":"World"}' | xdb put xdb://com.example/posts/post-123

# Put Record from YAML file
xdb put xdb://com.example/posts/post-123 --file post.yaml --format yaml

# Put with explicit format
echo 'title: Hello
content: World' | xdb put xdb://com.example/posts/post-123 --format yaml
```

`put` creates or updates a Record at the given URI from a JSON or YAML file, or from stdin.

**Flags:**

- `--file`, `-f`: Path to file (reads from stdin if omitted)
- `--format`: Input format: json (default) or yaml

### Remove

```bash
# Remove record (with confirmation prompt)
xdb remove xdb://com.example/posts/post-123

# Remove without confirmation
xdb rm xdb://com.example/posts/post-123 --force

# Remove schema
xdb rm xdb://com.example/posts --force
```

`remove` (aliases: `rm`, `delete`) deletes a Record, Attribute, or Schema at the given URI.

**Flags:**

- `--force`, `-f`: Skip confirmation prompt

## Piping and Automation

XDB commands work well with Unix pipes:

```bash
# Get record and pipe to jq
xdb get xdb://com.example/posts/post-123 | jq '.title'

# List schemas and count them
xdb ls xdb://com.example | jq '. | length'

# Export records to file
xdb get xdb://com.example/posts/post-123 --output json > backup.json

# Import from file
cat backup.json | xdb put xdb://com.example/posts/post-123
```
