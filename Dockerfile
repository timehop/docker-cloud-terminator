# Before building this container run:
# GOOS=linux GOARCH=amd64 go build -o terminator cmd/terminator/main.go
FROM alpine:3.4
MAINTAINER Timehop <tech@timehop.com>

# From the docs:
# All requests should be sent to https://cloud.docker.com/ endpoint using Basic authentication using your API key as password
# Example api usage:
# curl -v -H "Authorization: $DOCKERCLOUD_AUTH" -H "Accept: application/json" https://cloud.docker.com/api/app/v1/service/

# If you provide your service with "API Full Access" this
# env var will be set by Docker Cloud on your behalf. In development
# you must override it yourself.
ENV DOCKERCLOUD_AUTH 'Must be overriden! E.g.: DOCKERCLOUD_AUTH="Basic `echo -n "<user>:<pass>" | base64`'"

ADD ./terminator /usr/bin/terminator
RUN mkdir /terminator-callbacks
CMD terminator