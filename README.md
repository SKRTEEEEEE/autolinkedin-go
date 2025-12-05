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
make deps
```

4. Run with Docker (recommended):
```bash
make docker-dev
```

Or run locally:
```bash
make run
```

The API will be available at `http://localhost:8080`.

### Running Tests

Run all tests:
```bash
make test
```

Run tests in isolated Docker environment:
```bash
make docker-test
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
├── configs/                 # Configuration files
└── Makefile                 # Build and development commands
```

## 🔧 Available Commands

- `make build` - Build the application binary
- `make run` - Run the application locally
- `make test` - Run all tests
- `make docker-dev` - Start development environment with Docker
- `make docker-test` - Run tests in isolated Docker containers
- `make lint` - Run code linters
- `make fmt` - Format code

## 🐳 Docker Environments

### Development Mode
Uses hot reload for instant code changes:
```bash
make docker-dev
```

### Test Mode
Isolated ephemeral containers with automatic cleanup:
```bash
make docker-test
```

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
