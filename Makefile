.PHONY: audit
audit:
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Linting code...'
	golangci-lint run -c .golangci.yaml

.PHONY: clean
clean:
	go version
	rm -rf target
	rm -rf vendor
	go mod tidy

.PHONY: test
test:
	go mod vendor
	go fmt ./...
	go build ./...
	go test -race ./...

.PHONY: build
build: clean test
	mvn clean install