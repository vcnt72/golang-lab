# Go HTTP Server Learning Roadmap

## Goal

Build a small HTTP/1.1 server from raw TCP using Go.

The purpose is not to replace `net/http`. The purpose is to understand:

- TCP connection handling

- HTTP request parsing

- buffered I/O

- concurrency

- timeouts and cancellation

- routing and middleware

- memory allocation

- benchmarking and profiling

- why `net/http` is designed the way it is

## Constraints

For the main server implementation:

- Use `net`

- Use `bufio`

- Use `io`

- Use `context`

- Use `sync`

- Do not use `net/http` to serve requests

- Use `net/http` only in tests or for comparison later

- Support HTTP/1.1 only

- Do not aim for production completeness

## Initial Scope

Support:

- `GET`

- `POST`

- request headers

- `Content-Length`

- text and JSON responses

- multiple concurrent connections

- basic keep-alive

- simple routing

- middleware

- graceful shutdown

- read and write timeouts

Ignore initially:

- HTTP/2

- HTTP/3

- TLS

- WebSockets

- proxies

- multipart forms

- compression

- trailers

- advanced chunked encoding

- every HTTP edge case

---

# Phase 1 — Raw TCP Server

## Objective

Understand how a server accepts and manages network connections.

## Tasks

- Create a TCP listener with `net.Listen`

- Accept one connection

- Read raw bytes from the connection

- Write a response back

- Close the connection

- Handle multiple clients concurrently

- Handle `io.EOF`

- Log connection open and close events

- Add graceful process shutdown

## Questions to Answer

- What does `Accept` block on?
It blocks on client connection

- What happens when a client disconnects?
the connection on the server is also closed

- Can one connection perform multiple reads?
One connection can perform multiple reads.

- Can one `Read` return only part of the sent data?
Yes one Read can only return part of the sent data, according to the filled buffer length

- How many goroutines are created per connection?
There's 1 main goroutine, 1 goroutine for signaling the shutdown and 1 goroutine per accepted connection

- What happens when the server shuts down while clients are connected?
The client is also closed, and the work is lost

## Deliverable

A TCP echo server that supports multiple simultaneous clients.

---

# Phase 2 — Inspect Raw HTTP

## Objective

See HTTP as bytes transported over TCP.

## Tasks

- Send a request using `curl`

- Print the raw request

- Identify the request line

- Identify headers

- Identify the empty line separating headers and body

- Inspect a request with a body

- Compare requests from `curl`, a browser, and a custom client

## Example Request

```http
GET /users?id=10 HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.0
Accept: */*

```

## Questions to Answer

- Where does the request line end?
I think it's the first line

- How do you know where headers end?
\r\n

- How do you know whether a body exists?
If there's any Content-Length

- Why can the request arrive across multiple TCP reads?
Because TCP actually breaks request (packet) into smaller chunk and make it ordered

- Why should parsing not depend on TCP packet boundaries?
Because there's no such boundaries. TCP is an ordered dumb pipe, it's just receive request. If we use TCP packet boundaries we never know if it's 1 request or many

## Deliverable

A server that reads and prints one complete HTTP request.

---

# Phase 3 — Request Parser

## Objective

Convert raw HTTP bytes into structured Go values.

## Suggested Types

```go
type Request struct {
    Method     string
    Target     string
    Path       string
    Query      string
    Version    string
    Headers    Header
    Body       []byte
    RemoteAddr string
}

type Header map[string][]string
```

## Tasks

- Parse the request line

- Validate the HTTP method

- Parse the request target

- Parse the HTTP version

- Parse headers

- Handle repeated headers

- Normalize or compare header names safely

- Detect the end of headers

- Read a body using `Content-Length`

- Reject malformed requests

- Reject bodies exceeding a configured limit

- Write table-driven parser tests

- Test truncated requests

- Test requests arriving in small fragments

## Questions to Answer

- Should header names be normalized?
Yes it should be normalized. The HTTP server is pretext

- Should headers use `map[string]string` or `map[string][]string`?
Because headers like Set-Cookie is valid

