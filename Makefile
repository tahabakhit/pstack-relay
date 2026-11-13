.PHONY: build test check package plugin plugin-check

build:
	mkdir -p bin
	go build -trimpath -o bin/pstack-rex ./cmd/pstack-rex

test:
	go test -race ./...

check: test plugin-check
	go vet ./...
	go run ./cmd/upstream sync --check

package:
	bash scripts/package.sh

plugin:
	go run ./cmd/pstack-rex plugin build

plugin-check:
	go run ./cmd/pstack-rex plugin build --check
