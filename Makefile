build:
	go build ./cmd/mforge

test:
	go test ./...

fmt:
	gofmt -w .
