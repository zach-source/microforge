build:
	go build ./cmd/mforge

test:
	go test ./...

fmt:
	gofmt -w .

test-e2e-claude:
	scripts/integration/claude_turn_test.sh
