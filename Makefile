build:
	go build ./cmd/mf

test:
	go test ./...

fmt:
	gofmt -w .
