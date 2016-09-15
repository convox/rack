FROM golang:1.7-alpine

COPY . $GOPATH/src/github.com/convox/rack
RUN go install github.com/convox/rack/api/cmd/build
