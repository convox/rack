FROM golang:1.4

RUN go get github.com/codegangsta/gin

EXPOSE 5000
ENV PORT 5000

COPY . /go/src/github.com/convox/kernel/web
WORKDIR /go/src/github.com/convox/kernel/web
RUN go get .

CMD ["bin/web"]
