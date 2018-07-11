FROM golang:1.10

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

COPY . $GOPATH/src/github.com/convox/rack

RUN go install github.com/convox/rack/cmd/build
RUN env CGO_ENABLED=0 go install --ldflags '-extldflags "-static"' github.com/convox/rack/cmd/convox-env
