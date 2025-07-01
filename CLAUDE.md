# Simple Email Server - Development Guidelines

## Core Principles

### 1. Performance First
- Use connection pooling for SMTP connections
- Implement efficient DNS caching with configurable TTL
- Use goroutines for parallel email delivery
- Benchmark all critical paths
- Target: 500+ emails/second throughput

### 2. Red-Green Development (TDD)
- Write failing tests first (Red)
- Implement minimal code to pass (Green)
- Refactor for performance/clarity
- Every feature must have tests before implementation
- Maintain >90% test coverage

### 3. Documentation Requirements
- Update README.md with every new feature
- Document all public APIs
- Include usage examples for each component
- Performance characteristics must be documented

### 4. Testing Strategy
- Unit tests for all components
- Integration tests for SMTP flows
- Benchmark tests for performance-critical paths
- Load tests for queue system
- Mock external dependencies (DNS, SMTP servers)

### 5. Code Organization
```
/cmd/emailserver     - Main server binary
/internal/smtp       - SMTP server implementation
/internal/queue      - Email queue and retry logic
/internal/delivery   - Email delivery client
/internal/api        - HTTP/gRPC API
/pkg/email          - Public email types
/tests              - Integration tests
/benchmarks         - Performance tests
```

## Development Workflow

1. Create GitHub issue for feature
2. Write tests in `*_test.go` files
3. Implement feature to pass tests
4. Run benchmarks to verify performance
5. Update README.md with usage
6. Create PR with test results

## Performance Guidelines

- DNS lookups must be cached (5-minute default TTL)
- Connection pools: min 10, max 100 per destination
- Queue batching: process 100 emails per batch
- Use sync.Pool for object reuse
- Profile CPU and memory usage regularly

## Testing Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run specific test
go test -run TestSMTPServer ./internal/smtp
```

## Key Metrics to Track

- Emails sent per second
- Average delivery latency
- Queue depth
- Failed delivery rate
- DNS cache hit ratio
- Connection pool utilization