- What happens when `Content-Length` is invalid?
the content will be skipped
- What happens when the body is shorter than declared?
or even parsed partially. JSON Body will be error

- How should parser errors be represented?
I use panic for it. the alternative is log fatal

- Which parsing operations allocate memory?
I think all operation allocate memory ya, this is because i heavily rely on string split
  - readRequestLine allocate str split memory
  - readHeader also the same
  - readBody allocate memory based on content length

## Deliverable

```go
func ReadRequest(r *bufio.Reader) (*Request, error)
```

---

# Phase 4 — Response Writer

## Objective

Generate valid HTTP responses.

## Suggested API

```go
type ResponseWriter interface {
    Header() Header
    WriteHeader(statusCode int)
    Write(body []byte) (int, error)
}
```

## Tasks

- Write the status line

- Write response headers

- Add `Content-Length`

- Add `Content-Type`

- Write the response body

- Prevent headers from being written twice

- Default to `200 OK`

- Support common status codes

- Handle empty response bodies

- Add a JSON response helper

- Flush buffered output

## Questions to Answer

- Why must headers be written before the body?
because header can contain metadata of the body
- What should happen when `WriteHeader` is called twice?
I use map[string][]string, so mutiple WriteHeader should be appended

- When should `Content-Length` be calculated?
content length is calculated on flush

- When is streaming preferable to buffering?
streaming is preferable when you don't know how long is your content.

- Why does `net/http.ResponseWriter` use an interface?
so we can create new response writer and override it or even extend it

## Deliverable

A server that returns a response successfully readable by `curl`.

```bash
curl -v http://localhost:8080/
```

---

# Phase 5 — Handler Abstraction

## Objective

Separate networking from application logic.

## Suggested API

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}
```

## Tasks

- Define `Handler`

- Define `HandlerFunc`

- Pass parsed requests to handlers

- Return handler responses through `ResponseWriter`

- Add a default not-found handler

- Recover from handler panics

- Write handler unit tests without opening a TCP port

## Questions to Answer

- Why use an interface here?

- Why is a function adapter useful?

- How can handlers be tested independently?

- Which responsibilities belong to the connection layer?

- Which responsibilities belong to the handler?

## Deliverable

A request can move through:

```text
TCP connection
    ↓
request parser
    ↓
handler
    ↓
response writer
    ↓
