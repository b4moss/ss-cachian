.PHONY: test lint fmt act ci-go emulator

GO_DIR := go/sscachian

fmt:
	cd $(GO_DIR) && gofmt -w .

lint:
	cd $(GO_DIR) && test -z "$$(gofmt -l .)" && go vet ./...

emulator:
	bash scripts/start-firestore-emulator.sh

test: emulator
	FIRESTORE_EMULATOR_HOST=$${FIRESTORE_EMULATOR_HOST:-127.0.0.1:8080} \
		cd $(GO_DIR) && go test ./...

ci-go: lint test

# Wiring smoke for CI (requires Docker + act image pull).
act:
	act pull_request -W .github/workflows/ci.yml
