# Redis Clone in Go — Learning Project

This project is a small Redis-like server built in Go as a systems-programming learning exercise.

The goal is not to recreate Redis feature-for-feature. The goal is to understand the engineering concepts that sit underneath systems like Redis by implementing them incrementally while **learning Go**.

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

The command layer converts the relative duration into an absolute expiration timestamp before passing it to the Store.

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
- convert it into an absolute expiration time
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

# Phase 2 — Integration Testing, Server Hardening, and Persistence

Phase 2 focused on moving beyond the basic in-memory Redis clone and exploring how a networked system can be tested, hardened, persisted, and recovered after restart.

## What was implemented

- Reusable TCP server lifecycle through `serve(listener, store, persister)`
- Automated TCP integration tests using ephemeral ports
- Shared state verification across multiple TCP connections
- Concurrent TCP command testing
- RESP parser hardening
- Protocol-error versus command-error handling
- TCP half-close testing for incomplete payloads
- Append-only persistence using RESP serialization
- File-backed `Persister` abstraction
- Write-ahead persistence before in-memory mutation
- Durability using `File.Sync()`
- Absolute expiration persistence using `EXPIREAT`
- AOF replay during server startup
- Recovery of `SET`, `INCR`, `DEL`, and expiration state
- Concurrent AOF write protection using a dedicated mutex
- Concurrent persistence and replay testing
- Truncated and corrupted AOF detection
- Startup recovery from an existing append-only log

---

# What I Learned in Phase 2

## 1. Automated TCP Integration Testing

Phase 1 relied partly on manual testing using `nc`.

Phase 2 introduced automated tests that start the real TCP server and communicate with it through real TCP connections.

Tests use:

```go
net.Listen("tcp", ":0")
```

Using port `0` asks the operating system to allocate an available port automatically.

This avoids hard-coding port `6379` in tests and allows integration tests to run independently.

The integration tests verify the complete path:

```text
TCP client
    ↓
RESP parser
    ↓
command execution
    ↓
shared Store
    ↓
RESP encoder
    ↓
TCP response
```

This helped expose bugs that unit tests could not detect.

One example was an incorrectly encoded RESP command:

```text
$3
INCR
```

even though `INCR` is four bytes.

The Store-level `INCR` tests passed, but the TCP integration test failed because the actual protocol payload was malformed.

This demonstrated why unit tests and integration tests solve different problems.

---

## 2. Server Lifecycle and Testability

The TCP accept loop was extracted from `main()` into a reusable server function.

Conceptually:

```text
main
 ↓
create Store
create Persister
create Listener
 ↓
serve(listener, store, persister)
```

This allowed tests to start the same server implementation used by the real application.

Another important lesson appeared when closing a test listener.

Originally an `Accept()` error caused the server loop to continue.

When the listener was closed, this caused repeated errors such as:

```text
use of closed network connection
```

The correct behavior was to return from the server loop when the listener can no longer accept connections.

This reinforced the idea that resource shutdown is part of server lifecycle design.

---

## 3. Protocol Errors vs Command Errors

The server now distinguishes between errors at different layers.

A valid RESP message containing an invalid Redis command should not destroy the TCP connection.

For example:

```text
GET foo extra
```

is valid RESP but invalid command semantics.

The server responds with a Redis error and keeps the connection alive:

```text
valid RESP
→ command validation fails
→ send -ERR
→ continue reading commands
```

A malformed or incomplete RESP payload is different:

```text
malformed RESP
→ parser fails
→ connection handler returns
→ TCP connection closes
```

This introduced the idea of **error boundaries**.

Different layers own different failures:

```text
protocol error
→ protocol / connection layer

command error
→ application command layer

network EOF
→ connection lifecycle
```

---

## 4. TCP Half-Close

Testing incomplete RESP payloads introduced the TCP concept of a half-close.

Calling:

```go
tcpConn.CloseWrite()
```

tells the server:

```text
the client will send no more bytes
```

while still allowing the client to read from the connection.

This is different from:

```go
conn.Close()
```

which closes both directions.

This was necessary when testing incomplete requests because otherwise the server could continue waiting for bytes that might still arrive.

It also introduced Go type assertions:

```go
tcpConn := conn.(*net.TCPConn)
```

because `net.Conn` exposes general connection behavior while `*net.TCPConn` exposes TCP-specific operations such as `CloseWrite()`.

---

## 5. RESP Parser Hardening

