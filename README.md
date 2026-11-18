# Redis Clone in Go — Learning Project

This project is a small Redis-like server built in Go as a systems-programming learning exercise.

The goal is not to recreate Redis feature-for-feature. The goal is to understand the engineering concepts that sit underneath systems like Redis by implementing them incrementally while **learning go**.

## Why this project exists

Modern AI coding tools can generate large amounts of code very quickly.

For this project, I intentionally use AI differently.

The AI acts as a **mentor rather than a code generator**.

Instead of asking it to implement the project for me, the workflow is:

1. Introduce the next system concept.
2. Understand why that concept is needed.
3. Reason about the design and trade-offs.
4. Implement the code myself.
5. Review the implementation with the AI.
6. Write tests.
7. Fix mistakes and understand why they happened.
8. Move to the next concept.

The objective is to preserve the part of programming that involves understanding, debugging, reasoning, and making engineering decisions.

---

# Phase 1 — Building the Core Redis Clone

Phase 1 focused on building a working Redis-like server from the network layer upward.

## What was implemented

- TCP server listening for Redis clients
- Multiple concurrent client connections using goroutines
- RESP protocol parsing
- RESP response encoding
- Shared in-memory key/value store
- Thread-safe access using `sync.RWMutex`
- Lazy key expiration
- Redis-style commands:
  - `SET`
  - `GET`
  - `INCR`
  - `EXPIRE`
  - `TTL`
  - `DEL`
  - `EXISTS`
- Redis-style integer, bulk-string, simple-string, null, and error responses
- Store-level unit tests
- Command-layer unit tests
- Concurrent operation tests
- Go race detector testing
- Manual end-to-end testing over real TCP connections using `nc`

---

# What I Learned in Phase 1

## 1. TCP and Server Architecture

Before this project, it was easy to think of backend communication mainly in terms of HTTP.

Redis uses a lower-level TCP connection.

The basic flow became:

```text
Client
  ↓
TCP connection
  ↓
RESP parser
  ↓
Command execution
  ↓
In-memory store
  ↓
RESP encoder
  ↓
TCP response
```

I learned how:

- `net.Listen` creates a TCP listener
- `Accept()` waits for incoming connections
- each client receives its own `net.Conn`
- TCP is a byte stream rather than a message protocol
- TCP read boundaries do not necessarily match command boundaries

---

## 2. Goroutines and Concurrent Connections

Initially, handling a client directly after `Accept()` prevented the server from accepting another client until the first connection closed.

Using:

```go
go handleConnection(conn, store)
```

allows every connection to run independently.

This introduced important concurrency concepts:

- goroutines
- shared memory
- synchronization
- race conditions
- resource lifetime
- `defer`

---

## 3. RESP Protocol Parsing

Redis clients communicate using RESP.

For example:

```text
SET username alex
```

can arrive as:

```text
*3\r\n
$3\r\n
SET\r\n
$8\r\n
username\r\n
$4\r\n
alex\r\n
```

The parser converts the protocol representation into something the command layer can understand:

```go
[]string{"SET", "username", "alex"}
```

The RESP layer intentionally knows nothing about storage structures such as `Entry`.

That created a useful separation:

```text
protocol.go
RESP bytes → []string

command.go
[]string → command semantics

store.go
data storage and concurrency
```

---

## 4. Buffered Network Reading

A `bufio.Reader` is created once per connection.

This was important because TCP may:

- split one command across multiple reads
- combine multiple pieces of data into one read
- buffer ahead

The parser therefore reads based on RESP structure rather than assuming one TCP read equals one Redis command.

---

## 5. Go Maps and Shared State

The in-memory database started as:

```go
map[string]string
```

and later evolved into:

```go
map[string]Entry
```

where:

```go
type Entry struct {
    Value     string
    ExpiresAt time.Time
}
```

This allowed values to carry metadata such as expiration time.

---

## 6. Mutexes and Race Conditions

Go maps are not safe for concurrent reads and writes.

The Store therefore owns its synchronization:

```go
type Store struct {
    data  map[string]Entry
    mutex sync.RWMutex
}
```

An important design decision was keeping locking inside the Store rather than spreading locking responsibility throughout connection handlers.

I learned the difference between:

```go
RLock()
```

for read-only access and:

```go
Lock()
```

for operations that may mutate shared state.

Some operations that appear to be reads, such as `GET`, `TTL`, and `EXISTS`, may still require a write lock because lazy expiration can delete stale entries.

---

## 7. Atomic Operations

`INCR` demonstrated an important concurrency lesson.

This would not be safe:

```text
GET value
increment value
SET value
```

even if `GET` and `SET` were individually protected by locks.

Two goroutines could both read the same value before either writes the updated result.

Instead, the whole read-modify-write operation must happen while holding one lock.

This introduced the idea of **atomic operations**.

---

## 8. Lazy Expiration

Keys can have an `ExpiresAt` timestamp.

Instead of constantly scanning the entire database for expired keys, this implementation currently uses lazy expiration.

When an operation accesses a key:

```text
key missing
→ behave as missing

key has no expiry
→ use normally

key expiry is in the past
→ delete it
→ behave as missing

key expiry is in the future
→ use normally
```

This avoids constant background scans but means expired keys that are never accessed may temporarily remain in memory.

That trade-off was intentional for the learning implementation.

---

## 9. Redis Commands Implemented

### SET

Stores a value.

