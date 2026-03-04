# Code Style Guide

This project follows Go's standard formatting conventions using `gofmt`, `goimports`, and `golangci-lint` to maintain consistent code quality across all files.

## Formatting Rules

Go has a canonical format that is automatically enforced by the Go toolchain:

- **Indentation**: Tabs (not spaces)
- **Line Width**: No hard limit, but prefer readability (typically ~100 characters)
- **Naming**: Follow Go naming conventions (camelCase for exported, PascalCase for public)
- **Comments**: GoDoc style for exported packages, types, functions, and constants
- **Error Handling**: Always handle errors explicitly

## Setup

### 1. Install Go Tooling

```bash
# Install goimports (better version of gofmt)
go install golang.org/x/tools/cmd/goimports@latest

# Install golangci-lint (comprehensive linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install staticcheck (advanced static analysis)
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### 2. Editor Configuration

#### VSCode

Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go) and use the provided `.vscode/settings.json`:

```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.useLanguageServer": true
}
```

#### GoLand

GoLand has built-in support for Go formatting. Enable:

- **Settings → Go → Imports → Optimize imports**
- **Settings → Go → Inspections → Enable all**

## Usage

### Format all files

```bash
# Format with gofmt (built-in)
go fmt ./...

# Format with goimports (also sorts imports)
goimports -w .
```

### Check formatting without changing files

```bash
# Check with gofmt
gofmt -l .

# Check with goimports
goimports -l .

# Verify formatting in CI
goimports -l . | grep -q "\.go$" && exit 1 || exit 0
```

### Run linters

```bash
# Run all linters
golangci-lint run ./...

# Run specific linters
golangci-lint run --disable-all --enable=govet,staticcheck,unused ./...

# Run with auto-fix
golangci-lint run --fix ./...
```

## Editor Integration

The project includes editor configuration files:

- `.editorconfig`: Ensures consistent basic editor settings (indentation with tabs, charset UTF-8)
- `.vscode/settings.json`: VSCode-specific settings for Go development
- `.vscode/extensions.json`: Recommended extensions for VSCode

## Pre-commit Hooks

This project uses pre-commit hooks to enforce code quality:

```bash
# Install git hooks (run once after cloning)
make install-hooks

# Or manually with husky-like setup
go install github.com/evanpurkhiser/goprep@latest
goprep install
```

### Custom Pre-commit Script

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Pre-commit hook for Go projects

echo "Running goimports..."
goimports -w .
if [ -n "$(git diff --name-only)" ]; then
    echo "Code was formatted. Please review and commit again."
    exit 1
fi

echo "Running golangci-lint..."
golangci-lint run ./...
if [ $? -ne 0 ]; then
    echo "Linting failed. Please fix the issues."
    exit 1
fi

echo "Running tests..."
go test ./... -race
if [ $? -ne 0 ]; then
    echo "Tests failed. Please fix the issues."
    exit 1
fi

exit 0
```

Make it executable:

```bash
chmod +x .git/hooks/pre-commit
```

## File Naming Conventions

Go has specific file naming conventions:

| File Type       | Naming Convention | Example                   |
| --------------- | ----------------- | ------------------------- |
| Go source       | `snake_case.go`   | `user_repository.go`      |
| Test files      | `xxx_test.go`     | `user_repository_test.go` |
| Generated files | `xxx_gen.go`      | `wire_gen.go`             |
| Mock files      | `mock_xxx.go`     | `mock_user_repository.go` |

## Code Organization

### Package Structure

```
project/
├── cmd/                    # Main applications
│   ├── auth-service/
│   ├── user-service/
│   └── product-service/
├── internal/               # Private application code
│   ├── domain/
│   ├── repository/
│   ├── handler/
│   └── middleware/
├── pkg/                    # Public library code
│   ├── redis/
│   └── logger/
├── api/                    # API definitions (OpenAPI, protobuf)
├── configs/                # Configuration files
└── docs/                   # Documentation
```

### Import Grouping

```go
package mypackage

// 1. Standard library
import (
    "context"
    "fmt"
    "time"
)

// 2. External dependencies
import (
    "github.com/go-redis/redis/v9"
    "github.com/gorilla/mux"
)

// 3. Internal imports
import (
    "github.com/yourproject/internal/domain"
    "github.com/yourproject/pkg/logger"
)
```

Use `goimports` to automatically sort and group imports.

## Linting Rules

This project uses `golangci-lint` with the following recommended linters:

### Enabled Linters (in `.golangci.yml`)

