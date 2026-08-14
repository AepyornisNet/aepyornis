# Contributing to GoGIS

Thank you for your interest in contributing to GoGIS! This document provides guidelines and information for contributors.

## Code of Conduct

We are committed to providing a welcoming and inclusive experience for everyone. Please be respectful in all interactions.

## How to Contribute

### Reporting Issues

1. **Search existing issues** first to avoid duplicates
2. **Use the issue template** when creating new issues
3. **Provide clear reproduction steps** for bugs
4. **Include relevant system information** (Go version, PostgreSQL version, PostGIS version)

### Suggesting Features

1. **Check the roadmap** to see if the feature is already planned
2. **Open an issue** with the "enhancement" label
3. **Describe the use case** and why the feature would be valuable
4. **Consider implementation complexity** and backward compatibility

### Submitting Code Changes

1. **Fork the repository** and create a feature branch
2. **Write tests** for your changes
3. **Follow the coding standards** outlined below
4. **Update documentation** if needed
5. **Submit a pull request** with a clear description

## Development Setup

### Prerequisites

- Go 1.19 or later
- PostgreSQL 12+ with PostGIS 3.0+
- Git

### Local Development

1. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/gogis.git
   cd gogis
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Set up test database**:
   ```sql
   CREATE DATABASE gogis_test;
   \c gogis_test;
   CREATE EXTENSION postgis;
   ```

4. **Set environment variables**:
   ```bash
   export TEST_DATABASE_URL="host=localhost user=postgres dbname=gogis_test sslmode=disable"
   ```

5. **Run tests**:
   ```bash
   go test -v ./...
   ```

## Coding Standards

### Go Code Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format code
- Use meaningful variable names
- Add comments for exported functions and types
- Keep functions focused and small

### Documentation

- **GoDoc comments** for all exported types and functions
- **Usage examples** in comments when helpful
- **Update README.md** for significant changes
- **Include code examples** in documentation

### Testing

- **Unit tests** for all public functions
- **Table-driven tests** for multiple test cases
- **Integration tests** for database operations (with build tags)
- **Test edge cases** and error conditions
- **Aim for high test coverage** (>90%)

### Example Test Structure

```go
func TestPointString(t *testing.T) {
    tests := []struct {
        name     string
        point    Point
        expected string
    }{
        {
            name:     "positive coordinates",
            point:    Point{Lng: -74.0445, Lat: 40.6892},
            expected: "SRID=4326;POINT(-74.0445 40.6892)",
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.point.String()
            if result != tt.expected {
                t.Errorf("Point.String() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Pull Request Process

### Before Submitting

1. **Rebase your branch** on the latest main
2. **Run all tests** and ensure they pass
3. **Run linters** (golint, govet)
4. **Update documentation** if needed
5. **Add yourself** to CONTRIBUTORS.md

### PR Guidelines

1. **Use a descriptive title** that explains the change
2. **Reference related issues** in the description
3. **Explain the motivation** for the change
4. **List any breaking changes** clearly
5. **Include screenshots** for UI changes (if applicable)

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] All existing tests pass
- [ ] New tests added for new functionality
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings or errors
```

## Architecture Guidelines

### Adding New Geometry Types

When adding new PostGIS geometry types:

1. **Create new file** named `{type}.go`
2. **Implement required interfaces**: `Geometry`, `sql.Scanner`, `driver.Valuer`
3. **Add comprehensive documentation**
4. **Include WKB parsing** with both endianness support
5. **Add to GeometryCollection** if applicable
6. **Write complete test suite**
7. **Add usage examples**

### File Organization

```
gogis/
├── doc.go                    # Package documentation
├── point.go                  # Point implementation
├── linestring.go            # LineString implementation  
├── polygon.go               # Polygon implementation
├── geometry_collection.go   # GeometryCollection implementation
├── point_test.go            # Point tests
├── linestring_test.go       # LineString tests
├── polygon_test.go          # Polygon tests
├── geometry_collection_test.go # GeometryCollection tests
├── examples/                # Usage examples
│   ├── README.md
│   ├── basic_usage.go
│   ├── linestring_example.go
│   ├── polygon_example.go
│   └── geometry_collection_example.go
├── INDEXING.md             # Performance guide
├── CONTRIBUTING.md         # This file
└── README.md               # Main documentation
```

### Interface Design

All geometry types must implement:

```go
type Geometry interface {
    String() string  // WKT representation
}

// Plus sql.Scanner and driver.Valuer from database/sql
```

### Error Handling

- **Return descriptive errors** with context
- **Use fmt.Errorf** for error wrapping
- **Validate input parameters** before processing
- **Handle nil values** gracefully

## Testing Strategy

### Unit Tests

- Test all public methods
- Test error conditions
- Test edge cases (empty geometries, nil values)
- Use table-driven tests for multiple scenarios

### Integration Tests

- Test with real PostGIS database
- Use build tags: `// +build integration`
- Test WKB parsing with actual PostGIS output
- Test complex spatial queries

### Performance Tests

- Benchmark critical operations
- Test with realistic data sizes
- Compare with/without spatial indexes

## Documentation Requirements

### Code Documentation

Every exported type and function must have:

```go
// Point represents a geometric point in 2D space.
//
// Point implements the PostGIS Point geometry type and can be used in GORM models
// to store geographic locations. It uses the WGS 84 coordinate system (SRID 4326).
//
// Example:
//   point := Point{Lng: -74.0445, Lat: 40.6892}
//   fmt.Println(point.String()) // "SRID=4326;POINT(-74.0445 40.6892)"
type Point struct {
    Lng float64 `json:"lng"` // Longitude in decimal degrees
    Lat float64 `json:"lat"` // Latitude in decimal degrees
}
```

### README Updates

When adding features, update:
- Feature list
- Usage examples
- API documentation
- Migration notes (for breaking changes)

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Release Checklist

1. Update version in relevant files
2. Update CHANGELOG.md
3. Tag the release
4. Update documentation
5. Announce the release

## Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Email**: For security issues (private)

## Recognition

Contributors will be:
- Added to CONTRIBUTORS.md
- Mentioned in release notes
- Credited in documentation (for significant contributions)

Thank you for contributing to GoGIS!