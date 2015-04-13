FROM convox/alpine:3.1

RUN apk-install git go

ENV GOPATH /go
ENV GOBIN $GOPATH/bin
ENV PATH $GOBIN:$PATH

RUN go get -u github.com/jteeuwen/go-bindata/...

WORKDIR /go/src/github.com/convox/service
COPY . /go/src/github.com/convox/service
RUN go get .
RUN go-bindata template/

ENTRYPOINT ["/go/bin/service"]