```yaml
linters:
  enable:
    - govet # Go's built-in vet
    - staticcheck # Advanced static analysis
    - unused # Check for unused code
    - gosimple # Simplify code
    - structcheck # Find unused struct fields
    - varcheck # Find unused global variables
    - ineffassign # Detect ineffective assignments
    - deadcode # Find unused code
    - gofmt # Check code is gofmted
    - goimports # Check import ordering
    - misspell # Fix spelling mistakes
    - gocritic # Go-specific linter
    - revive # Fast, configurable linter
```

## Common Code Patterns

### Error Handling

```go
// Good: Explicit error handling
user, err := repo.FindByID(id)
if err != nil {
    return fmt.Errorf("failed to find user: %w", err)
}

// Bad: Ignoring errors
user, _ := repo.FindByID(id) // NEVER do this
```

### Context Propagation

```go
// Good: Always accept context as first parameter
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    // ...
}

// Good: Always pass context through
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    return r.repo.FindByID(ctx, id)
}
```

### Interface Design

```go
// Good: Small, focused interfaces
type UserFinder interface {
    FindByID(ctx context.Context, id string) (*User, error)
}

type UserCreator interface {
    Create(ctx context.Context, user *User) error
}

// Bad: God interface
type UserRepository interface { // Too many methods
    FindByID(ctx context.Context, id string) (*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
    FindByEmail(ctx context.Context, email string) (*User, error)
    // ... 20 more methods
}
```

## Naming Conventions

### Packages

- **All lowercase**: `userrepository` → `user` (better)
- **Short, descriptive**: `httpserver` → `http` (better)
- **No underscores**: `user_service` → `userservice` (better)
- **Single word when possible**: `authprovider` → `auth` (better)

### Variables and Functions

```go
// Exported (public): PascalCase
type UserService struct {}
func GetUser() {}
const MaxRetries = 3

// Unexported (private): camelCase
type userService struct {}
func getUser() {}
const defaultTimeout = 30
```

### Interfaces

```go
// Good: Interface names end with -er
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Stringer interface {
    String() string
}
```

## Comments and Documentation

### Package Comments

```go
// Package user provides user domain entities and business logic.
//
// This package handles user CRUD operations, authentication,
// and authorization for the microservices architecture.
package user
```

### Exported Function Comments

```go
// FindByID retrieves a user by their unique identifier.
// It returns ErrUserNotFound if no user exists with the given ID.
//
// The context is used for cancellation and timeout control.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    // ...
}
```

## Testing

### Test File Organization

```go
// user_repository_test.go
package repository_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserRepository_FindByID(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        want    *User
        wantErr bool
    }{
        {
            name:    "success - existing user",
            id:      "123",
            want:    &User{ID: "123", Name: "John"},
            wantErr: false,
        },
        {
            name:    "failure - user not found",
            id:      "999",
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Why Consistent Formatting Matters

1. **Reduced Cognitive Load**: Developers don't need to think about formatting, allowing them to focus on code logic
2. **Cleaner Diffs**: Changes in code reviews are easier to read when only functional changes are highlighted
3. **Automated Enforcement**: With pre-commit hooks, formatting is automatically enforced
4. **Team Consistency**: All developers follow the same style, regardless of personal preferences
5. **Easier Onboarding**: New developers can quickly adapt to the project's style
6. **Go Idioms**: Following Go conventions makes code more readable to the Go community

## Troubleshooting

### goimports not formatting files

1. Ensure goimports is installed: `go install golang.org/x/tools/cmd/goimports@latest`
2. Check that the file extension is `.go`
3. Verify the file is not in `.gitignore`

### golangci-lint running slowly

1. Use `--timeout=5m` to increase timeout
2. Run linters in parallel with `--concurrency=4`
3. Disable slow linters: `--disable=gosec`

### Editor-specific issues

For VSCode users:

1. Install the Go extension
2. Ensure the workspace settings are being applied
3. Reload the window if formatting isn't working after configuration changes
4. Check that `GOPATH` and `GOROOT` are configured correctly

## Quick Reference

| Task                 | Command                      |
| -------------------- | ---------------------------- |
| Format code          | `goimports -w .`             |
| Check formatting     | `goimports -l .`             |
| Run linters          | `golangci-lint run ./...`    |
| Run tests            | `go test ./... -race`        |
| Run tests with cover | `go test ./... -cover -race` |
| Vet code             | `go vet ./...`               |
| Install hooks        | `make install-hooks`         |
