BINARY := ship-hook
OUTPUT_DIR := _output
IMAGE_REPO ?= quay.io/petr-muller/ship-hook

.PHONY: build test integration-test vet verify image clean dev-server dev-webhook dev-state dev-reset dev-watch

build:
	go build -o $(OUTPUT_DIR)/$(BINARY) ./cmd/ship-hook/

test:
	go test ./...

integration-test:
	go test -tags=integration ./test/integration/...

vet:
	go vet ./...

verify: vet test

image: build
	docker build -t $(IMAGE_REPO):latest -f images/ship-hook/Dockerfile $(OUTPUT_DIR)/

clean:
	rm -rf $(OUTPUT_DIR)/

dev-server:
	go run ./cmd/devserver/ --port=8888 --state-port=8889

dev-webhook:
	go run ./cmd/devwebhook/ --address=http://localhost:8888/hook \
		--event=$(EVENT) --payload=$(PAYLOAD)

dev-state:
	@curl -s http://localhost:8889/state | jq .

dev-reset:
	@curl -s -XPOST http://localhost:8889/reset

dev-watch:
	watchexec -r -e go -- go run ./cmd/devserver/ --port=8888 --state-port=8889
