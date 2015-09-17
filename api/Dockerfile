FROM gliderlabs/alpine:3.2

RUN apk-install docker git python

COPY pkg/haproxy-1.5.10-r0.apk /tmp/haproxy-1.5.10-r0.apk
RUN apk add --allow-untrusted /tmp/haproxy-1.5.10-r0.apk
RUN rm /tmp/haproxy-1.5.10-r0.apk
COPY conf/haproxy.cfg /etc/haproxy/haproxy.cfg

RUN apk-install go
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun
RUN go get -u github.com/jteeuwen/go-bindata/...

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack/api
COPY . /go/src/github.com/convox/rack/api
RUN go get ./...
RUN go-bindata -pkg=models -prefix=models -o=models/templates.go models/templates/...

ENTRYPOINT ["/go/bin/init"]
CMD ["bin/web"]
