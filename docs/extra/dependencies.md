# Dependencies Documentation — LinkGen AI

## Overview
This document describes all core dependencies used in LinkGen AI, their purpose, version constraints, and update procedures.

---

## Core Dependencies

### Web Framework & Routing

#### `github.com/gin-gonic/gin` (v1.11.0)
**Purpose**: High-performance HTTP web framework  
**Rationale**: Gin provides excellent performance, middleware support, and clean routing for our REST API. It's widely adopted, well-maintained, and offers built-in validation, JSON handling, and error management.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Follow semantic versioning, test thoroughly before minor/major updates

---

### Database & Storage

#### `go.mongodb.org/mongo-driver` (v1.17.6)
**Purpose**: Official MongoDB Go driver  
**Rationale**: Required for all database operations (users, topics, ideas, drafts). Official driver ensures best compatibility and performance with MongoDB.  
**Compatibility**: Go 1.21+, MongoDB 4.0+  
**Update Strategy**: Update regularly for security patches, test migrations carefully

#### `github.com/go-playground/validator/v10` (v10.28.0)
**Purpose**: Struct validation  
**Rationale**: Provides declarative validation for domain entities and API request/response structures, reducing boilerplate code.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Minor updates are generally safe, validate custom rules after updates

---

### Messaging & Queue

#### `github.com/nats-io/nats.go` (v1.47.0)
**Purpose**: NATS messaging client  
**Rationale**: Lightweight message queue for asynchronous draft generation. Simple pub/sub pattern with minimal overhead and no complex streaming requirements.  
**Compatibility**: Go 1.21+, NATS Server 2.0+  
**Update Strategy**: Follow NATS server version compatibility, test queue operations after updates

---

### HTTP Client & Rate Limiting

#### `github.com/go-resty/resty/v2` (v2.17.0)
**Purpose**: Enhanced HTTP client  
**Rationale**: Provides retry logic, timeout handling, and better error management for LLM and LinkedIn API calls.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Update regularly, test external API integrations

#### `golang.org/x/time/rate` (v0.14.0)
**Purpose**: Rate limiting  
**Rationale**: Prevents overwhelming external APIs (LLM, LinkedIn) with too many requests. Implements token bucket algorithm.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Minimal breaking changes expected, update as needed

---

### Configuration Management

#### `github.com/spf13/viper` (v1.21.0)
**Purpose**: Configuration management  
**Rationale**: Supports multiple config sources (.env, YAML, environment variables, flags). Centralizes all configuration logic.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Test configuration loading after updates, verify default values

#### `github.com/joho/godotenv` (v1.5.1)
**Purpose**: .env file loader  
**Rationale**: Simplifies local development environment setup. Loads environment variables from .env files.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Stable package, update rarely

---

### Logging & Observability

#### `go.uber.org/zap` (v1.27.1)
**Purpose**: Structured logging  
**Rationale**: High-performance structured logger with zero allocations in hot paths. Better than logrus for production environments.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Test log output format after updates

#### `github.com/sirupsen/logrus` (v1.9.3)
**Purpose**: Legacy structured logging  
**Rationale**: Currently used in some legacy code, will be migrated to zap.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Minimal updates, plan migration to zap

---

### Testing & Mocking

#### `github.com/stretchr/testify` (v1.11.1)
**Purpose**: Testing assertions and test suites  
**Rationale**: Industry standard for Go testing. Provides assert, require, mock, and suite packages.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Safe to update regularly

#### `go.uber.org/mock` (v0.6.0)
**Purpose**: Mock generation  
**Rationale**: Generates mocks from interfaces for testing. Essential for unit testing with clean architecture.  
**Compatibility**: Go 1.21+  
**Usage**: `mockgen -source=<file.go> -destination=<mock_file.go>`  
**Update Strategy**: Test mock generation after updates

---

### Development Tools

#### `github.com/air-verse/air` (v1.63.4)
**Purpose**: Hot reload for development  
**Rationale**: Automatically rebuilds and restarts the application on code changes. Essential for development productivity.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Update as needed, not critical for production

#### `github.com/golang-migrate/migrate/v4` (v4.19.1)
**Purpose**: Database migrations  
**Rationale**: Manages MongoDB schema changes and data migrations in a version-controlled manner.  
**Compatibility**: Go 1.21+  
**Update Strategy**: Test migration rollback after updates

#### `github.com/swaggo/swag` (v1.16.6)
**Purpose**: Swagger documentation generator  
**Rationale**: Generates OpenAPI/Swagger specs from code annotations. Keeps API documentation in sync with code.  
**Compatibility**: Go 1.21+  
**Usage**: `swag init` to generate docs  
**Update Strategy**: Update periodically, verify generated docs

---

## Version Constraints

### Go Version
- **Minimum**: Go 1.21
- **Recommended**: Go 1.25+
- **Reason**: Clean architecture requires generics and modern Go features

### MongoDB Version
- **Minimum**: MongoDB 4.0
- **Recommended**: MongoDB 6.0+
- **Reason**: Requires transaction support and modern query features

### NATS Version
- **Minimum**: NATS Server 2.0
- **Recommended**: NATS Server 2.10+
- **Reason**: Requires basic pub/sub, persistence optional

---

## Update Procedures

### Regular Updates (Monthly)
```bash
# Update all dependencies
make deps-update

# Verify no breaking changes
make test
make lint

# Test Docker environments
make docker-test
```

### Security Updates (Immediate)
```bash
# Update specific package
cd src && go get -u <package@version>

# Rebuild and test
make deps
make test
make docker-test
```

### Major Version Updates
1. Check changelog for breaking changes
2. Update in a feature branch
3. Run full test suite including integration tests
4. Update Docker configurations if needed
5. Deploy to staging first
6. Monitor for issues

---

## Dependency Management Commands

```bash
# Install all dependencies
make deps

# Update all dependencies
make deps-update

# Clean module cache
make deps-clean

# Vendor dependencies (optional)
make vendor

# Verify dependencies
cd src && go mod verify

# Why is package X included?
cd src && go mod why <package>

# Graph of dependencies
cd src && go mod graph
```

---

## Conflict Resolution

If dependency conflicts occur:

1. **Check compatibility matrix** in this document
2. **Use go mod why** to understand why package is required
3. **Check for replace directives** needed in go.mod
4. **Consult package documentation** for compatible versions
5. **Test thoroughly** after resolving conflicts

---

## Notes

- All dependencies are managed via Go modules (`go.mod`, `go.sum`)
- Never manually edit `go.sum` - it's automatically generated
- Vendor directory is optional - we rely on module cache
- Keep `go.mod` in src/ directory, not root
- Test dependencies should also be declared in require section
- Use exact versions for critical dependencies (MongoDB, NATS)
- Allow minor/patch updates for development tools

---

## Last Updated
2025-12-05