The RESP parser was expanded beyond valid-input testing.

Tests now cover cases such as:

- invalid array prefixes
- non-numeric array lengths
- non-numeric bulk-string lengths
- invalid CRLF terminators
- incomplete payloads
- unexpected EOF during parsing

The parser validates the trailing RESP delimiter rather than merely consuming two bytes.

Conceptually:

```text
read payload
→ read next two bytes
→ verify they are exactly "\r\n"
```

This reinforced the idea that protocol parsers should validate structure explicitly instead of trusting clients.

---

## 6. Persistence Architecture

The original Store was entirely in memory.

Phase 2 introduced a small append-only persistence model.

The goal was not to reproduce Redis AOF exactly.

The goal was to understand:

- durability
- write-ahead logging
- serialization
- recovery
- filesystem APIs
- disk versus memory trade-offs

A persistence abstraction was introduced:

```go
type Persister interface {
    Append(command []string) error
}
```

The command layer depends on this capability rather than directly depending on files.

The concrete implementation is a `FilePersister`.

This demonstrated a simple form of dependency inversion:

```text
command execution
    ↓
Persister interface
    ↓
FilePersister
```

The actual concrete dependency is constructed at application startup and passed downward.

Unlike NestJS dependency injection, there is no DI container performing the wiring automatically.

In this project, `main()` acts as the composition root.

---

## 7. File Lifecycle and Resource Ownership

`FilePersister` keeps the append-only file open for the lifetime of the server.

The trade-off considered was:

```text
open/close for every command
→ simpler resource lifetime
→ additional filesystem overhead

keep one file open
→ faster repeated writes
→ consumes one long-lived file descriptor
```

Because persistence happens frequently and there is only one AOF file, keeping the file open was the appropriate choice.

The component that opens the file also owns closing it:

```text
NewFilePersister
→ open file

Append
→ use file

Close
→ release file
```

This reinforced the resource ownership principle.

---

## 8. RESP as the Persistence Format

Persisted commands are serialized using RESP.

For example:

```go
[]string{"SET", "foo", "bar"}
```

is encoded as:

```text
*3\r\n
$3\r\n
SET\r\n
$3\r\n
foo\r\n
$3\r\n
bar\r\n
```

This allowed the existing RESP parser to be reused during recovery.

The persistence layer therefore does not need a completely separate command parser.

The flow becomes:

```text
command []string
→ RESP encoder
→ append-only file

restart

append-only file
→ RESP parser
→ command []string
→ Store
```

This demonstrated the value of choosing a serialization format that existing components can already understand.

---

## 9. Write-Ahead Persistence

Mutating commands are persisted before their in-memory state is changed.

Conceptually:

```text
validate command
→ append to log
→ ensure persistence succeeded
→ mutate Store
→ return response
```

This avoids the situation where memory contains a successful mutation that was never recorded on disk.

For `SET`, the ordering is straightforward:

```text
validate
→ persist
→ store.Set
```

`INCR` was more complicated because validation and mutation originally happened atomically inside the Store.

The Store therefore accepts a `beforeCommit` callback.

Conceptually:

```text
lock Store
→ inspect and validate current value
→ calculate next value
→ beforeCommit()
    → persist command
→ mutate Store
→ unlock
```

This preserves atomicity while ensuring persistence occurs before the actual mutation.

---

## 10. Callbacks and Shared Command Execution

Introducing persistence and replay created a design problem.

Normal command execution must persist mutations:

```text
client command
→ persist
→ mutate
```

Replay must execute the same command logic without persisting the command again:

```text
AOF command
→ mutate only
```

Duplicating the entire command implementation would create two versions of the same behavior.

The command logic was therefore extracted from `executeCommand()` into a shared `runCommand()` function:

```text
executeCommand ─┐
                ├→ runCommand
replayCommand ──┘
```

`executeCommand()` supplies a callback that persists mutations.

`replayCommand()` supplies a no-op callback.

Conceptually:

```text
executeCommand
→ runCommand with persistence callback

replayCommand
→ runCommand with no-op callback
```

This was an important lesson in separating:

```text
what a command does
```

from:

```text
what policy should happen before mutation
```

---

## 11. Durability and `Sync()`

Writing data to a file does not necessarily mean the bytes have reached durable storage.

There are roughly several stages:

```text
Go application
→ operating-system buffers
→ storage device
```

`FilePersister.Append()` performs:

```text
encode
→ Write
→ verify full write
→ Sync
→ return success
```

