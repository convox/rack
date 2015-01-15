FROM golang:1.4

RUN go get github.com/codegangsta/gin

EXPOSE 5000
ENV PORT 5000

ADD . /go/src/github.com/ddollar/convox/kernel
WORKDIR /go/src/github.com/ddollar/convox/kernel
RUN go get .

CMD ["bin/web"]
