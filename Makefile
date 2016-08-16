tag := $(shell git name-rev --tags --always --name-only HEAD)
ifeq ($(tag),undefined)
tag := $(shell git rev-parse --short HEAD)
endif
# Check if we're working with a dirty tree
ifneq ($(shell git diff --shortstat),)
tag := $(tag)-dirty
endif
docker_username := $(shell docker info | grep Username | sed 's/Username: //')

all: build.go build.docker

build.go:
	GOOS=linux GOARCH=amd64 go build -o bin/terminator cmd/terminator/main.go

build.docker:
	docker build -t $(docker_username)/docker-cloud-terminator:$(tag) .
	@echo Docker image: $(docker_username)/docker-cloud-terminator:$(tag)

run:
	docker run --rm -it \
		-e DOCKERCLOUD_AUTH="$(DOCKERCLOUD_AUTH)" \
		-e POLLING_INTERVAL='1s' \
		-e AWS_REGION='us-east-1' \
		-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		$(docker_username)/docker-cloud-terminator:$(tag)