build:
	go build -o ./ ./...

start:
	${MAKE} build
	./contractMonitor ./config.toml