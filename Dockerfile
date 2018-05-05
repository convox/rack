FROM golang:1.10

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN curl -Ls https://github.com/mattgreen/watchexec/releases/download/1.8.6/watchexec-1.8.6-x86_64-unknown-linux-gnu.tar.gz | \
    tar -C /usr/bin --strip-components 1 -xz

WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./...

RUN env CGO_ENABLED=0 go install --ldflags '-extldflags "-static"' github.com/convox/rack/cmd/convox-env

CMD ["bin/web"]
