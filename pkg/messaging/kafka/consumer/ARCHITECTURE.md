# Kafka Consumer Architecture

## Overview

The Kafka consumer implementation follows a modular architecture similar to the outbox pattern, with separate components handling different responsibilities.

## Components

### 1. **Buffer** (`buffer.go`)
Contains the `channels` structure that provides communication channels between components.

```go
type channels struct {
    messages  chan *kafka.Message
    processed chan *kafka.Message
}
```

- **Purpose**: Decouples message reading from message processing and offset committing
- **Buffer sizes**: 
  - `messages`: 100 messages (from reader to processor)
  - `processed`: 100 messages (from processor to committer)

### 2. **Reader** (`reader.go`)
Responsible for reading messages from Kafka topics.

```go
type reader struct {
    consumer     *kafka.Consumer
    topic        string
    messagesChan chan<- *kafka.Message
    log          *zap.Logger
    // lifecycle management fields
}
```

- **Responsibilities**:
  - Subscribes to Kafka topic
  - Continuously polls Kafka for new messages
  - Handles read timeouts gracefully
  - Pushes messages to the buffer channel
  - Closes Kafka consumer on shutdown
  - Manages its own lifecycle (start/stop)

- **Error Handling**:
  - Ignores timeout errors (normal operation)
  - Logs other read errors and continues
  - Logs errors during consumer close

### 3. **Processor** (`processor.go`)
Processes messages from the buffer channel.

```go
type processor struct {
    messagesChan  <-chan *kafka.Message
    processedChan chan<- *kafka.Message
    handler       Handler
    deserializer  Deserializer
    log           *zap.Logger
    // lifecycle management fields
}
```

- **Responsibilities**:
  - Reads messages from the messages channel
  - Deserializes message payloads
  - Invokes the business logic handler
  - Sends successfully processed messages to committer
  - Implements retry logic with exponential backoff

- **Error Handling**:
  - **Deserialization errors**: Retry with backoff
  - **Skip messages**: If deserializer returns `ErrSkipMessage`, sends to committer immediately
  - **Processing errors**: Retry with backoff (max 10 seconds)

- **Key Methods**:
  - `handleMessage()`: Main processing logic with retry

### 4. **Committer** (`committer.go`)
Commits offsets for successfully processed messages.

```go
type committer struct {
    consumer      *kafka.Consumer
    processedChan <-chan *kafka.Message
    log           *zap.Logger
    // lifecycle management fields
}
```

- **Responsibilities**:
  - Receives processed messages from processor
  - Batches messages for efficient committing
  - Commits offsets with retry logic
  - Performs final commit on shutdown
  - Manages its own lifecycle (start/stop)

- **Batching Strategy**:
  - Collects up to 100 messages
  - Flushes every 1 second (whichever comes first)
  - Commits the highest offset in the batch

- **Error Handling**:
  - **Commit errors**: Retry with exponential backoff (max 10 seconds)
  - Logs detailed information about failed commits

- **Key Methods**:
  - `commitBatch()`: Commits a batch of processed messages
  - `flush()`: Performs final commit during shutdown

### 5. **Consumer** (`consumer.go`)
Main consumer orchestrator that coordinates all components.

```go
type consumer struct {
    consumer  *kafka.Consumer
    topic     string
    reader    *reader
    processor *processor
    committer *committer
    log       *zap.Logger
    // lifecycle management fields
}
```

- **Responsibilities**:
  - Creates and manages Kafka consumer
  - Subscribes to topics
  - Coordinates reader, processor, and committer lifecycle
  - Handles graceful shutdown
  - Final commit on shutdown

## Data Flow

```
Kafka Topic
    ↓
[Reader] ──→ messages channel ──→ [Processor] ──→ processed channel ──→ [Committer]
    │                                    │                                     │
    │                                    ↓                                     │
    │                              Deserializer                                │
    │                                    │                                     │
    │                                    ↓                                     │
    │                                Handler                                   │
    │                                                                          │
    └──────────────────────────────────────────────────────────────────────────┴──→ Offset Commit
```

## Lifecycle Management

All components implement independent lifecycle management using `start()` and `stop()` methods:

1. **Start Sequence**:
   ```
   Consumer.Start()
     ├─→ Subscribe to topic
     ├─→ Reader.start()
     ├─→ Processor.start()
     └─→ Committer.start()
   ```

2. **Stop Sequence**:
   ```
   Consumer.Stop()
     ├─→ Reader.stop()
     ├─→ Processor.stop()
     ├─→ Committer.stop()
     ├─→ Final commit
     └─→ Close Kafka consumer
   ```

## Module Integration (`module.go`)

The module uses Uber FX for dependency injection and lifecycle management:

```go
RegisterHandlerAndConsumer(
    "consumer-name",
    handlerConstructor,
    deserializer,
)
```

This:
- Creates Kafka consumer instance
- Initializes channels for message passing (messages and processed)
- Creates Reader, Processor, and Committer components
- Registers FX lifecycle hooks for each component
- **No consumer orchestrator needed** - FX manages lifecycle directly

## Benefits of Modular Architecture

1. **Separation of Concerns** - each component has a single, well-defined responsibility
2. **Testability** - components can be tested independently
3. **Maintainability** - changes to one component don't affect others
4. **Scalability** - easy to add multiple readers/processors/committers if needed
5. **Consistency** - follows the same pattern as the outbox implementation
6. **Batching** - committer batches offset commits for better performance
7. **At-least-once delivery** - messages are only committed after successful processing
8. **No orchestrator needed** - FX manages lifecycle, reducing complexity

## Comparison with Outbox Pattern

| Outbox | Consumer |
|--------|----------|
| Store | Kafka Consumer Client |
| Fetcher | Reader |
| Sender | Processor |
| Confirmer | Committer |
| Channels (entities, delivery) | Channels (messages, processed) |

## Configuration

Consumer configuration is loaded from YAML:

```yaml
kafka:
  brokers: "localhost:9092"
  consumers:
    groupId: "default-group"
    autoOffsetReset: "earliest"
    consumerConfig:
      - name: "consumer-name"
        topic: "topic-name"
        groupId: "specific-group"  # optional
        autoOffsetReset: "latest"   # optional
```
