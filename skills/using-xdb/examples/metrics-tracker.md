# Metrics Tracking Agent Example

Store time-series metrics with automatic timestamps. Run `xdb --help` for command syntax.

## Setup

```bash
# Create flexible schema for metrics
xdb make-schema xdb://agent.monitor/metrics
```

## Nanoid Helper

```bash
nanoid() { openssl rand -base64 12 | tr -dc 'a-zA-Z0-9' | head -c 21; }
```

## Store Metric Function

```bash
store_metric() {
  local name="$1"
  local value="$2"
  local ts=$(date +%s)
  echo "{\"name\": \"$name\", \"value\": $value, \"timestamp\": $ts}" | \
    xdb put "xdb://agent.monitor/metrics/$(nanoid)"
}
```

## Usage

```bash
# Record various metrics
store_metric "cpu_usage" 45.2
store_metric "memory_mb" 1024
store_metric "requests_per_sec" 150
store_metric "error_rate" 0.02

# List all metrics
xdb ls xdb://agent.monitor/metrics

# Get specific metric (use ID from list output)
xdb get xdb://agent.monitor/metrics/V1StGXR8Z5jdHi9
```

## With Strict Schema

```bash
cat > /tmp/metric-schema.json <<EOF
{
  "name": "Metric",
  "mode": "strict",
  "fields": [
    {"name": "name", "type": "STRING"},
    {"name": "value", "type": "FLOAT"},
    {"name": "unit", "type": "STRING"},
    {"name": "timestamp", "type": "INTEGER"},
    {"name": "tags", "type": "MAP", "map_key": "STRING", "map_value": "STRING"}
  ]
}
EOF

xdb make-schema xdb://agent.monitor/metrics --schema /tmp/metric-schema.json
```

## Store Metric with Tags

```bash
xdb put xdb://agent.monitor/metrics/$(nanoid) <<EOF
{
  "name": "api_latency_ms",
  "value": 42.5,
  "unit": "milliseconds",
  "timestamp": $(date +%s),
  "tags": {"endpoint": "/api/users", "method": "GET", "region": "us-east-1"}
}
EOF
```
