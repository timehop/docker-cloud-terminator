FROM alpine:3.4
MAINTAINER Timehop <tech@timehop.com>

# WHAT
RUN apk --update add ca-certificates

# If you provide your service with "API Full Access" this env var will be set
# for you by Docker Cloud. Elsewhere you must override it yourself. From the
# docs: All requests should be sent to https://cloud.docker.com/ endpoint
# using basic authentication using your API key as password. Example usage:
# $ export DOCKERCLOUD_AUTH="Basic $(echo -n "<username>:<api_key>" | base64)'"
# $ curl -v -H "Authorization: $DOCKERCLOUD_AUTH" -H "Accept: application/json" https://cloud.docker.com/api/app/v1/service/
ENV DOCKERCLOUD_AUTH please_set_me

# The polling interval is how frequently the Docker Cloud API will be
# invoked to discover unreachable nodes.
ENV POLLING_INTERVAL 5s

# A comma-separated list of cloud providers to be targeted for host termination. Defaults to 'aws'.
ENV CLOUD_PROVIDERS aws

# Before building this container run:
# GOOS=linux GOARCH=amd64 go build -o terminator cmd/terminator/main.go
ADD bin/terminator /usr/bin/terminator
CMD terminator