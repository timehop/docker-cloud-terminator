tag := $(shell git name-rev --tags --always --name-only HEAD)
ifeq ($(tag),undefined)
tag := $(shell git rev-parse --short HEAD)
endif
# Check if we're working with a dirty tree
ifneq ($(shell git diff --shortstat),)
tag := $(tag)-dirty
endif

all: build

build: docker.build

go.build:
	GOOS=linux GOARCH=amd64 go build -o bin/terminator cmd/terminator/main.go

docker.build: go.build
	docker build -t timehop/docker-cloud-terminator:$(tag) .
	docker tag timehop/docker-cloud-terminator:$(tag) timehop/docker-cloud-terminator:latest
	@echo Docker image: timehop/docker-cloud-terminator:$(tag)

docker.push:
	docker push timehop/docker-cloud-terminator:$(tag)
	docker push timehop/docker-cloud-terminator:latest

run: build
	docker run --rm -it \
		-e DOCKERCLOUD_AUTH="$(DOCKERCLOUD_AUTH)" \
		-e POLLING_INTERVAL='1s' \
		-e AWS_REGION='us-east-1' \
		-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		timehop/docker-cloud-terminator:$(tag)