# Scheduler API

A Go-based REST API for managing student and tutor scheduling, built with OpenAPI and code generation.

## Prerequisites

1. **Install Go** (version 1.24.5 or later)
   - Download from [golang.org](https://golang.org/downloads/)
   - Or use a package manager:
     - macOS: `brew install go`
     - Linux: `sudo apt install golang` (Ubuntu/Debian)
     - Windows: Download installer from golang.org

2. **Verify Go installation**

   ```bash
   go version
   ```

## Setup

1. **Clone the repository**

   ```bash
   git clone <repository-url>
   cd bookSmart
   ```

2. **Install dependencies**

   ```bash
   go mod download
   ```

3. **Set up code quality tools**

   ```bash
   # Install pre-commit
   pip install pre-commit
   # or
   brew install pre-commit

   # Install the hooks
   pre-commit install
   ```

4. **Generate API code from OpenAPI spec**

   ```bash
   go generate ./...
   ```

   This generates Go server code and models from the OpenAPI specification in `internal/scheduler/scheduler.openapi.yml`.

## Project Structure

```text
├── go.mod                          # Go module definition
├── internal/
│   └── scheduler/
│       ├── generate.go             # Go generate directives
│       ├── models.cfg.yml          # Config for generating models
│       ├── server.cfg.yml          # Config for generating server
│       ├── scheduler.openapi.yml   # OpenAPI specification
│       ├── models.gen.go           # Generated data models
│       └── server.gen.go           # Generated server code
```

## API Overview

The API provides endpoints for:

- **Organizations** - Create and manage organizations
- **Users** - Manage students, tutors, and administrators
- **Availability** - Track when users are available
- **Courses** - Define recurring course structures
- **Classes** - Schedule individual class sessions
- **Trackers** - Monitor course progress and completion

## Development

### Regenerating Code

After modifying the OpenAPI spec (`scheduler.openapi.yml`), regenerate the Go code:

```bash
go generate ./...
```

### API Documentation

The OpenAPI specification is located at `internal/scheduler/scheduler.openapi.yml`. You can view it in:

- [Swagger Editor](https://editor.swagger.io/) (paste the YAML content)
- Any OpenAPI-compatible tool

## Code Quality & Linting

### Run linting

```bash
just lint
```

The linters will automatically run on every commit.

## Next Steps

- Implement handler functions for the generated server interfaces
- Add database integration
- Set up authentication/authorization
- Add tests
- Deploy configuration
