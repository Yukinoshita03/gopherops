SERVICES := identity platform agent-worker cluster-tools

.PHONY: fmt check build
fmt:
	gofmt -w cmd internal

check:
	go test ./...
	go vet ./...

build:
	@mkdir -p bin
	@set -e; for service in $(SERVICES); do go build -o bin/$$service ./cmd/$$service; done
