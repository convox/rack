FROM gliderlabs/alpine:3.1

RUN apk-install curl docker go git python py-setuptools zip

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN git clone https://github.com/docker/compose /tmp/compose
WORKDIR /tmp/compose
RUN python setup.py install

RUN go get github.com/jteeuwen/go-bindata/...

COPY . /go/src/github.com/convox/build
WORKDIR /go/src/github.com/convox/build
RUN go-bindata data/
RUN go get .

ENTRYPOINT ["/go/src/github.com/convox/build/bin/entrypoint"]
