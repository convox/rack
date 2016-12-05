FROM golang:1.7-alpine

RUN apk update && apk add build-base docker git haproxy openssh openssl python tar

# add nobody to the docker group
RUN addgroup nobody docker

# need a real pid 1 for signal handling, zombie reaping, etc
ADD http://convox-binaries.s3.amazonaws.com/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

RUN go get github.com/convox/rerun

RUN mkdir -p /etc/ssl/convox
RUN chown -R nobody:nogroup /etc/ssl/convox
RUN chown -R nobody:nogroup /var/lib/haproxy

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./api
RUN go install ./api/cmd/build
RUN go install ./api/cmd/monitor

RUN chown -R nobody:nogroup /go/src/github.com/convox/rack

USER nobody

CMD ["api/bin/web"]
