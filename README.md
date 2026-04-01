# Stockyard Paddock

**Uptime monitor.** Track endpoint health, get alerts, share a public status page. Single binary, no external dependencies.

Part of the [Stockyard](https://stockyard.dev) suite of self-hosted developer tools.

## Quick Start

```bash
# Download and run
curl -sfL https://stockyard.dev/install/paddock | sh
paddock

# Or with Docker
docker run -p 8820:8820 -v paddock-data:/data ghcr.io/stockyard-dev/stockyard-paddock:latest
```

Dashboard at [http://localhost:8820/ui](http://localhost:8820/ui)
Public status page at [http://localhost:8820/status](http://localhost:8820/status)

## Usage

```bash
# Add a monitor
curl -X POST http://localhost:8820/api/monitors \
  -H "Content-Type: application/json" \
  -d '{"name":"My API","url":"https://api.example.com/health","interval_seconds":300}'

# List monitors with current status
curl http://localhost:8820/api/monitors

# Check history
curl http://localhost:8820/api/monitors/{id}/history

# Public status page (JSON)
curl http://localhost:8820/api/status

# Public status page (HTML — share this URL)
open http://localhost:8820/status
```

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/monitors | Add monitor |
| GET | /api/monitors | List monitors with status |
| GET | /api/monitors/{id} | Monitor detail + recent checks |
| PUT | /api/monitors/{id} | Update monitor |
| DELETE | /api/monitors/{id} | Delete monitor |
| GET | /api/monitors/{id}/history | Check history |
| POST | /api/monitors/{id}/alerts | Add alert webhook (Pro) |
| GET | /api/monitors/{id}/alerts | List alerts |
| DELETE | /api/alerts/{id} | Delete alert |
| GET | /api/status | Status page data (JSON) |
| GET | /status | Public status page (HTML) |
| GET | /health | Health check |
| GET | /ui | Web dashboard |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8820 | HTTP port |
| DATA_DIR | ./data | SQLite data directory |
| RETENTION_DAYS | 30 | Check history retention |
| PADDOCK_LICENSE_KEY | | Pro license key |

## Free vs Pro

| Feature | Free | Pro ($2.99/mo) |
|---------|------|----------------|
| Monitors | 3 | Unlimited |
| Check interval | 5 min | 30 sec |
| History retention | 7 days | 90 days |
| Public status page | ✓ | ✓ |
| Alert webhooks | — | ✓ |
| SSL expiry monitoring | — | ✓ |

## License

Apache 2.0 — see [LICENSE](LICENSE).
