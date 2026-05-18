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

## Project structure

```
.
├── main.go
└── internal/
    └── config/
        ├── config-service.go     # Config loading and parsing
        └── default.config.json   # Embedded default config
```
