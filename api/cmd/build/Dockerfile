FROM golang:1.9-alpine

RUN apk update && apk add docker openssh

COPY . $GOPATH/src/github.com/convox/rack
RUN go install github.com/convox/rack/api/cmd/build
