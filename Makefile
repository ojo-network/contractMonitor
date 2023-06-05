IMAGE_NAME := relayer-monitor
DOCKERFILE := Dockerfile

.PHONY: docker-build

docker-build:
	docker build -t $(IMAGE_NAME) -f $(DOCKERFILE) .

docker-run:
	${MAKE} docker-build
	docker run --env-file .env ${IMAGE_NAME}

build:
	go build -o ./build/monitor .