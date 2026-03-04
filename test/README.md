# Testing Guide

This document provides comprehensive guidelines for testing the Go microservices boilerplate.

## Table of Contents

- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Test Categories](#test-categories)
- [Best Practices](#best-practices)

## Test Structure

```
test/
├── testutil/          # Testing utilities and helpers
├── suite/             # Test suite base classes
├── integration/       # Integration tests (to be added)
├── e2e/              # End-to-end tests (to be added)
└── README.md         # This file

internal/<service>/
├── usecase/
│   ├── <service>_usecase_test.go       # Unit tests
│   └── <service>_usecase_bench_test.go # Benchmarks
├── delivery/
│   ├── handler_test.go                # HTTP handler tests
│   └── mocks/                         # Mock implementations
└── <service>_integration_test.go      # Integration tests
```

## Running Tests

### All Tests
```bash
make test
```

### Tests with Coverage
```bash
make test-coverage
```

### Specific Package
```bash
make test-internal/user/usecase
```

### Integration Tests
```bash
make test-integration
```

### E2E Tests
```bash
make test-e2e
```

### With Race Detection
```bash
go test -race ./...
```

### Run Specific Test
```bash
go test -v -run TestUpdateProfile ./internal/user/usecase/...
```

### Run Benchmarks
```bash
go test -bench=. -benchmem ./...
```

## Writing Tests

### Unit Tests

Use table-driven tests for multiple scenarios:

```go
func TestUpdateProfile(t *testing.T) {
    tests := []struct {
        name        string
        req         *dto.UpdateProfileRequest
        setupMocks  func(*MockRepository, *MockEventBus)
        wantErr     bool
        expectedErr error
    }{
        {
            name: "successful update",
            req:  &dto.UpdateProfileRequest{...},
            setupMocks: func(repo *MockRepository, bus *MockEventBus) {
                repo.On("GetProfile", ...).Return(profile, nil)
            },
            wantErr: false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            repo := new(MockRepository)
            tt.setupMocks(repo, ...)

            // Execute
            err := usecase.UpdateProfile(context.Background(), tt.req)

            // Assert
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests

Test database operations with real database:

```go
func TestUserRepository_Integration(t *testing.T) {
    db := testutil.SetupTestDB(t, &domain.User{}, &domain.Profile{})
    defer testutil.CleanupDB(t, db, &domain.User{}, &domain.Profile{})

    repo := repository.NewUserRepository(db)

    // Test create and find
    user := &domain.User{Email: "test@example.com", ...}
    err := repo.Create(context.Background(), user)
    require.NoError(t, err)

    found, err := repo.FindByID(context.Background(), user.ID, ...)
    require.NoError(t, err)
    assert.Equal(t, user.Email, found.Email)
}
```

### HTTP Handler Tests

Test HTTP handlers with mock use cases:

```go
func TestGetUser_Success(t *testing.T) {
    mockUseCase := new(mocks.MockUserUseCase)
    handler := delivery.NewUserHandler(mockUseCase)

    expectedUser := &dto.UserResponse{ID: "user-123", ...}
    mockUseCase.On("GetUser", ...).Return(expectedUser, nil)

    req, _ := http.NewRequest("GET", "/api/v1/users/user-123", nil)
    w := httptest.NewRecorder()

    router := setupTestRouter(handler)
    router.GET("/api/v1/users/:id", handler.GetUser)
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    mockUseCase.AssertExpectations(t)
}
```

## Test Categories

### 1. Unit Tests
- Test individual functions and methods
- Use mocks for external dependencies
- Fast execution (< 1ms per test)

### 2. Integration Tests
- Test interaction between components
- Use real database (in-memory or test container)
- Slower execution (10-100ms per test)

### 3. E2E Tests
- Test complete workflows
- Use real services and infrastructure
- Slowest execution (> 100ms per test)

### 4. Benchmark Tests
- Measure performance of critical paths
- Use `b.ResetTimer()` for accurate measurements
- Run with `go test -bench=.`

## Best Practices

### DO's
- Use table-driven tests for multiple scenarios
- Use `require` for fatal assertions, `assert` for non-fatal
- Clean up resources in `defer` or `t.Cleanup()`
- Use descriptive test names that explain what is being tested
- Test error paths, not just happy paths
- Use subtests with `t.Run()` for related test cases
- Use race detector in CI (`go test -race`)
- Aim for 80%+ code coverage

### DON'Ts
- Don't sleep in tests (use channels or context)
- Don't test external services (mock them)
- Don't skip tests without a reason
- Don't ignore test failures
- Don't use production database credentials
- Don't share state between tests

### Organization

1. **Arrange-Act-Assert Pattern**
```go
// Arrange - Setup test data and mocks
user := &domain.User{...}
mockRepo.On("Create", ...).Return(nil)

// Act - Execute the code under test
err := repo.Create(context.Background(), user)

// Assert - Verify expected outcomes
require.NoError(t, err)
assert.NotEmpty(t, user.ID)
```

2. **Test Naming**
```go
// Good
func TestUpdateProfile_Success(t *testing.T) { ... }
func TestUpdateProfile_ValidationError(t *testing.T) { ... }
func TestUpdateProfile_UserNotFound(t *testing.T) { ... }

// Bad
func TestProfile1(t *testing.T) { ... }
func TestProfile2(t *testing.T) { ... }
```

3. **Test Isolation**
```go
func TestSomething(t *testing.T) {
    // Create fresh database for each test
    db := testutil.SetupTestDB(t, &domain.User{})
    defer testutil.CleanupDB(t, db, &domain.User{})

    // Test runs in isolation
}
```

## CI/CD Integration

Tests run automatically in CI:

```yaml
# .github/workflows/test.yml
- name: Run tests
  run: make test

- name: Run tests with coverage
  run: make test-coverage

- name: Run integration tests
  run: make test-integration
  env:
    TEST_DB_HOST: postgres-test
```

## Environment Variables for Testing

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_DB_HOST` | Test database host | localhost |
| `TEST_DB_PORT` | Test database port | 5432 |
| `TEST_DB_NAME` | Test database name | test_db |
| `TEST_DB_USER` | Test database user | test |
| `TEST_DB_PASSWORD` | Test database password | test |

## Useful Test Commands

```bash
# Run tests with verbose output
go test -v ./...

# Run tests with coverage for specific package
go test -coverprofile=coverage.out -covermode=atomic ./internal/user/...

# View coverage report
go tool cover -html=coverage.out

# Run tests and skip long-running tests
go test -short ./...

# Run tests with race detector
go test -race ./...

# Run tests with count (for flaky test detection)
go test -count=10 ./...

# Run tests and generate coverage badge
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Test Utilities

### testutil Package

```go
// Setup in-memory database
db := testutil.SetupTestDB(t, &domain.User{}, &domain.Profile{})

// Create test data
user := testutil.CreateTestUser(t, db, "test@example.com")

// Clean up
testutil.CleanupDB(t, db, &domain.User{})

// Get environment variable with default
value := testutil.GetTestEnv("MY_VAR", "default")

// Wait for condition
testutil.WaitForCondition(t, func() bool {
    return checkSomething()
}, 5*time.Second, 100*time.Millisecond)
```

### Suite Package

```go
type MyTestSuite struct {
    suite.TestSuite
    DB *gorm.DB
}

func (s *MyTestSuite) SetupSuite() {
    s.DB = testutil.SetupTestDB(s.T(), &domain.User{})
}

func (s *MyTestSuite) TestSomething() {
    // Use s.DB in tests
}

func TestMyTestSuite(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}
```

## Resources

- [Go Testing Guide](https://golang.org/pkg/testing/)
- [Testify Assertions](https://github.com/stretchr/testify)
- [Table Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Go Mock](https://github.com/golang/mock)
