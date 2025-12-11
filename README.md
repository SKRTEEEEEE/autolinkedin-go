# LinkGen AI

**LinkGen AI** is an automated LinkedIn content generation system built with Go and Clean Architecture principles. It generates ideas, creates drafts (posts and articles), refines content, and publishes to LinkedIn automatically.

## 🏗️ Architecture

This project follows **Clean Architecture** with four distinct layers:

- **Domain**: Core entities (User, Topic, Idea, Draft, Prompt) and business rules
- **Application**: Use cases (Idea generation, Draft creation, Refinement, Publishing) and orchestration
- **Infrastructure**: External services (MongoDB, NATS, LLM HTTP client, LinkedIn API)
- **Interfaces**: HTTP handlers and routes for REST API

For detailed architecture documentation, see [docs/flujo-app.md](./docs/flujo-app.md).

## 🚀 Quick Start

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose
- MongoDB
- NATS

### Clone & Setup
```bash
git clone https://github.com/linkgen-ai/backend.git
cd backend
cp .env.example .env && nano .env
```

### Install Dependencies
```bash
cd src && go mod download && go mod tidy && cd ..
```

### Run Application
```bash
# With Docker (recommended)
docker-compose up -d

# Or run locally
cd src && go run main.go
```

The API will be available at `http://localhost:8080`.

### Testing
```bash
# Run all tests locally
go test -v -race -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out -o coverage.html

# Run tests in Docker
docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from app
docker-compose -f docker-compose.test.yml down -v
```

## 📁 Project Structure

```
├── src/              # Source code: domain/app/infra/interfaces
├── test/             # Tests (mirrors src structure + http folder)
├── docs/             # Documentation (flujo-app.md)
├── scripts/          # Utility scripts
├── configs/          # Configuration files
└── bin/              # Build output
```





## 📚 Documentation

- [Application Flow](./docs/flujo-app.md)
- [Agent Guidelines](./AGENTS.md)
- [API Documentation](./docs/api/)

## 🤝 Contributing

See [AGENTS.md](./AGENTS.md) for guidelines:
- Conventional Commits format required
- Signed commits with attribution
- 80%+ test coverage
- Clean Architecture boundaries

## 📝 License

[Add license information]

## 👥 Authors
### [SKRTEEEEEE](dev.desarollador.tech)
👿 *This project is auto-generated as a part of the training and testing program for Agent666*
#### CO-CREATED by Agent666 — ⟦ Product of SKRTEEEEEE ⟧
