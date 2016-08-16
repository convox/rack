FROM golang:1.7-alpine

RUN apk update && apk add build-base docker git haproxy openssh openssl python tar

RUN go get github.com/ddollar/init
RUN go get github.com/convox/rerun

COPY dist/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack

RUN go install ./api
RUN go install ./api/cmd/monitor
RUN go install ./cmd/build

ENTRYPOINT ["/go/bin/init"]
CMD ["api/bin/web"]
