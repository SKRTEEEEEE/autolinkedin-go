# Configuration Guide

## Overview

LinkGen AI uses a hierarchical configuration system with the following precedence (highest to lowest):

1. **Command-line flags** (e.g., `--server-port=8080`)
2. **Environment variables** (prefixed with `LINKGEN_`)
3. **Configuration files** (`configs/*.yaml`)
4. **Default values** (defined in code)

## Environment Variables

### Naming Convention

All environment variables must be prefixed with `LINKGEN_` to avoid conflicts with system variables.

**Format**: `LINKGEN_<COMPONENT>_<SETTING>`

Examples:
- `LINKGEN_SERVER_PORT`
- `LINKGEN_MONGODB_URI`
- `LINKGEN_LLM_MODEL`

### Philosophy

The `.env` file should contain **only**:
1. **Secrets** (API keys, passwords, tokens)
2. **Environment-specific values** (different between dev/staging/prod)

All other configuration should use sensible defaults defined in the code (`src/infrastructure/config/loader.go`).

### Required Variables

Only the LLM configuration is required:

```bash
LINKGEN_LLM_ENDPOINT=http://100.105.212.98:8317/
LINKGEN_LLM_MODEL=claude-3-7-sonnet-20250219
```

### Optional Variables

Everything else has defaults and only needs to be set if you want to override them.

See `.env.example` for a complete list of available variables.

## Docker Compose

When running with Docker Compose, the following variables are automatically overridden to use Docker service names:

- `LINKGEN_MONGODB_URI=mongodb://mongodb:27017/linkgenai`
- `LINKGEN_NATS_URL=nats://nats:4222`

This allows the containerized app to communicate with other services in the Docker network.

## Configuration Precedence Example

Given this setup:

**Code default**: `Port = 8000`
**Environment variable**: `LINKGEN_SERVER_PORT=9000`
**Command-line flag**: `--server-port=7000`

The application will use **port 7000** (command-line flag wins).

## Development Setup

1. Copy the example file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and set your LLM configuration:
   ```bash
   LINKGEN_LLM_ENDPOINT=http://100.105.212.98:8317/
   LINKGEN_LLM_MODEL=claude-3-7-sonnet-20250219
   ```

3. (Optional) Override any defaults if needed:
   ```bash
   LINKGEN_LOG_LEVEL=debug
   LINKGEN_SERVER_PORT=9000
   ```

4. Start the application:
   ```bash
   docker-compose up -d
   ```

## Production Setup

For production, you should set secrets via environment variables, not in the `.env` file:

```bash
export LINKGEN_LINKEDIN_CLIENT_ID="your-real-client-id"
export LINKGEN_LINKEDIN_CLIENT_SECRET="your-real-secret"
export LINKGEN_LLM_API_KEY="your-llm-api-key"
```

Alternatively, use a secrets management system like:
- Docker Secrets
- Kubernetes Secrets
- HashiCorp Vault
- AWS Secrets Manager

## Available Models

Check available LLM models at: http://100.105.212.98:8317/v1/models

Current recommended models:
- `claude-3-7-sonnet-20250219` - Best balance of quality and speed
- `claude-sonnet-4-5-20250929` - Highest quality
- `gpt-5.2` - OpenAI's latest (if available)

## Troubleshooting

### Variable not being read

1. Check the prefix: all variables must start with `LINKGEN_`
2. Restart the application/containers after changing `.env`
3. In Docker, check with: `docker exec linkgenai-app-dev env | grep LINKGEN`

### Default value not working

Check `src/infrastructure/config/loader.go` for the actual default value.

### Docker networking issues

If the app can't connect to MongoDB/NATS inside Docker:
- Verify docker-compose is overriding the connection URLs
- Check container names match the service names in docker-compose.yml
- Ensure all services are on the same network
