BINARY := boxship
OUTPUT_DIR := _output
IMAGE_REPO ?= quay.io/petr-muller/boxship

.PHONY: build test vet verify image clean

build:
	go build -o $(OUTPUT_DIR)/$(BINARY) ./cmd/boxship/

test:
	go test ./...

vet:
	go vet ./...

verify: vet test

image: build
	docker build -t $(IMAGE_REPO):latest -f images/boxship/Dockerfile $(OUTPUT_DIR)/

clean:
	rm -rf $(OUTPUT_DIR)/
