build:
	go build -o ./build/ ./...

start:
	${MAKE} build
	./build/contractMonitor ./config.toml

.PHONY: build start