```text
SET username alex
```

### GET

Retrieves a value.

Missing values return a RESP null bulk string.

### INCR

Atomically increments an integer value.

It also demonstrated why compound operations need to be synchronized as a single unit.

### EXPIRE

Assigns a TTL to an existing key.

```text
EXPIRE session 10
```

Expired or missing keys behave as nonexistent.

### TTL

Returns remaining lifetime in seconds.

```text
key missing       → -2
key has no expiry → -1
key has expiry    → remaining seconds
```

### DEL

Deletes one or more keys.

```text
DEL a b c
```

Returns the number of live keys that were actually deleted.

### EXISTS

Checks one or more keys.

```text
EXISTS a b c
```

Returns the number of live keys that currently exist.

Expired keys behave as missing.

---

## 10. Testing

Tests were written at multiple layers.

### Store tests

These verify storage behavior independently from Redis protocol concerns.

Examples include:

- missing keys
- existing keys
- expired keys
- TTL behavior
- atomic increments
- multi-key delete
- multi-key exists

### Command-layer tests

These verify that commands correctly interpret parsed RESP values.

For example:

```text
["EXPIRE", "session", "10"]
```

must:

- parse `"10"`
- convert it into a Go duration
- call the Store
- convert the Store result into Redis response semantics

### Concurrency tests

`sync.WaitGroup` was used to run concurrent operations and wait for all goroutines to complete.

The race detector is also used:

```bash
go test -race ./...
```

This helps detect unsafe concurrent access to shared memory.

---

## 11. Manual End-to-End Testing

The server has also been tested using real TCP connections.

For example:

```bash
printf '*2\r\n$3\r\nGET\r\n$7\r\nsession\r\n' | nc localhost 6379
```

This verifies the complete path:

```text
TCP
→ RESP parser
→ command layer
→ store
→ response encoder
→ TCP
```

---

# Phase 2 — Automated Integration Testing and Server Hardening

Phase 2 focuses less on adding Redis commands and more on making the server easier to verify and more robust.

## Learning Goals

### 1. Automated TCP Integration Tests

Replace manual `nc` verification with automated tests that start a real TCP server.

Instead of using the fixed Redis port:

```text
6379
```

tests will ask the operating system for an available port:

```go
net.Listen("tcp", ":0")
```

This introduces:

- integration testing
- ephemeral ports
- test server lifecycle
- real TCP clients inside tests
- testing multiple application layers together

---

### 2. Refactoring the Server Lifecycle

The TCP accept loop will be moved out of `main()` into a reusable function.

Conceptually:

```text
main()
  ↓
create listener
create store
  ↓
serve(listener, store)
```

Tests can then call the same server implementation:

```text
test
 ↓
temporary listener
 ↓
serve(listener, store)
 ↓
real TCP client
```

This demonstrates an important design principle:

> application startup code should be thin, while reusable behavior should live in testable functions.

---

### 3. Automated End-to-End RESP Testing

Integration tests will send actual RESP payloads over TCP.

For example:

```text
SET session abc
EXPIRE session 10
TTL session
GET session
DEL session
```

The tests will verify the actual bytes returned by the server.

This ensures that:

- RESP parsing works
- commands execute correctly
- state is shared between connections
- encoding is correct
- networking works

all at the same time.

---

### 4. RESP Parser Hardening

The current parser works for valid commands.

The next stage will explore malformed input, including:

- invalid array headers
- invalid bulk-string lengths
- malformed CRLF sequences
- incomplete payloads
- unexpected protocol types
- disconnected clients during parsing

The goal is to understand defensive protocol parsing rather than assuming clients always send valid input.

---

### 5. Error Boundaries

Different failures should have different outcomes.

For example:

```text
invalid Redis command
→ send RESP error
→ keep connection alive

malformed protocol
→ protocol error
→ potentially close connection

network EOF
→ client disconnected
→ close handler
```

This will help explore how network servers distinguish application errors from transport/protocol failures.

---

### 6. Persistence

The current database exists only in memory.

If the server stops:

```text
all data disappears
```

A small persistence experiment will be added to understand how databases survive restarts.

The goal is not to fully reproduce Redis persistence.

Possible learning exercises include:

```text
snapshot
```

or a small:

```text
append-only log
```

This will introduce concepts such as:

- disk persistence
- recovery after restart
- serialization
- durability
- memory vs disk trade-offs

---

# Goals after current phase

- clustering
- replication
- Redis Sentinel
- Lua scripting
- streams
- full transactions
- pub/sub
- full RDB compatibility
- full AOF compatibility
- production-grade eviction
- every Redis data type and command

The project should stop when the major learning goals have been achieved.

---

# Philosophy

The value of this project is not the number of Redis commands implemented.

The value is being able to explain why the system works:

```text
Why TCP?
Why buffering?
Why goroutines?
Why mutexes?
Why atomic operations?
Why lazy expiration?
Why separate protocol and storage layers?
Why unit tests and integration tests?
Why persistence?
```

If those questions can be answered clearly, the project has achieved its purpose.

---

# Current Status

Phase 1: **Complete**

Implemented and tested:

```text
SET
GET
INCR
EXPIRE
TTL
DEL
EXISTS
```

along with TCP networking, RESP parsing, concurrency, expiration, and unit testing.

Phase 2: **Next**

The next milestone is extracting the TCP accept loop into a reusable server function and building the first automated end-to-end TCP integration test.
