FROM gliderlabs/alpine:3.1

RUN apk-install docker git

RUN apk-install go
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun

ENV PORT 3000
WORKDIR /go/src/github.com/convox/kernel/web
COPY . /go/src/github.com/convox/kernel/web
RUN go get .

ENTRYPOINT ["/go/bin/init"]
CMD ["bin/web"]
