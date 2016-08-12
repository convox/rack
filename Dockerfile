FROM convox/golang

RUN apt-get update && apt-get install -y docker haproxy openssl python

RUN go get github.com/ddollar/init
RUN go get github.com/convox/rerun
RUN go get github.com/convox/cfssl/cmd/cfssl

COPY conf/haproxy.cfg /etc/haproxy/haproxy.cfg

ENV PORT 3000
WORKDIR /go/src/github.com/convox/rack
COPY . /go/src/github.com/convox/rack
RUN go install ./...

ENTRYPOINT ["/go/bin/init"]
CMD ["api/bin/web"]
