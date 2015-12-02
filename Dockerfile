FROM gliderlabs/alpine:3.2

RUN apk-install docker git go haproxy python

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN apk-install gcc libc-dev libtool libgcc
RUN go get github.com/cloudflare/cfssl/cmd/cfssl

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun

COPY conf/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack
RUN go install ./...

ENTRYPOINT ["/go/bin/init"]
CMD ["api/bin/web"]
