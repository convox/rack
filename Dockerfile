FROM golang:1.6.3-alpine

RUN apk update && apk add build-base docker git haproxy openssh openssl python tar

RUN go get github.com/ddollar/init
RUN go get github.com/convox/rerun

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./cmd/build
RUN go install ./server

ENTRYPOINT ["/go/bin/init"]
CMD ["api/bin/web"]
