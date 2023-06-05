BUILD_DIR ?= $(CURDIR)/build

build: go.sum
	CGO_ENABLED=0 go build -mod=readonly -o $(BUILD_DIR)/monitor ./...