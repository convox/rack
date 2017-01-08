FROM golang:1.7.3-alpine

RUN apk update && apk add build-base docker git haproxy openssh openssl python tar

# need a real pid 1 for signal handling, zombie reaping, etc
ADD http://convox-binaries.s3.amazonaws.com/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

RUN go get github.com/convox/rerun

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./api
RUN go install ./api/cmd/build
RUN go install ./api/cmd/monitor

CMD ["api/bin/web"]
