# Traefik HTTP Merge

[![Docker Image Size](https://img.shields.io/docker/image-size/hhftechnology/traefik-http-merge/latest)](https://hub.docker.com/r/hhftechnology/traefik-http-merge)
[![License](https://img.shields.io/github/license/hhftechnology/traefik-http-merge)](LICENSE)

A lightweight Go-based HTTP proxy and merger for combining two Traefik dynamic configuration providers into a single API endpoint. The primary provider is treated as read-only (its data takes precedence in merges), while the secondary supports read/write operations.

This tool acts as a shim: for GET requests, it fetches and deeply merges configurations from both providers (primary overrides secondary on conflicts). For non-GET methods (e.g., POST, PUT, DELETE), it proxies requests directly to the secondary provider for writes.

Ideal for scenarios where you have a read-only Traefik config source (e.g., from a monitoring service) and a writable one (e.g., from an admin dashboard), and you want a unified API for Traefik's dynamic configuration.

## Features

- **Deep JSON Merge**: Recursively merges nested objects and appends arrays from both providers.
- **Read-Only Primary**: Primary endpoint data is fetched on reads but never modified.
- **Read/Write Secondary**: All writes are proxied to the secondary endpoint.
- **Configurable via Env Vars**: Easy setup with Docker or bare Go.
- **Lightweight**: Single static binary, no external dependencies (standard library only).
- **Timeout Handling**: Configurable HTTP client timeouts for reliability.
- **Docker Multi-Arch Support**: Builds for amd64; extendable for arm64.

## Quick Start

### Using Docker

Pull the latest image:

```bash
docker pull hhftechnology/traefik-http-merge:latest
```

Run with environment variables for your endpoints:

```bash
docker run -d \
  --name traefik-merge \
  -p 9000:9000 \
  -e MERGE_ENDPOINTS="http://primary:8080/api/traefik,http://secondary:8080/api/traefik" \
  -e MERGE_LISTEN=":9000" \
  hhftechnology/traefik-http-merge:latest
```

- `MERGE_ENDPOINTS`: Comma-separated list (first=primary/read-only, second=secondary/read-write).
- `MERGE_LISTEN`: Bind address (default `:9000`).

Your unified API will be available at `http://localhost:9000/traefik-merged`.

### Using Docker Compose

Example `docker-compose.yml` (adapt to your network):

```yaml
services:
  traefik-merge:
    image: hhftechnology/traefik-http-merge:latest
    container_name: traefik-merge
    restart: unless-stopped
    ports:
      - "9000:9000"
    environment:
      # Primary (read-only) first, secondary (read-write) second
      MERGE_ENDPOINTS: >
        http://pangolin:3001/api/v1/traefik-config,
        http://middleware-manager:80/traefik-config
      MERGE_LISTEN: ":9000"
    networks:
      - pangolin

networks:
  pangolin:
    external: true
```

Run with `docker-compose up -d`.

## API Usage

The proxy exposes a single endpoint: `/traefik-merged`.

### GET /traefik-merged
- **Purpose**: Fetch merged configuration.
- **Response**: JSON object merging both providers (primary overrides secondary).
- **Example**:
  ```bash
  curl http://localhost:9000/traefik-merged
  ```
  Response:
  ```json
  {
    "http": {
      "routers": {
        // Merged routers from both (primary wins on key conflicts)
      },
      "services": {
        // Merged services
      }
    }
  }
  ```

### Non-GET Methods (POST, PUT, DELETE, etc.)
- **Purpose**: Modify secondary provider only.
- **Behavior**: Proxied transparently to secondary (body, headers, method preserved).
- **Example** (adding a router):
  ```bash
  curl -X POST http://localhost:9000/traefik-merged \
    -H "Content-Type: application/json" \
    -d '{
      "http": {
        "routers": {
          "my-router": {
            "rule": "Host(`example.com`)"
          }
        }
      }
    }'
  ```

## Configuration

| Env Var          | Description | Default | Example |
|------------------|-------------|---------|---------|
| `MERGE_ENDPOINTS` | Comma-separated URLs: primary (read-only), secondary (read-write). Required. | N/A | `http://primary:8080/traefik.json,http://secondary:8080/traefik.json` |
| `MERGE_LISTEN`   | HTTP listen address. | `:9000` | `:8080` |

## Building from Source

### Prerequisites
- Go 1.23+
- Docker (for image build)

### Go Build
```bash
git clone https://github.com/hhftechnology/traefik-http-merge.git
cd traefik-http-merge
go mod download
go build -ldflags="-s -w" -o traefik-merge traefik-merge.go
./traefik-merge
```

### Docker Build
```bash
docker build -t traefik-http-merge .
docker run -p 9000:9000 traefik-http-merge
```

## CI/CD with GitHub Actions

This repo includes a workflow (`.github/workflows/publish.yml`) that:
- Triggers on pushes to `main` or tags (`v*`).
- Builds and pushes multi-platform images to Docker Hub (`hhftechnology/traefik-http-merge`) and GitHub Container Registry (`ghcr.io/hhftechnology/traefik-http-merge`).
- Tags: `latest`, branch names, semver (e.g., `v1.0.0`, `1.0`).

Secrets required: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`.

## Contributing

1. Fork the repo.
2. Create a feature branch (`git checkout -b feature/my-feature`).
3. Commit changes (`git commit -m 'Add my feature'`).
4. Push to branch (`git push origin feature/my-feature`).
5. Open a Pull Request.

Report issues or suggest improvements via [GitHub Issues](https://github.com/hhftechnology/traefik-http-merge/issues).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Go](https://go.dev/) standard library.
- Inspired by Traefik's dynamic configuration needs.