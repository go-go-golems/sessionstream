---
title: "Investigation Diary"
status: "active"
intent: "long-term"
topics:
  - "llm"
  - "event-streaming"
  - "agents"
  - "architecture"
  - "framework"
created: "2026-04-19"
doc-type: "reference"
---

# Investigation Diary: Event Streaming LLM Framework

## Entry 1: Initial Analysis - 2026-04-19

### What was analyzed

Analyzed two architecture diagrams for building a reusable base framework for event streaming LLM applications.

#### Diagram 1: Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    AGENTIC AI FRAMEWORK                 │
├─────────────────────────────────────────────────────────┤
│                   TOOL + MEMORY + RAG                   │
├─────────────────────────────────────────────────────────┤
│                  STREAMING LLM APPS                     │
└─────────────────────────────────────────────────────────┘
```

This 3-layer model shows the natural abstraction hierarchy:
- **Bottom (Streaming LLM Apps)**: Raw streaming, connection management, low-level events
- **Middle (Tool + Memory + RAG)**: Domain-specific capabilities, composable building blocks
- **Top (Agentic AI Framework)**: High-level orchestration, policies, multi-step reasoning

#### Diagram 2: Detailed Component Architecture

Key components identified:

| Layer | Components |
|-------|------------|
| Inputs | Chat, Prompt, Vision, File, Context, Streaming |
| Core | LLM with streaming orchestration |
| Outputs | Text, JSON, Tool Calls, Code, Streaming, Structured Output |
| Memory | Vector DB, KV Store, Graph DB |
| Orchestration | Streaming Orchestration, Event Broker |
| Execution | Tools (Web, Code Interpreter, Search, Custom) |
| Patterns | ReAct Loop, Custom Events |

### What worked

- Clear separation between input sources and output destinations
- Event broker as central hub enables loose coupling
- Multiple memory backends (vector, KV, graph) serve different use cases
- Tool registry pattern allows extensibility without modifying core

### What didn't work

- Diagrams don't specify error handling or retry strategies
- Missing explicit connection between Event Broker and Memory layer
- No indication of how multi-turn context is maintained across sessions

### What was tricky to understand

- The relationship between "Streaming Orchestration" and "Event Broker" — are they separate components or the same?
  - **Resolution**: Streaming Orchestration handles async coordination/backpressure at the connection level; Event Broker handles routing/distribution at the message level

### Key decisions made

1. **Event Broker is the central component** — all LLM outputs flow through it
2. **Three memory types serve distinct purposes**:
   - Vector DB: Semantic search, RAG context
   - KV Store: Fast session state
   - Graph DB: Entity relationships, knowledge graphs
3. **ReAct loop is a first-class citizen**, not an afterthought

### Code review instructions

When reviewing framework implementation, check:
- [ ] Event broker supports at-least-once delivery
- [ ] Streaming backpressure doesn't block the event loop
- [ ] Memory backends are swappable via interface
- [ ] Tool execution is sandboxed
- [ ] ReAct loop has configurable max iterations to prevent infinite loops

---

## Entry 2: Architecture Decisions - 2026-04-19

### Decision: Event Broker Pattern

**Context**: Need to decouple LLM streaming from downstream consumers (UI, tools, memory)

**Options considered**:
1. Direct callback/promise — tight coupling
2. Message queue (Kafka/RabbitMQ) — adds infrastructure complexity
3. In-process pub/sub event broker — lightweight, flexible

**Chosen**: In-process pub/sub with pluggable transport

```typescript
// Core event broker interface
interface EventBroker {
  emit(event: LLMEvent): Promise<void>;
  subscribe(types: EventType[], handler: EventHandler): () => void;
}

// Default: in-memory implementation
// Future: can swap to Redis/Kafka for distributed deployment
```

**Tradeoffs**: 
- Pro: Zero infrastructure for simple deployments
- Pro: Easy to test (in-process)
- Con: Doesn't scale horizontally without transport swap

### Decision: Memory Abstraction

**Context**: Different use cases need different storage backends

**Chosen**: Unified interface with three backend implementations

```typescript
interface MemoryStore {
  // All operations return Promises for async backends
  get(key: string): Promise<unknown>;
  set(key: string, value: unknown): Promise<void>;
  
  // Vector-specific
  embed(text: string): Promise<Float32Array>;
  search(query: string, topK: number): Promise<SearchResult[]>;
  
  // Graph-specific
  addNode(label: string, properties: Record<string, unknown>): Promise<string>;
  relate(from: string, relation: string, to: string): Promise<void>;
}
```

**Tradeoffs**:
- Pro: Apps choose which backends to instantiate
- Pro: Easy to mock for tests
- Con: Some queries can't be expressed across backends

### Code review instructions

When reviewing memory implementation, check:
- [ ] Embedding model is configurable (OpenAI, local models)
- [ ] Vector index is updated on new memory entries
- [ ] KV store has TTL support for automatic cleanup
- [ ] Graph schema migrations are handled gracefully

---

## Entry 3: Next Steps

### Immediate actions
- [ ] Define event type catalog with schemas
- [ ] Prototype event broker with in-memory transport
- [ ] Implement ReAct loop skeleton

### Deferred decisions
- [ ] How to handle streaming connection recovery?
- [ ] What's the upgrade path from single-instance to distributed?
- [ ] How to expose framework configuration via environment variables?

### Verification checklist

- [ ] Can run streaming chat with text output
- [ ] Tool calls are captured as events
- [ ] Memory stores data correctly
- [ ] Event broker routes to correct consumers
- [ ] ReAct loop terminates on max iterations

---

## Summary

**Goal**: Create reusable framework for event streaming LLM apps

**Status**: Analysis complete, ready for implementation

**Evidence**: Two architecture diagrams analyzed, core components identified, key decisions documented

**Delivery**: This document + architecture analysis uploaded to reMarkable
