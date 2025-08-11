# Nginx Proxy Manager Helper

A Go application that automatically manages Nginx Proxy Manager configurations based on Docker container labels.

## Features

- Automatically creates and manages proxy hosts based on Docker container labels
- Supports HTTPS with Let's Encrypt certificates
- Cloudflare DNS integration for wildcard certificates
- Database persistence for domain configurations
- Container lifecycle monitoring and cleanup
- RESTful API integration with Nginx Proxy Manager

## Project Structure

```
nginx-proxy-manager-helper/
├── cmd/nginx-proxy-manager-helper/     # Application entry point
│   └── main.go
├── internal/                           # Private application code
│   ├── cloudflare/                     # Cloudflare DNS client
│   │   ├── client.go
│   │   └── client_test.go
│   ├── config/                         # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── database/                       # Database operations
│   │   ├── database.go
│   │   └── database_test.go
│   ├── models/                         # Data models
│   │   ├── domain.go
│   │   └── domain_test.go
│   ├── npm/                           # Nginx Proxy Manager client
│   │   ├── client.go
│   │   ├── client_test.go
│   │   ├── certificates.go
│   │   └── certificates_test.go
│   ├── reconciler/                     # Container reconciliation logic
│   │   ├── reconciler.go
│   │   └── reconciler_test.go
│   └── utils/                          # Utility functions
│       ├── utils.go
│       └── utils_test.go
├── test/                               # Integration tests
│   └── integration_test.go
├── Makefile                            # Build and test automation
├── go.mod                              # Go module definition
├── go.sum                              # Dependency checksums
└── .env                                # Environment configuration
```

## Configuration

Create a `.env` file or set environment variables:

```bash
# Nginx Proxy Manager Configuration
NPM_BASE_URL=http://localhost:81
NPM_EMAIL=admin@example.com
NPM_PASSWORD=changeme

# Cloudflare Configuration (optional)
CLOUDFLARE_ENABLED=true
CLOUDFLARE_TOKEN=your-cloudflare-api-token
CLOUDFLARE_ZONE_ID=your-zone-id
CLOUDFLARE_DOMAINS=example.com,*.example.com

# Certificate Configuration
CREATE_WILDCARD_CERTS=false
LETSENCRYPT_EMAIL=your-email@example.com
```

## Docker Container Labels

Add labels to your Docker containers to automatically create proxy hosts:

```bash
# Single domain
docker run -l nginx-domain="example.com~192.168.1.10~8080" your-image

# Multiple domains
docker run \
  -l nginx-domain-1="api.example.com~192.168.1.10~8080" \
  -l nginx-domain-2="https://secure.example.com~192.168.1.10~8443" \
  your-image

# Multiple entries in one label
docker run -l nginx-domain="api.example.com~192.168.1.10~8080,web.example.com~192.168.1.10~3000" your-image
```

Format: `domain~ip_address~port`
- Use `https://` prefix for domains that should use SSL certificates
- Use `http://` or no prefix for HTTP-only domains

## Building and Running

### Using Make

```bash
# Build the application
make build

# Run tests
make test

# Run specific test suites
make test-unit          # Unit tests only
make test-integration   # Integration tests only
make test-coverage      # Tests with coverage report

# Development workflow
make dev               # Format, vet, test, and build

# Run the application
make run
```

### Manual Build

```bash
# Build
cd cmd/nginx-proxy-manager-helper
go build -o ../../nginx-proxy-manager-helper

# Run
./nginx-proxy-manager-helper
```

## Testing

The project includes comprehensive tests:

### Test Types

1. **Unit Tests** - Test individual components in isolation
2. **Integration Tests** - Test component interactions
3. **Mock Tests** - Test with mocked external dependencies

### Running Tests

```bash
# All tests
make test

# Unit tests only (fastest)
make test-unit

# Integration tests
make test-integration

# Tests with coverage
make test-coverage

# Individual package tests
make test-config
make test-models
make test-utils
make test-database
make test-npm
make test-cloudflare
make test-reconciler
```

### Test Coverage

Generate a coverage report:

```bash
make test-coverage
# Opens coverage.html in your browser
```

## Development

### Code Organization

- **cmd/** - Application entry points
- **internal/** - Private application packages
- **test/** - Integration tests

### Package Responsibilities

- **config** - Configuration loading and validation
- **cloudflare** - Cloudflare DNS API client
- **database** - SQLite database operations
- **models** - Data structures and domain models
- **npm** - Nginx Proxy Manager API client
- **reconciler** - Container monitoring and reconciliation
- **utils** - Shared utility functions

### Code Quality

```bash
# Format code
make fmt

# Vet code
make vet

# Run linter (requires golangci-lint)
make lint

# Clean up modules
make tidy
```

### Adding Tests

When adding new functionality:

1. Add unit tests for individual functions
2. Add integration tests for component interactions
3. Update the Makefile if needed
4. Ensure all tests pass: `make test`

## Architecture

The application follows these principles:

- **Clean Architecture** - Separation of concerns with clear boundaries
- **Dependency Injection** - Components receive their dependencies
- **Interface Segregation** - Small, focused interfaces
- **Testability** - All components can be tested in isolation
- **Error Handling** - Comprehensive error handling with logging

## API Integration

### Nginx Proxy Manager API

The application integrates with NPM's REST API to:
- Create/update/delete proxy hosts
- Manage SSL certificates
- Configure domain routing

### Cloudflare API

Optional integration for:
- Dynamic DNS updates
- Wildcard certificate creation
- DNS challenge validation

## Monitoring

The application provides extensive logging:
- Container lifecycle events
- API interactions
- Configuration changes
- Error conditions

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `make test`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.