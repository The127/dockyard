# Dockyard

Dockyard is an OCI (Open Container Initiative) compliant container registry implementation written in Go. It provides a lightweight, multi-tenant Docker registry server with support for projects and repositories.

## Features

- **OCI Compliant**: Implements the OCI Distribution Specification for container image storage
- **Multi-tenant Support**: Built-in tenant isolation with project and repository organization
- **Flexible Storage**: Configurable database and cache backends
  - In-memory database (default)
  - In-memory cache (default)
  - Redis cache support
- **REST API**: Administrative API for tenant management and health checks
- **Lightweight**: Minimal dependencies and simple configuration

## Prerequisites

- Go 1.25 or higher
- (Optional) Redis for cache backend

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/The127/dockyard.git
cd dockyard
```

2. Build the application:
```bash
go build -o dockyard ./cmd/dockyard
```

3. Run the server:
```bash
./dockyard
```

## Configuration

Dockyard can be configured using a `config.yml` file or environment variables.

### Configuration File

Create a `config.yml` file in the project root:

```yaml
server:
  port: 8082
  host: 0.0.0.0
  allowedOrigins:
    - "*"

database:
  mode: memory  # or "postgres", "mysql", etc.

cache:
  mode: memory  # or "redis"
  # redis:
  #   host: localhost
  #   port: 6379
```

### Environment Variables

Configuration can also be provided via environment variables with the prefix matching the YAML structure.

## Usage

### Starting the Server

```bash
./dockyard
```

The server will start on port 8082 by default.

### API Endpoints

#### Health Check
```bash
curl http://localhost:8082/admin/api/v1/health
```

#### OCI Registry Endpoint
```bash
curl http://localhost:8082/v2/
```

### Docker Client Usage

Configure your Docker client to use Dockyard as a registry:

```bash
# Push an image
docker tag myimage:latest localhost:8082/tenant/project/myimage:latest
docker push localhost:8082/tenant/project/myimage:latest

# Pull an image
docker pull localhost:8082/tenant/project/myimage:latest
```

## Project Structure

```
.
├── cmd/
│   └── dockyard/          # Main application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── database/          # Database abstractions
│   ├── handlers/          # HTTP request handlers
│   │   ├── adminhandlers/ # Admin API handlers
│   │   └── ocihandlers/   # OCI API handlers
│   ├── middlewares/       # HTTP middlewares
│   ├── repositories/      # Data repositories
│   ├── server/            # HTTP server setup
│   └── services/          # Business logic services
├── config.yml             # Configuration file
└── go.mod                 # Go module dependencies
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o dockyard ./cmd/dockyard
```

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues, questions, or contributions, please use the GitHub issue tracker.
