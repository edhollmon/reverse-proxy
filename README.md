# reverse-proxy

A reverse proxy written in Go, similar in spirit to nginx. It loads a JSON config file and forwards incoming TCP and HTTP traffic to a pool of upstream hosts using a configurable load balancing strategy.

## How it works

On startup the proxy loads a config file (or falls back to the built-in default) and sets up listeners for each defined connection. Incoming requests are forwarded to one of the configured upstream hosts according to the connection's load balancing strategy.

## Config

The config file is JSON. Pass a path at startup, or omit it to use the embedded default config.

```json
{
    "connections": {
        "tcp": [
            {
                "type": "tcp",
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

### Connection fields

| Field | Description |
|---|---|
| `type` | Protocol — `tcp` or `http` |
| `lbstrategy` | Load balancing strategy — e.g. `round-robin`, `least-connections` |
| `hosts` | List of upstream hosts |
| `hosts[].host` | Upstream hostname or IP |
| `hosts[].port` | Upstream port |

## Running

```bash
go run .
```

## Local TCP load balancing walkthrough

The default config listens on port `9090` and round-robins connections across two backends on `localhost:9091` and `localhost:9092`. The steps below let you observe load balancing with nothing but `nc` (netcat) across a few terminals.

**Terminal 1 — backend A (port 9091)**
```bash
nc -lk 9091
```

**Terminal 2 — backend B (port 9092)**
```bash
nc -lk 9092
```

**Terminal 3 — start the proxy**
```bash
go run .
```

**Terminals 4, 5, … — send connections**

Each `nc` call opens one connection through the proxy. Round-robin distributes them alternately to backend A and backend B, so whatever you type in terminal 4 appears in terminal 1, terminal 5 appears in terminal 2, terminal 6 back to terminal 1, and so on.

```bash
# first connection → backend A (9091)
nc localhost 9090

# second connection → backend B (9092)
nc localhost 9090

# third connection → backend A again
nc localhost 9090
```

The proxy logs each connection with its assigned backend:

```
level=INFO msg="client proxying" cid=1 backend=localhost:9091
level=INFO msg="client proxying" cid=2 backend=localhost:9092
level=INFO msg="client proxying" cid=3 backend=localhost:9091
```

## Project structure

```
.
├── main.go
└── internal/
    └── config/
        ├── config-service.go     # Config loading and parsing
        └── default.config.json   # Embedded default config
```
