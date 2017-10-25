FROM golang:1.9-alpine

RUN apk update && apk add \
    build-base \
    curl \
    git \
    haproxy \
    openssh \
    openssl \
    python \
    tar

RUN curl -fsSLO https://get.docker.com/builds/Linux/x86_64/docker-1.13.0.tgz \
    && tar --strip-components=1 -xvzf docker-1.13.0.tgz -C /usr/local/bin

RUN go get -u github.com/convox/rerun

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./api
RUN go install ./api/cmd/build
RUN go install ./api/cmd/monitor

CMD ["api/bin/web"]
