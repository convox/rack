FROM gliderlabs/alpine:3.2

RUN apk-install curl docker go git python py-setuptools zip

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN git clone https://github.com/docker/compose /tmp/compose
WORKDIR /tmp/compose
RUN python setup.py install

RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/convox/rack/manifest

COPY . /go/src/github.com/convox/rack/cmd/build
WORKDIR /go/src/github.com/convox/rack/cmd/build
RUN go-bindata data/
RUN go get .

ENTRYPOINT ["/go/src/github.com/convox/rack/cmd/build/bin/entrypoint"]
