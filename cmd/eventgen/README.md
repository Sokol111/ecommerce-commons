# eventgen - Event Code Generator

`eventgen` is a CLI tool that generates Go code from Avro payload schemas. It combines payload schemas with a standard `EventMetadata` to create complete event envelope schemas.

## Features

- **Single source of truth for EventMetadata** - No more duplicating metadata schema across projects
- **Automatic envelope generation** - Combines payload + metadata into complete event schemas
- **Go type generation** - Uses `avrogen` to generate type-safe Go structs
- **Constants & embeddings** - Generates event type constants, topic constants, and embedded schemas
- **AsyncAPI support** - Extracts topic names from AsyncAPI specs

## Installation

```bash
# Install eventgen
go install github.com/Sokol111/ecommerce-commons/cmd/eventgen@latest

# Install avrogen (required dependency)
go install github.com/hamba/avro/v2/cmd/avrogen@latest
```

## Quick Start

### 1. Create payload schemas

Create Avro schemas for your event payloads in the `avro/` directory. Name them with `_payload.avsc` suffix:

**`avro/product_created_payload.avsc`:**
```json
{
  "type": "record",
  "name": "ProductCreatedPayload",
  "namespace": "com.ecommerce.events.product",
  "doc": "Business data for product creation event",
  "fields": [
    {"name": "product_id", "type": "string"},
    {"name": "name", "type": "string"},
    {"name": "price", "type": "float"}
  ]
}
```

### 2. Generate code

```bash
eventgen generate \
  --payloads ./avro \
  --output ./gen/events \
  --package events \
  --asyncapi ./asyncapi/asyncapi.yaml \
  --verbose
```

### 3. Generated files

```
gen/events/
├── schemas/
│   ├── event_metadata.avsc           # EventMetadata schema
│   ├── product_created_envelope.avsc # Complete event schema
│   └── product_created_combined.json # Self-contained schema for Avro
├── types.gen.go                      # Go types
├── constants.gen.go                  # Event type & topic constants
├── schemas.gen.go                    # Embedded schemas
└── asyncapi.yaml                     # Copied AsyncAPI spec
```

## CLI Reference

### `eventgen generate`

Generate Go code from Avro payload schemas.

```
Flags:
  -p, --payloads string    Directory containing *_payload.avsc files (required)
  -o, --output string      Output directory for generated code (required)
  -n, --package string     Go package name (default "events")
  -a, --asyncapi string    AsyncAPI spec file for topic extraction
  -m, --metadata string    Custom EventMetadata schema file
      --metadata-namespace string  Namespace for EventMetadata (default "com.ecommerce.events")
  -v, --verbose            Enable verbose output
```

### `eventgen validate`

Validate payload schemas without generating code.

```
Flags:
  -p, --payloads string    Directory containing *_payload.avsc files (required)
  -v, --verbose            Enable verbose output
```

## Integration

### Makefile

Include the provided Makefile fragment in your project:

```makefile
include $(shell go list -m -f '{{.Dir}}' github.com/Sokol111/ecommerce-commons)/pkg/eventgen/makefiles/eventgen.mk

# Or define targets manually:
generate-events:
	eventgen generate --payloads ./avro --output ./gen/events --package events
```

### go:generate

Add a `doc.go` file to your generated package:

```go
// Package events contains generated event types and schemas.
//go:generate go run github.com/Sokol111/ecommerce-commons/cmd/eventgen generate --payloads ../../avro --output . --package events
package events
```

Then run:
```bash
go generate ./gen/events/...
```

### GitHub Actions

Use the reusable workflow:

```yaml
jobs:
  generate-events:
    uses: Sokol111/ecommerce-commons/.github/workflows/eventgen.yml@main
    with:
      payloads-dir: avro
      output-dir: gen/events
      package-name: events
      asyncapi-file: asyncapi/asyncapi.yaml
      check-diff: true
```

Or run directly:

```yaml
- name: Generate events
  run: |
    go install github.com/hamba/avro/v2/cmd/avrogen@latest
    go install github.com/Sokol111/ecommerce-commons/cmd/eventgen@latest
    eventgen generate --payloads ./avro --output ./gen/events --package events
```

## Generated Code Usage

```go
package main

import (
    "github.com/your-org/your-api/gen/events"
    "github.com/hamba/avro/v2"
)

func main() {
    // Create an event
    event := &events.ProductCreatedEvent{
        Metadata: events.EventMetadata{
            EventId:   "uuid-here",
            EventType: events.EventTypeProductCreated,
            Source:    "catalog-service",
            Timestamp: time.Now().UnixMilli(),
        },
        Payload: events.ProductCreatedPayload{
            ProductId: "prod-123",
            Name:      "Widget",
            Price:     99.99,
        },
    }

    // Parse schema from embedded bytes
    schema, _ := avro.Parse(string(events.ProductCreatedCombinedSchema))
    
    // Serialize
    data, _ := avro.Marshal(schema, event)
}
```

## Migration from Legacy Approach

If you're migrating from the old approach with `event_metadata.avsc` in each project:

1. **Remove duplicated metadata schemas:**
   ```bash
   rm avro/event_metadata.avsc
   ```

2. **Rename event schemas to payload schemas:**
   ```bash
   # Old: product_created.avsc with embedded metadata
   # New: product_created_payload.avsc with just payload
   ```

3. **Simplify payload schemas:**
   Remove the `metadata` field and envelope wrapper - keep only the business payload.

4. **Update Makefile:**
   Replace old generation commands with `eventgen generate`.

5. **Run generation:**
   ```bash
   make generate-events
   ```

## Contributing

The eventgen tool is part of ecommerce-commons. To contribute:

1. Make changes in `pkg/eventgen/`
2. Run tests: `go test ./pkg/eventgen/...`
3. Test locally: `go run ./cmd/eventgen generate --payloads ./pkg/eventgen/testdata --output /tmp/test --verbose`
