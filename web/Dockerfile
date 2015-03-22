FROM golang:1.4

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun

ENV PORT 3000
WORKDIR /go/src/github.com/convox/kernel/web
COPY . /go/src/github.com/convox/kernel/web
RUN go get .

ENTRYPOINT ["init"]
CMD ["bin/web"]
