build:
	go build -o ./ ./...

start:
	${MAKE} build
	./relayerMonitor ./config.toml