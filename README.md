# reverse-proxy

A reverse proxy written in Go, similar in spirit to nginx. It loads a JSON config file and forwards incoming TCP and HTTP traffic to a pool of upstream hosts using a configurable load balancing strategy.

## How it works

On startup the proxy loads a config file (or falls back to the built-in default) and sets up listeners for each defined connection. Incoming requests are forwarded to one of the configured upstream hosts according to the connection's load balancing strategy.

## Config

The config file is JSON. Pass a path at startup, or omit it to use the embedded default config.

```json
{
    "metrics": {
        "port": "9090",
        "enabled": true
    },
    "connections": {
        "tcp": [
            {
                "type": "tcp",
                "port": "9095",
                "lbstrategy": "round-robin",
                "hosts": [
                    { "host": "10.0.0.1", "port": "9000" },
                    { "host": "10.0.0.2", "port": "9000" }
                ]
            }
        ],
        "http": [
            {
                "type": "http",
                "prefix": "/api",
                "lbstrategy": "least-connections",
                "hosts": [
                    { "host": "10.0.1.1", "port": "8080" },
                    { "host": "10.0.1.2", "port": "8080" }
                ]
            }
        ]
    }
}
```

### Metrics fields

| Field | Default | Description |
|---|---|---|
| `metrics.port` | `9090` | Port the `/metrics` endpoint listens on |
| `metrics.enabled` | `true` | Set to `false` to disable the metrics server entirely |

### Connection fields

| Field | Description |
|---|---|
| `type` | Protocol — `tcp` or `http` |
| `port` | Port to listen on (TCP only) |
| `prefix` | URL prefix to match (HTTP only) |
| `lbstrategy` | Load balancing strategy — `round-robin` or `least-connections` |
| `hosts` | List of upstream hosts |
| `hosts[].host` | Upstream hostname or IP |
| `hosts[].port` | Upstream port |

## Running

```bash
# use the built-in default config
go run .

# use a custom config file
go run . -config path/to/config.json
```

## Metrics

The proxy exposes a Prometheus metrics endpoint at `GET /metrics` (default port `9090`).

| Metric | Type | Labels | Description |
|---|---|---|---|
| `reverse_proxy_http_requests_total` | Counter | `prefix`, `code` | Total HTTP requests proxied |
| `reverse_proxy_http_request_duration_seconds` | Histogram | `prefix` | HTTP request latency |
| `reverse_proxy_tcp_connections_total` | Counter | `listen_addr` | Total TCP connections accepted |
| `reverse_proxy_tcp_active_connections` | Gauge | `listen_addr` | Currently open TCP connections |

```bash
curl http://localhost:9090/metrics
```

To disable the metrics server, set `"enabled": false` in the `metrics` config block.

## Local TCP load balancing walkthrough

The default config listens on port `9095` and round-robins connections across two backends on `localhost:9096` and `localhost:9097`. The steps below let you observe load balancing with nothing but `nc` (netcat) across a few terminals.

**Terminal 1 — backend A (port 9096)**
```bash
nc -lk 9096
```

**Terminal 2 — backend B (port 9097)**
```bash
nc -lk 9097
```

**Terminal 3 — start the proxy**
```bash
go run .
```

**Terminals 4, 5, … — send connections**

Each `nc` call opens one connection through the proxy. Round-robin distributes them alternately to backend A and backend B, so whatever you type in terminal 4 appears in terminal 1, terminal 5 appears in terminal 2, terminal 6 back to terminal 1, and so on.

```bash
# first connection → backend A (9096)
nc localhost 9095

# second connection → backend B (9097)
nc localhost 9095

# third connection → backend A again
nc localhost 9095
```

The proxy logs each connection with its assigned backend:

```
level=INFO msg="client proxying" cid=1 backend=localhost:9096
level=INFO msg="client proxying" cid=2 backend=localhost:9097
level=INFO msg="client proxying" cid=3 backend=localhost:9096
```

## Project structure

```
.
├── main.go
└── internal/
    ├── config/
    │   ├── config-service.go     # Config loading and parsing
    │   └── default.config.json   # Embedded default config
    └── server/
        ├── reverse-proxy.go      # Server lifecycle (start, shutdown)
        ├── http-router.go        # HTTP routing and load balancing
        ├── tcp-router.go         # TCP routing and load balancing
        └── metrics.go            # Prometheus metrics and /metrics server
```
