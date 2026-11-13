.PHONY: build test check package plugin plugin-check

build: plugin

test:
	go test -race ./...

check: test plugin-check
	go vet ./...
	shellcheck -s sh overlay/bin/pstack-relay plugins/pstack-relay/bin/pstack-relay overlay/hooks/session-start.sh
	go run ./cmd/upstream sync --check

package:
	bash scripts/package.sh

plugin:
	go run ./cmd/build-plugin

plugin-check:
	go run ./cmd/build-plugin --check
