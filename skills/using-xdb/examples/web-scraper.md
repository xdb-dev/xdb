# Web Scraper Agent Example

Store scraped web pages with automatic ID generation. Run `xdb --help` for command syntax.

## Setup

```bash
# Create flexible schema for scraped data
xdb make-schema xdb://agent.scraper/pages
```

## Nanoid Helper

```bash
nanoid() { openssl rand -base64 12 | tr -dc 'a-zA-Z0-9' | head -c 21; }
```

## Scrape and Store Function

```bash
scrape_and_store() {
  local url="$1"
  local id=$(nanoid)
  echo "{\"url\": \"$url\", \"content\": \"...\", \"scraped_at\": \"$(date -Iseconds)\"}" | \
    xdb put "xdb://agent.scraper/pages/page-$id"
}
```

## Usage

```bash
# Store a scraped page
scrape_and_store "https://example.com/article"

# List all scraped pages
xdb ls xdb://agent.scraper/pages

# Get specific page (use ID from list output)
xdb get xdb://agent.scraper/pages/page-V1StGXR8Z5jdHi9

# Get page URL only
xdb get xdb://agent.scraper/pages/page-V1StGXR8Z5jdHi9#url
```

## With Strict Schema

For structured scraping, define a schema:

```bash
cat > /tmp/page-schema.json <<EOF
{
  "name": "Page",
  "mode": "strict",
  "fields": [
    {"name": "url", "type": "STRING"},
    {"name": "title", "type": "STRING"},
    {"name": "content", "type": "STRING"},
    {"name": "links", "type": "ARRAY", "array_of": "STRING"},
    {"name": "scraped_at", "type": "TIME"}
  ]
}
EOF

xdb make-schema xdb://agent.scraper/pages --schema /tmp/page-schema.json
```
