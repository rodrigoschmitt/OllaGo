# OllaGo – AI Assistant

A local, ChatGPT-style chat interface powered by **Ollama** and built with Go + Vanilla JS.
Created by **Rodrigo Andrade**.

---

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.22+ | Backend server |
| Ollama | latest | LLM inference |
| A pulled model | e.g. `gemma3:12b` | Answering questions |

**Install Ollama:** https://ollama.com
**Pull a model:**
```bash
ollama pull gemma3:12b
```

---

## Run locally

```bash
# 1. Clone and enter the project
git clone <repo-url> ollago && cd ollago

# 2. (Optional) override defaults via env vars
export OLLAMA_URL=http://localhost:11434   # default
export ADDR=:8080                          # default

# 3. Start Ollama (if not already running)
ollama serve

# 4. Run the server
go run ./cmd/server

# 5. Open the UI
open http://localhost:8080
```

The `static/` directory is served from the working directory, so run the
`go run` command from the project root.

---

## Project structure

```
ollago/
├── cmd/
│   └── server/
│       └── main.go          # Entry point: HTTP server, graceful shutdown
├── internal/
│   └── handler/
│       ├── chat.go          # POST /api/chat → SSE streaming handler
│       └── models.go        # GET /api/models → installed model list
├── pkg/
│   └── ollama/
│       └── client.go        # Ollama API client (streaming + model listing)
├── scripts/
│   └── update_models.py     # Patches index.html with the current ollama list
├── static/
│   └── index.html           # Full frontend (HTML + CSS + JS, no build step)
├── go.mod
└── README.md
```

---

## How SSE streaming works end-to-end

```
Browser                 Go server              Ollama
  │                        │                      │
  │── POST /api/chat ──────▶│                      │
  │   {model, messages}    │── POST /api/chat ────▶│
  │                        │   {stream: true}      │
  │                        │                      │
  │                        │◀── NDJSON stream ─────│
  │                        │   {"message":{"content":"Hello"},"done":false}
  │◀── SSE: data: "Hello" ─│   {"message":{"content":" there"},"done":false}
  │◀── SSE: data: " there"─│   {"done":true}
  │◀── SSE: [DONE] ────────│
```

### Server side (`internal/handler/chat.go`)

1. Sets `Content-Type: text/event-stream` and `Cache-Control: no-cache`.
2. Launches a goroutine that calls `ollama.Client.Chat()`, which scans
   the NDJSON response line-by-line and sends each content token to a
   buffered `chan string`.
3. The handler loop reads from the channel and writes:
   ```
   data: "token"\n\n
   ```
   then calls `flusher.Flush()` to push the bytes immediately.
4. When the channel closes it sends `data: [DONE]\n\n`.
5. If the client disconnects, `r.Context().Done()` unblocks and the
   goroutine is cancelled via context — no goroutine leak.

### Browser side (`static/index.html`)

1. On load, calls `GET /api/models` to populate the model selector dynamically
   from the locally installed Ollama models — no hard-coded list.
2. Uses `fetch()` with a `ReadableStream` body reader (not `EventSource`,
   because `EventSource` only supports GET requests).
3. Accumulates chunks in a string buffer, splits on `\n\n` to extract
   complete SSE events, strips the `data:` prefix, and `JSON.parse()`s
   each token.
4. The first token replaces the typing indicator with a real bubble;
   subsequent tokens are appended directly to the bubble via the Markdown
   renderer, so bold, tables, and code blocks render live as tokens arrive.

### Image attachments

The attach button is available for all models. Only vision-capable models
(e.g. `gemma3`, `llava`, `minicpm-v`) will process an attached image — if you
send an image to a model that does not support vision, Ollama returns an error
which is displayed inline in the chat as a `⚠️` message. No hard-coded list of
supported models is maintained in the frontend; capability is determined at
inference time by Ollama itself.

---

## Syncing the model list (offline / static patching)

If you prefer to pre-bake the model list into the HTML rather than relying on
the live `/api/models` call, run the helper script:

```bash
# Preview changes without writing
python3 scripts/update_models.py --dry-run

# Apply — rewrites static/index.html in place
python3 scripts/update_models.py

# Point at a different HTML file
python3 scripts/update_models.py --html path/to/index.html
```

The script runs `ollama list`, parses the installed model names, and replaces
the `<select id="model-select">` block in `static/index.html`.
Requires Python 3.9+ and Ollama running locally.

---

## Scaling to multiple users

### Current design
Each request gets its own goroutine and a dedicated HTTP connection to
Ollama. Go's scheduler handles thousands of concurrent goroutines cheaply,
so you can serve many users on a single machine before hitting limits.

### Bottlenecks & mitigations

| Concern | Mitigation |
|---|---|
| Ollama is single-process | Run multiple Ollama instances on different ports; load-balance at the Go layer or with nginx |
| GPU VRAM is finite | Limit concurrency with a semaphore (`chan struct{}`) in the handler; queue or 429 overflow requests |
| Long-lived connections exhaust file descriptors | Increase `ulimit -n`; tune `IdleTimeout` on the Go server |
| No auth | Add an `Authorization` middleware in `main.go` before deploying beyond localhost |
| No rate limiting | Add `golang.org/x/time/rate` per-IP limiter as middleware |
| Stateless context (history lives in browser) | Move `messages[]` to a server-side session store (Redis, Postgres) keyed by session ID for multi-device support |

### Horizontal scaling

```
                 ┌──────────────────┐
   users ──────▶ │  nginx / caddy   │  (TLS termination + load balancing)
                 └──────┬───────────┘
           ┌────────────┼────────────┐
           ▼            ▼            ▼
       ollago:8080  ollago:8081  ollago:8082  (multiple Go replicas)
           │            │            │
           └────────────┴────────────┘
                        │
                 ┌──────▼───────┐
                 │  Ollama pool │  (GPU node(s))
                 └──────────────┘
```

Use `proxy_buffering off` and `proxy_read_timeout 0` in nginx to avoid
breaking SSE connections.
