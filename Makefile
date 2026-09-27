.PHONY: test lint fmt act ci-go

GO_DIR := go/sscachian

fmt:
	cd $(GO_DIR) && gofmt -w .

lint:
	cd $(GO_DIR) && test -z "$$(gofmt -l .)" && go vet ./...

test:
	cd $(GO_DIR) && go test ./...

ci-go: lint test

# Wiring smoke for CI (requires Docker + act image pull).
act:
	act pull_request -W .github/workflows/ci.yml
