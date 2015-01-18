FROM golang:1.4

RUN go get github.com/codegangsta/gin

EXPOSE 5000
ENV PORT 5000

ADD . /go/src/github.com/convox/convox/builder
WORKDIR /go/src/github.com/convox/convox/builder
RUN go get .

CMD ["bin/web"]
