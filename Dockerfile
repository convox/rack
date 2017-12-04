FROM golang:1.9-alpine

RUN apk add --no-cache build-base curl git haproxy openssh openssl python tar

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-17.09.0-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN go get -u github.com/convox/rerun

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./...

RUN env CGO_ENABLED=0 go install --ldflags '-extldflags "-static"' github.com/convox/rack/cmd/convox-env

CMD ["bin/web"]
