FROM gliderlabs/alpine:edge

RUN apk-install docker git haproxy

RUN apk-install go
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun

ENV PORT 3000
WORKDIR /go/src/github.com/convox/kernel
COPY . /go/src/github.com/convox/kernel
RUN go get .

COPY data/haproxy.cfg /etc/haproxy/haproxy.cfg

ENTRYPOINT ["/go/bin/init"]
CMD ["bin/web"]