Calling:

```go
file.Sync()
```

asks the operating system to flush file data toward durable storage.

For this learning implementation, every persisted mutation is synced immediately.

This favors simpler durability semantics over performance.

A more advanced system could use policies such as:

```text
sync every write
sync periodically
batch multiple writes
```

This demonstrated the durability-versus-performance trade-off found in real databases.

---

## 12. Persisting Expiration Correctly

`EXPIRE` introduced an important persistence problem.

Suppose the command is:

```text
EXPIRE foo 10
```

Persisting that exact command would be incorrect.

If the server restarted five seconds later and replayed:

```text
EXPIRE foo 10
```

the key would incorrectly receive a fresh ten-second TTL.

Instead, the relative expiration is converted into an absolute timestamp.

Normal execution:

```text
EXPIRE foo 10
→ calculate absolute expiry once
→ persist EXPIREAT foo <unix timestamp>
→ Store.ExpireAt(foo, absoluteTime)
```

Replay:

```text
EXPIREAT foo <unix timestamp>
→ restore the original absolute expiry
```

If that timestamp is already in the past when replay occurs, the Store removes the key.

This preserves the original expiration semantics across restarts.

---

## 13. Absolute Time in the Store

The Store expiration API was changed from accepting a duration:

```text
expire this key 20 seconds from now
```

to accepting an absolute timestamp:

```text
expire this key at this exact time
```

The command layer is responsible for converting relative Redis commands into an absolute time.

The Store only understands:

```go
ExpireAt(key, expiresAt)
```

This keeps relative command semantics outside the storage layer.

It also avoids calculating the expiration time independently in the persistence layer and Store, which could produce slightly different timestamps.

---

## 14. AOF Replay

On startup, the server creates a fresh Store and replays the append-only file before accepting clients.

The startup flow is:

```text
create empty Store
→ replay existing AOF
→ open FilePersister for new writes
→ create TCP listener
→ serve clients
```

`replayAOF()`:

```text
opens AOF
→ creates bufio.Reader
→ repeatedly calls parseCommand()
→ replayCommand()
→ runCommand()
→ rebuilds Store
```

A clean EOF between commands means replay completed normally.

An EOF in the middle of a RESP command means the AOF is truncated or corrupted and replay returns an error.

---

## 15. First Startup and Missing Persistence Files

On the first server startup, `appendonly.aof` may not exist.

Replay treats this differently from real recovery errors.

The startup layer checks:

```go
errors.Is(err, os.ErrNotExist)
```

A missing file means:

```text
there is simply nothing to recover yet
```

The `FilePersister` then creates the AOF automatically using:

```go
os.O_CREATE
```

This reinforced the distinction between:

```text
expected absence
```

and:

```text
unexpected failure
```

during application startup.

---

## 16. Concurrent Persistence

Every TCP connection runs in its own goroutine.

That means multiple clients can attempt persistence concurrently:

```text
client A → Append()
client B → Append()
client C → Append()
```

`FilePersister` therefore owns its own mutex:

```go
type FilePersister struct {
    file  *os.File
    mutex sync.Mutex
}
```

The persistence critical section is serialized:

```text
lock
→ write complete RESP command
→ verify write
→ sync
→ unlock
```

The RESP encoding itself can happen before acquiring the file mutex because it only works with local data.

The Store mutex and Persister mutex protect different resources:

```text
Store mutex
→ protects in-memory state

Persister mutex
→ protects AOF writes
```

---

## 17. Multiple Locks and Deadlock Reasoning

`INCR` currently holds the Store mutex while performing its `beforeCommit` persistence callback.

That creates this lock ordering:

```text
Store lock
→ Persister lock
```

This is safe as long as another code path never holds the Persister lock while attempting to acquire the Store lock.

Otherwise two goroutines could form a cycle:

```text
goroutine A
Store lock
→ waiting for Persister lock

goroutine B
Persister lock
→ waiting for Store lock
```

That would deadlock.

The current implementation avoids the reverse lock order.

This introduced the broader concurrency principle that when a system has multiple locks, their acquisition order becomes part of the system design.

---

## 18. Concurrent Persistence Testing

A concurrent persistence test creates many goroutines that append different `SET` commands to the same AOF.

After all writers finish:

```text
AOF
→ replayAOF
→ fresh Store
```

The test verifies that every expected key and value can be reconstructed.

This validates more than simply checking whether `Append()` returned an error.

