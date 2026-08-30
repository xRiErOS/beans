# Web UI and API

This page documents what `beans-serve` starts, exactly which HTTP routes it exposes, and its network exposure and authentication posture. Read the security posture section before starting it anywhere other than a trusted local machine.

## Starting the server

`beans-serve` is a [separate binary](commands/separate-binaries.md) from `beans`; running it with no arguments (or any flag as the first argument) is equivalent to `beans-serve serve`:

```
$ beans-serve --help
Start the web server

Usage:
  beans-serve serve [flags]

Aliases:
  serve, server, s

Flags:
      --cors-origin strings   Allowed CORS origins (use * to allow all) (default [http://localhost:*,http://127.0.0.1:*])
  -h, --help                  help for serve
  -p, --port int              Port to listen on (default 8080)

Global Flags:
      --beans-path string   Path to data directory (overrides config and BEANS_PATH env var)
      --config string       Path to config file (default: searches upward for .beans.yml)
```

The port and the allowed CORS origins each resolve in the same order: the `--port`/`--cors-origin` flag if given, else `server.port`/`server.cors_origins` from `.beans.yml`, else the built-in default (port `8080`; origins `http://localhost:*` and `http://127.0.0.1:*`). See [Configuration Reference](configuration.md) for the `.beans.yml` keys.

At startup the server starts a filesystem watcher on the beans store (for GraphQL subscriptions), creates the worktree root directory if missing (default `~/.beans/worktrees/<project>/`), restores any previously allocated workspace ports, and logs the port and configured origins before it starts listening. Shutdown is graceful on `SIGINT`/`SIGTERM`: in-flight requests get up to 5 seconds to finish before the listener force-closes, and the whole process force-exits after a 10-second hard deadline if something (an unresponsive subprocess, a stuck WebSocket handler) blocks shutdown from completing.

## Network binding

The HTTP server binds to `:<port>` — an empty host, which means every network interface, not just loopback. There is no `--host`, `--bind`, or `--listen-address` flag to restrict it to `127.0.0.1`. Starting `beans-serve` on a machine with a routable network interface makes it reachable from that network, not just from `localhost`, regardless of the CORS origin allowlist described below: CORS governs which *browser* origins may read cross-origin responses, and does not prevent a non-browser client (`curl`, another service, a script) elsewhere on the network from reaching the port directly.

## Routes

`beans-serve` exposes exactly five route groups, mounted on a single Gin router:

| Method | Path | Purpose |
|---|---|---|
| any | `/api/graphql` | The GraphQL API — queries, mutations, and WebSocket subscriptions, all on one endpoint |
| `GET` | `/playground` | The GraphQL Playground, an interactive query IDE pointed at `/api/graphql` |
| `GET` | `/api/attachments/:beanId/:filename` | Serves an image attached to an agent chat message |
| `GET` | `/api/terminal` | WebSocket endpoint that bridges a browser terminal to a PTY session, in the central project directory or a worktree |
| any unmatched path | (fallback) | Serves the embedded frontend single-page app, with client-side routing for unmatched non-asset paths |

`/api/graphql` accepts `GET`, `POST`, and WebSocket upgrade (subprotocol `graphql-transport-ws`, 10-second keepalive ping) on the same path, so it doubles as both the query/mutation endpoint and the subscription endpoint. There is no separate `/api/graphql/subscriptions` path.

The frontend fallback route serves the same Svelte-based single-page app that ships with the web UI (including its embedded `index.html` and the `planning`/`planning/board` views); a path that doesn't match an embedded file and doesn't contain a `.` is treated as client-side-routed and served `index.html`, while a path that looks like a missing asset (contains a `.`) returns a plain 404.

## Request handling and limits

Every request passes through, in order: Gin's request logger, a panic-recovery middleware, a request-body size cap (10 MB — GraphQL documents and bean bodies are far smaller; agent chat image attachments are the largest legitimate payload), a shared concurrent-WebSocket-connection cap (100, shared across `/api/graphql` subscriptions and `/api/terminal`, since both compete for the same process's file descriptors), and the CORS middleware. The underlying `net/http.Server` enforces a 15-second read timeout, a 15-second write timeout, and a 60-second idle timeout on top of those.

## CORS and origin checking

The CORS middleware reflects `Access-Control-Allow-Origin` back only for an origin that matches the configured allowlist; an unmatched `Origin` header gets no CORS header at all, which browsers treat as a same-origin-only response. The allowlist accepts three pattern shapes:

- `*` — allow every origin (the middleware then always answers with `Access-Control-Allow-Origin: *`)
- an exact origin, e.g. `http://localhost:5173`
- an origin with a trailing `:*` port wildcard, e.g. `http://localhost:*`, matching any port on that host

The same allowlist and the same `Checker` gate both the CORS middleware and every WebSocket upgrade (`/api/graphql` subscriptions and `/api/terminal`), via `CheckOrigin` on the `gorilla/websocket` upgraders — a WebSocket handshake from a disallowed origin is rejected before the connection is established.

## Security posture

There is no authentication or authorization anywhere in `beans-serve`: no login, no API key, no bearer token, no session cookie. Every route — the GraphQL API (queries and mutations, including workspace creation, terminal commands run through the agent tooling, and file changes), the Playground, and the PTY-backed terminal WebSocket — is reachable by anyone who can reach the port and satisfy the CORS/origin check above. The origin check is a browser-focused CSRF-style guard, not an authentication mechanism: it does nothing against a direct, non-browser client that can already reach the port on the network.

The GraphQL Playground at `/playground` is not gated separately from the API itself — anyone who can load it can execute any query or mutation the schema exposes, with full read/write access to the bean store, worktrees, and terminal sessions. The terminal WebSocket at `/api/terminal` grants an interactive shell in the project's own working directory or one of its worktrees; nothing in the request path checks who is opening that shell beyond the CORS/origin allowlist. There is no separate "read-only" or "public" mode.

Given this, `beans-serve` is meant to be run on a trusted local machine, bound to a network the operator controls, with the default `localhost`-only CORS origins left in place. Widening `--cors-origin` to `*` or to a non-localhost origin, or exposing the port beyond loopback (a shared network, a container port mapping, a reverse proxy without its own authentication layer), extends every one of the capabilities above to whoever can then reach it.

## Related documentation

- [Separate Binaries](commands/separate-binaries.md)
- [Configuration Reference](configuration.md)
- [Querying and Automation](commands/querying-and-automation.md)
- [TUI Companion](tui-companion.md)
- [Troubleshooting](troubleshooting.md)
