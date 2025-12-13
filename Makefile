BIN_DIR := $(HOME)/.local/bin
APP     := wslbridge

build:
	go build -o bin/$(APP) ./cmd/$(APP)

install: build
	mkdir -p $(BIN_DIR)
	install -m 755 bin/$(APP) $(BIN_DIR)/$(APP)

run:
	go run ./cmd/$(APP)