It verifies that:

```text
concurrent writes
→ remain complete RESP commands
→ file remains parseable
→ recovery remains correct
```

The Go race detector is still used separately:

```bash
go test -race ./...
```

The race detector checks unsafe memory access, while the replay test checks logical persistence correctness.

---

## 19. Corrupted and Truncated AOF Recovery

Recovery behavior was tested using intentionally incomplete RESP payloads.

For example, an AOF may claim a bulk string has five bytes but contain only two before the file ends.

The recovery policy for this project is:

```text
clean EOF between commands
→ replay complete

EOF halfway through command
→ corruption / truncation
→ return error
→ startup should fail
```

A second test verifies an AOF containing:

```text
valid command
truncated command
```

All complete commands before the corruption point are applied.

The incomplete command is not applied, and replay returns an error.

The server does not silently ignore corrupted persistence data and continue with potentially incomplete state.

---

# Phase 2 Testing

Phase 2 added tests across several boundaries.

## TCP integration tests

Tests verify:

- real TCP connections
- `SET` over TCP
- `SET` followed by `GET` on the same connection
- state shared across multiple connections
- concurrent `INCR` from multiple TCP clients
- command errors that preserve the connection
- malformed RESP that closes the connection

## Protocol tests

Tests verify:

- invalid RESP array headers
- invalid numeric lengths
- malformed CRLF
- incomplete bulk strings
- unexpected EOF

## Persistence tests

Tests verify:

- RESP command encoding
- commands are physically written to the AOF
- multiple commands can be appended
- files are properly opened and closed
- concurrent append operations

## Replay tests

Tests verify:

- `SET` restoration
- `INCR` restoration
- `DEL` restoration
- `EXPIREAT` restoration
- TTL restoration using absolute expiry
- concurrent persistence followed by replay
- truncated AOF detection
- corruption after valid commands

---

# Current Architecture

The current client request path is approximately:

```text
TCP client
    ↓
handleConnection
    ↓
RESP parser
    ↓
executeCommand
    ↓
runCommand
    ↓
beforeMutation
    ↓
FilePersister
    ↓
Store
    ↓
CommandResult
    ↓
RESP encoder
    ↓
TCP client
```

Recovery uses the same shared command semantics:

```text
appendonly.aof
    ↓
RESP parser
    ↓
replayCommand
    ↓
runCommand
    ↓
no-op beforeMutation
    ↓
Store
```

This allows normal execution and recovery to reuse the same command implementation without duplicating command logic.

---

# Important Trade-offs in the Current Design

This is intentionally a learning implementation rather than a production Redis replacement.

Several decisions favor clarity and correctness over maximum throughput.

## Sync on every mutation

Every AOF append calls `Sync()`.

This provides simple durability semantics but increases disk I/O.

## Store lock held during INCR persistence

`INCR` holds the Store mutex while its persistence callback performs file I/O.

This preserves atomicity but can temporarily block unrelated Store operations.

A higher-performance database would use a more sophisticated WAL and transaction architecture.

## Lazy expiration

Expired keys may remain physically present until accessed.

## Fail hard on corrupted AOF

The server does not currently attempt automatic truncation or repair of corrupted persistence files.

These trade-offs are intentional because the project is focused on understanding the underlying engineering concepts.

---

# Goals Outside the Scope of This Project

The following are intentionally not goals for this learning project:

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
Why write before mutating memory?
Why absolute expiry timestamps?
Why replay?
Why separate Store and Persister locks?
Why does lock ordering matter?
```

If those questions can be answered clearly, the project has achieved its purpose.

---

# Current Status

## Phase 1: Complete

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

along with:

- TCP networking
- RESP parsing
- concurrency
- lazy expiration
- unit testing
- race detection
- manual TCP testing

## Phase 2: Complete

Implemented and tested:

```text
automated TCP integration tests
server lifecycle refactoring
RESP parser hardening
protocol vs command error boundaries
TCP half-close behavior
append-only persistence
RESP command serialization
write-ahead mutation ordering
file Sync durability
absolute expiration persistence
AOF replay
restart recovery
concurrent AOF writes
lock-order reasoning
truncated/corrupted AOF detection
```

The Redis clone now supports persistence across restarts and can reconstruct its in-memory Store from the append-only log.

Final validation:

```bash
go test ./...
go test -race ./...
```

The remaining work is primarily a final architecture and learning review rather than adding major Redis functionality.

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
