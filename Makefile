build:
	go build -o ./build/ ./...

start:
	${MAKE} build
	./contractMonitor ./config.toml

.PHONY: build start