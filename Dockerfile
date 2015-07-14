FROM convox/alpine:3.1

RUN apk-install docker git

COPY pkg/haproxy-1.5.10-r0.apk /tmp/haproxy-1.5.10-r0.apk
RUN apk add --allow-untrusted /tmp/haproxy-1.5.10-r0.apk
RUN rm /tmp/haproxy-1.5.10-r0.apk
COPY data/haproxy.cfg /etc/haproxy/haproxy.cfg

RUN apk-install make python py-pip zip
RUN pip install awscli

RUN apk-install go
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun

ENV PORT 3000
WORKDIR /go/src/github.com/convox/kernel
COPY . /go/src/github.com/convox/kernel
RUN go get .

ENTRYPOINT ["/go/bin/init"]
CMD ["bin/web"]
