FROM golang:1.9-alpine

RUN apk update && apk add build-base curl docker git haproxy openssh openssl python tar

RUN go get -u github.com/convox/rerun

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./api
RUN go install ./api/cmd/...
RUN env CGO_ENABLED=0 go install --ldflags '-extldflags "-static"' github.com/convox/rack/api/cmd/convox-env

CMD ["api/bin/web"]
