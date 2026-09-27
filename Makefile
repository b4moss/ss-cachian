.PHONY: test lint fmt act ci-go emulator lint-node test-node

GO_DIR := go/sscachian
NODE_DIR := node/sscachian

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

lint-node:
	cd $(NODE_DIR) && npm ci && npm run lint

test-node:
	cd $(NODE_DIR) && npm ci && npm test

# Wiring smoke for CI (requires Docker + act image pull).
act:
	act pull_request -W .github/workflows/ci.yml
