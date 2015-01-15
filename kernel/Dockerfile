FROM golang:1.3

RUN go get github.com/codegangsta/gin

EXPOSE 5000
ENV PORT 5000

ADD . /go/src/github.com/ddollar/convox/kernel
WORKDIR /go/src/github.com/ddollar/convox/kernel

CMD ["bin/web"]
