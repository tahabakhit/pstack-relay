.PHONY: build test check package

build:
	mkdir -p bin
	go build -trimpath -o bin/pstack-rex ./cmd/pstack-rex

test:
	go test -race ./...

check: test
	go vet ./...
	go run ./cmd/upstream check

package:
	bash scripts/package.sh
