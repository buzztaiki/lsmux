default: fmt check test

.PHONY: check
check:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: fmt
fmt:
	go fmt ./...