TCP connection
```

---

# Phase 6 — Router

## Objective

Route requests based on method and path.

## Version 1

Use a map:

```go
type RouteKey struct {
    Method string
    Path   string
}
```

## Tasks

- Register routes

- Match exact paths

- Match HTTP methods

- Return `404 Not Found`

- Return `405 Method Not Allowed`

- Add path parameters

- Add wildcard routes

- Add route groups

- Detect conflicting routes

- Benchmark route lookup

## Progression

### Router V1

Exact map lookup:

```text
GET /users
POST /users
GET /health
```

### Router V2

Path parameters:

```text
GET /users/:id
```

### Router V3

Trie or radix-tree experiment:

```text
/users/:id
/users/:id/orders
/files/*path
```

## Questions to Answer

- Why is a map sufficient for exact routes?

- Why do frameworks use tries or radix trees?

- What happens when static and parameterized routes conflict?

- Should routes be mutable after the server starts?

- How does routing affect allocations?

## Deliverable

A router with exact paths and path parameters.

---

# Phase 7 — Middleware

## Objective

Implement composable request processing.

## Suggested API

```go
type Middleware func(Handler) Handler
```

## Tasks

- Implement middleware chaining

- Logging middleware

- Recovery middleware

- Request ID middleware

- Authentication placeholder

- Timeout middleware

- Metrics middleware

- Test middleware execution order

## Example

```go
handler := Logging(
    Recovery(
        RequestID(
            router,
        ),
    ),
)
```

## Questions to Answer

- In what order does middleware execute?

- How should middleware stop a request?

- Where should panic recovery happen?

- Should logging happen before or after the handler?

- How should shared request data be stored?

## Deliverable

A middleware chain with predictable execution order.

---

# Phase 8 — Connection Lifecycle

## Objective

Understand persistent connections and resource limits.

## Tasks

- Support one request per connection

- Add HTTP keep-alive

- Read multiple requests from one connection

- Respect `Connection: close`

- Add idle timeout

- Add maximum requests per connection

- Handle client disconnects

- Prevent goroutine leaks

- Track active connections

## Questions to Answer

- Why does HTTP/1.1 use persistent connections by default?

- How does the server know when to close a connection?

- What happens when a client stays connected but sends nothing?

- Can requests be pipelined?

- What resources does every idle connection consume?

## Deliverable

A connection can process multiple sequential HTTP requests safely.

---

# Phase 9 — Timeouts and Cancellation

## Objective

Prevent slow or broken clients from consuming resources forever.

## Tasks

- Set read deadlines

- Set write deadlines

- Add idle timeout

- Add maximum header-read duration

- Add handler timeout

- Attach `context.Context` to requests

- Cancel the context when the client disconnects

- Cancel the context during server shutdown

- Verify timeout behavior with tests

## Questions to Answer

- What is the difference between a timeout and a deadline?

- What is a slowloris attack?

- Which timeout protects which server resource?

- How does cancellation propagate?

- What happens when a handler ignores its context?

## Deliverable

Connections and handlers terminate predictably under timeout conditions.

---

# Phase 10 — Graceful Shutdown

## Objective

Stop accepting new work while allowing existing work to finish.

## Tasks

- Stop accepting new connections

- Track active connections

- Wait for active requests

- Cancel requests after a shutdown deadline

- Close idle connections

- Prevent new keep-alive requests

- Test shutdown during an active request

- Test forced shutdown after timeout

## Questions to Answer

- What does graceful shutdown guarantee?

- What happens to requests currently being processed?

- How should idle and active connections be treated differently?

- When should cancellation be forced?

## Deliverable

```go
func (s *Server) Shutdown(ctx context.Context) error
```

---

# Phase 11 — Testing

## Objective

Make protocol and concurrency behavior reproducible.

## Test Categories

### Unit Tests

- Request-line parser

- Header parser

- Body reader

- Response writer

- Router

- Middleware

### Integration Tests

- Start a real TCP server

- Send raw HTTP requests

- Read and validate responses

- Test malformed requests

- Test multiple requests per connection

- Test client disconnects

- Test graceful shutdown

### Concurrency Tests

- Run many clients concurrently

- Run tests with the race detector

- Detect leaked goroutines

- Test shutdown under load

## Commands

```bash
go test ./...
go test -race ./...
go test -count=100 ./...
```

---

# Phase 12 — Benchmarking and Profiling

## Objective

Measure behavior instead of guessing.

## Tasks

- Benchmark request parsing

- Benchmark response writing

- Benchmark routing

- Measure allocations

- Benchmark persistent connections

- Create a simple load generator

- Capture CPU profiles

- Capture heap profiles

- Capture mutex profiles

- Inspect goroutine profiles

- Inspect escape analysis output

## Commands

```bash
go test -bench=. -benchmem ./...
go test -run=^$ -bench=BenchmarkReadRequest -cpuprofile=cpu.out
go tool pprof cpu.out
go test -gcflags="-m=2" ./...
```

## Experiments

- `bufio.Reader` versus direct reads

- Different buffer sizes

- `strings.Split` versus manual parsing

- Map router versus trie router

- Allocating response buffers versus reusing them

- One goroutine per connection versus bounded concurrency

- Copying request bodies versus streaming them

## Experiment Template

```markdown
## Experiment

### Hypothesis

### Implementation

### Benchmark Setup

### Results

### Interpretation

### Trade-offs
```

---

# Phase 13 — Compare with `net/http`

## Objective

Understand design decisions in the standard library.

Do this only after implementing the corresponding feature yourself.

## Areas to Read

- `http.Server`

- `Server.Serve`

- connection handling

- request reading

- `ResponseWriter`

- `Handler`

- `HandlerFunc`

- `ServeMux`

- shutdown behavior

- timeout handling

## Comparison Questions

For every feature, write:

- How did I implement it?

- How does `net/http` implement it?

- What edge case did I miss?

- What abstraction does `net/http` introduce?

- What performance concern does it handle?

- What complexity would be unnecessary in my version?

Do not attempt to read the whole package sequentially.

---

# Optional Advanced Features

Add these only after the core implementation is stable.

## Chunked Transfer Encoding

- Parse chunked request bodies

- Write chunked responses

- Handle terminating chunks

- Handle malformed chunks

## Streaming

- Stream a large response

- Flush partial response data

- Handle client disconnects during streaming

## Static Files

- Serve files

- Prevent path traversal

- Add content types

- Support range requests as an experiment

## TLS

- Wrap the listener with TLS

- Load a local certificate

- Understand where TLS ends and HTTP begins

## Server Limits

- Maximum header size

- Maximum body size

- Maximum concurrent connections

- Per-IP connection limit

- Request rate limit

---

# Suggested Repository Structure

```text
go-http-lab/
├── cmd/
│   ├── server/
│   │   └── main.go
│   ├── client/
│   │   └── main.go
│   └── loadgen/
│       └── main.go
├── internal/
│   ├── server/
│   ├── protocol/
│   ├── router/
│   └── middleware/
├── experiments/
├── docs/
│   ├── architecture.md
│   ├── protocol.md
│   ├── decisions.md
│   └── experiments/
├── go.mod
└── README.md
```

Start with fewer packages. Extract packages only when the boundaries become clear.

A simpler initial structure is enough:

```text
go-http-lab/
├── main.go
├── server.go
├── request.go
├── response.go
├── router.go
└── server_test.go
```

---

# Eight-Week Schedule

## Week 1 — TCP Foundations

- TCP listener

- connection handling

- echo server

- multiple clients

- graceful process signal handling

## Week 2 — HTTP Request Parsing

- request line

- headers

- `Content-Length`

- malformed request handling

- parser tests

## Week 3 — HTTP Responses

- status line

- headers

- body

- `ResponseWriter`

- handler abstraction

## Week 4 — Routing and Middleware

- exact route matching

- method matching

- path parameters

- logging middleware

- recovery middleware

## Week 5 — Connection Management

- keep-alive

- multiple requests per connection

- idle timeout

- read and write deadlines

- connection tracking

## Week 6 — Context and Shutdown

- request context

- client disconnect cancellation

- handler timeout

- graceful shutdown

- shutdown integration tests

## Week 7 — Performance

- parser benchmarks

- router benchmarks

- allocation analysis

- CPU and heap profiling

- load generator

## Week 8 — Standard Library Comparison

- inspect relevant `net/http` code

- document differences

- fix important design mistakes

- write final architecture notes

- publish benchmark results

---

# Weekly Time Budget

For a full-time worker:

|Session|Duration|Activity|
|---|--:|---|
|Two weekdays|45–60 minutes|Implementation|
|One weekday|30–45 minutes|Reading and tests|
|Saturday|2–3 hours|Main milestone|
|Sunday|1–2 hours|Refactoring, benchmarking, notes|

Target approximately five to seven focused hours per week.

---

# Session Template

Use this structure for each study session:

```markdown
## Session Goal

One concrete result for today.

## Work Completed

- 

## What I Learned

- 

## Questions

- 


## Next Small Step
- 
```

The next step should be small enough to start without planning.

---

# Definition of Done

The project is successful when you can explain and demonstrate:

- why TCP reads may be partial

- how HTTP requests are framed

- how request bodies are delimited

- how keep-alive works

- why servers need multiple timeout types

- how cancellation reaches a request handler

- how middleware composition works

- how graceful shutdown handles active connections

- where allocations occur

- how routing choices affect performance

- how your implementation differs from `net/http`

The project does not need to support every valid HTTP request.

The real output is not the server itself. It is the ability to reason about the server's behavior, limitations, performance, and failure modes.
