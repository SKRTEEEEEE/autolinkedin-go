# LinkGen AI

**LinkGen AI** is an automated LinkedIn content generation system built with Go and Clean Architecture principles. It generates ideas, creates drafts (posts and articles), refines content, and publishes to LinkedIn automatically.

## 🏗️ Architecture

This project follows **Clean Architecture** with four distinct layers:

- **Domain**: Business logic and entities (User, Topic, Idea, Draft)
- **Application**: Use cases and orchestration (Idea generation, Draft creation, Publishing)
- **Infrastructure**: External services (MongoDB, NATS, LLM HTTP client, LinkedIn API)
- **Interfaces**: HTTP handlers and routes

For detailed architecture documentation, see [docs/arquitectura-app.md](./docs/arquitectura-app.md).

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- MongoDB
- NATS

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/linkgen-ai/backend.git
cd backend
```

2. Copy environment configuration:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Install dependencies:
```bash
cd src
go mod download
go mod tidy
cd ..
```

4. Run with Docker (recommended):
```bash
docker-compose up -d
```

Or run locally:
```bash
cd src
go run main.go
```

The API will be available at `http://localhost:8080`.

### Running Tests

Run all tests:
```bash
go test -v -race -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out -o coverage.html
```

Run tests in isolated Docker environment:
```bash
docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from app
docker-compose -f docker-compose.test.yml down -v
```

## 📁 Project Structure

```
.
├── src/                      # Source code
│   ├── domain/              # Business entities and rules
│   ├── application/         # Use cases and services
│   ├── infrastructure/      # External implementations
│   └── interfaces/          # HTTP handlers and routes
├── test/                    # Tests (mirrors src structure)
├── docs/                    # Documentation
├── scripts/                 # Utility scripts
└── configs/                 # Configuration files
```

## 🔧 Available Commands

### Development
- `cd src && go build -o ../bin/linkgenai main.go` - Build the application binary
- `cd src && go run main.go` - Run the application locally
- `cd src && go mod download && go mod tidy` - Install Go dependencies
- `cd src && go fmt ./... && cd ../test && go fmt ./...` - Format code

### Testing
- `go test -v -race -coverprofile=coverage.out ./test/...` - Run all tests locally
- `docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from app` - Run tests in isolated Docker environment
- `bash scripts/lint.sh` - Run golangci-lint for code quality checks
- `bash scripts/ci-check.sh` - Run complete CI/CD validation suite

### Docker
- `docker-compose up -d` - Start development environment with hot reload
- `docker-compose down && docker-compose -f docker-compose.test.yml down -v` - Stop all Docker containers
- `bash scripts/validate-docker.sh` - Validate Docker configurations

### Utilities
- `rm -rf bin/ coverage.out coverage.html && cd src && go clean` - Clean build artifacts and caches
- `cd src && go get -u ./... && go mod tidy` - Update Go dependencies

## 🐳 Docker Environments

### Development Mode
Uses hot reload for instant code changes:
```bash
docker-compose up -d
# or
bash scripts/dev.sh
```

The development environment includes:
- **Hot reload** with Air - code changes are detected automatically
- **Persistent volumes** for MongoDB and NATS data
- **Volume mounts** for source code (./src:/app)
- Services: MongoDB (27017), NATS (4222), App (8080)

### Test Mode
Isolated ephemeral containers with automatic cleanup:
```bash
docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from app
docker-compose -f docker-compose.test.yml down -v
# or
bash scripts/test.sh
```

The test environment features:
- **Ephemeral storage** using tmpfs (RAM-based, no disk writes)
- **Isolated network** separate from development
- **Automatic cleanup** after test execution
- **No persistent volumes** - all data is temporary

### Docker Validation
Validate your Docker configurations:
```bash
bash scripts/validate-docker.sh
```

This checks:
- Docker and Docker Compose installation
- Syntax validation of docker-compose.yml and docker-compose.test.yml
- Dockerfile multi-stage build validation

## 📚 Documentation

- [Architecture Overview](./docs/arquitectura-app.md)
- [Agent Guidelines](./AGENTS.md)
- [API Documentation](./docs/api/)

## 🤝 Contributing

This project follows strict contribution guidelines defined in [AGENTS.md](./AGENTS.md):

- All commits must use Conventional Commits format
- All commits must be signed
- Tests must maintain 80%+ coverage
- Follow Clean Architecture boundaries

## 📝 License

[Add license information]

## 👥 Authors

CO-CREATED by Agent666 — ⟦ Product of SKRTEEEEEE ⟧
