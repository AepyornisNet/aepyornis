.PHONY: test test-coverage view-coverage fmt fmt-check vet report-check

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	@echo "Coverage report generated: coverage.out"

	@echo "To view the coverage report, run:"
	@echo "make view-coverage"

# View coverage report in browser
view-coverage:
	@echo "Opening coverage report in browser..."
	@go tool cover -html=coverage.out -o coverage.html
	@open coverage.html || xdg-open coverage.html || start coverage.html

fmt:
	go fmt ./...
	gofmt -s -w .

fmt-check:
	@echo "Checking code formatting..."
	@if [ "$$(gofmt -s -d . | wc -l)" -eq 0 ]; then \
		echo "✓ All files are properly formatted"; \
	else \
		echo "✗ Some files need formatting:"; \
		gofmt -s -d .; \
		echo "Run 'make fmt' to fix formatting issues"; \
		exit 1; \
	fi

# Run go vet
vet:
	go vet ./...

# Check for Go Report Card issues
report-check:
	@echo "=== Go Report Card Quality Check ==="
	@echo ""
	@echo "1. Checking gofmt..."
	@make fmt-check
	@echo ""
	@echo "2. Checking go vet..."
	@go vet ./...
	@echo "✓ go vet passed"
	@echo ""
	@echo "3. Checking gocyclo (if available)..."
	@if command -v gocyclo >/dev/null 2>&1; then \
		gocyclo -over 15 .; \
		if [ $$? -eq 0 ]; then echo "✓ gocyclo passed"; fi; \
	else \
		echo "⚠ gocyclo not installed (optional)"; \
	fi
	@echo ""
	@echo "4. Checking ineffassign (if available)..."
	@if command -v ineffassign >/dev/null 2>&1; then \
		ineffassign .; \
		if [ $$? -eq 0 ]; then echo "✓ ineffassign passed"; fi; \
	else \
		echo "⚠ ineffassign not installed (optional)"; \
	fi
	@echo ""
	@echo "5. Checking misspell (if available)..."
	@if command -v misspell >/dev/null 2>&1; then \
		misspell -error .; \
		if [ $$? -eq 0 ]; then echo "✓ misspell passed"; fi; \
	else \
		echo "⚠ misspell not installed (optional)"; \
	fi
	@echo ""
	@echo "✓ Go Report Card check completed!"