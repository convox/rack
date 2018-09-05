FROM golang:1.11 AS development

RUN curl -Ls https://github.com/krallin/tini/releases/download/v0.18.0/tini -o /tini && chmod +x /tini
ENTRYPOINT ["/tini", "--"]

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN curl -Ls https://github.com/mattgreen/watchexec/releases/download/1.8.6/watchexec-1.8.6-x86_64-unknown-linux-gnu.tar.gz | \
    tar -C /usr/bin --strip-components 1 -xz

ENV DEVELOPMENT=true

WORKDIR /go/src/github.com/convox/rack

COPY . .

RUN env CGO_ENABLED=0 go install --ldflags '-extldflags "-static"' ./cmd/convox-env
RUN go install ./cmd/...
RUN go install .

CMD ["bin/web"]

FROM debian:stretch

RUN apt-get -qq update && apt-get -qq -y install curl

RUN curl -Ls https://github.com/krallin/tini/releases/download/v0.18.0/tini -o /tini && chmod +x /tini
ENTRYPOINT ["/tini", "--"]

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

ENV DEVELOPMENT=false
ENV GOPATH=/go
ENV PATH=$PATH:/app/bin

WORKDIR /app

# binaries
COPY --from=development /go/bin/build bin/
COPY --from=development /go/bin/convox-env bin/
COPY --from=development /go/bin/monitor bin/
COPY --from=development /go/bin/rack bin/

# aws templates
COPY --from=development /go/src/github.com/convox/rack/provider/aws/formation provider/aws/
COPY --from=development /go/src/github.com/convox/rack/provider/aws/templates provider/aws/

CMD ["/app/bin/rack"]
