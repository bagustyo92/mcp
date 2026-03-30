.PHONY: build run test clean copy-config

BINARY_NAME=pr-maker-mcp
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

run: build
	$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

copy-config:
	@if [ ! -f config.yaml ]; then \
		cp config.example.yaml config.yaml; \
		echo "Created config.yaml from config.example.yaml — fill in your credentials."; \
	else \
		echo "config.yaml already exists, skipping."; \
	fi
