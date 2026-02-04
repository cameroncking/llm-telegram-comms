# Contributing to llm-telegram-comms

Thank you for your interest in contributing to llm-telegram-comms!

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/exedev/llm-telegram-comms.git
   cd llm-telegram-comms
   ```

2. Ensure you have Go 1.21+ installed:
   ```bash
   go version
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build the project:
   ```bash
   go build -o llm-telegram-comms .
   ```

## Project Structure

```
llm-telegram-comms/
├── main.go              # Entry point, CLI parsing, daemon mode
├── main_test.go         # Tests for CLI and daemon functionality
├── config/
│   ├── config.go        # Configuration loading and validation
│   └── config_test.go   # Configuration tests
├── backend/
│   ├── backend.go       # Command execution
│   └── backend_test.go  # Backend tests
├── bot/
│   ├── bot.go           # Telegram bot handler
│   ├── bot_test.go      # Bot tests
│   └── attachment_test.go # Attachment handling tests
├── config.example.json  # Example configuration
├── README.md
├── LICENSE
└── CONTRIBUTING.md
```

## Code Standards

### Formatting

- Run `go fmt` before committing:
  ```bash
  go fmt ./...
  ```

- Run `go vet` to catch common issues:
  ```bash
  go vet ./...
  ```

### Style Guidelines

- Follow standard Go conventions and idioms
- Use meaningful variable and function names
- Keep functions focused and reasonably sized
- Add comments for exported functions and non-obvious logic
- Handle errors explicitly; don't ignore them

### Commit Messages

- Use clear, descriptive commit messages
- Start with a short summary line (50 chars or less)
- Add a blank line followed by details if needed
- Reference issues when applicable

Example:
```
Add support for voice message attachments

- Handle voice messages in handleAttachments()
- Save as .ogg files with timestamp prefix
- Add tests for voice message handling
```

## Testing

### Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test ./... -v
```

Run tests for a specific package:
```bash
go test ./config -v
go test ./bot -v
go test ./backend -v
```

Run a specific test:
```bash
go test -run TestIsUserAllowed ./bot -v
```

### Test Coverage

Generate coverage report:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Writing Tests

- All new features must include tests
- All bug fixes should include a test that reproduces the bug
- Use table-driven tests where appropriate
- Test both success and error cases
- Keep tests independent and idempotent

Example table-driven test:
```go
func TestIsUserAllowed(t *testing.T) {
    tests := []struct {
        name     string
        cfg      Config
        userID   int64
        expected bool
    }{
        {
            name:     "allowlist not required",
            cfg:      Config{UserAllowlistRequired: false},
            userID:   123,
            expected: true,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := tt.cfg.IsUserAllowed(tt.userID); got != tt.expected {
                t.Errorf("IsUserAllowed(%d) = %v, want %v", tt.userID, got, tt.expected)
            }
        })
    }
}
```

## Submitting Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit them:
   ```bash
   git add .
   git commit -m "Add your feature"
   ```

3. Ensure all tests pass:
   ```bash
   go test ./...
   ```

4. Ensure code is formatted:
   ```bash
   go fmt ./...
   go vet ./...
   ```

5. Push your branch and open a pull request

## Reporting Issues

When reporting issues, please include:

- Go version (`go version`)
- Operating system
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Relevant log output or error messages

## License

By contributing, you agree that your contributions will be licensed under the ISC License.
