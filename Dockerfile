FROM convox/alpine:3.1

RUN apk-install curl go git zip

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN curl -L https://dl.bintray.com/mitchellh/packer/packer_0.7.5_linux_amd64.zip -o /tmp/packer.zip
RUN mkdir -p /tmp/packer
RUN unzip /tmp/packer.zip -d /tmp/packer/
RUN cp /tmp/packer/packer /usr/local/bin/
RUN cp /tmp/packer/packer-builder-amazon-ebs /usr/local/bin/
RUN cp /tmp/packer/packer-provisioner-file /usr/local/bin/
RUN cp /tmp/packer/packer-provisioner-shell /usr/local/bin/
RUN rm -rf /tmp/packer*

RUN go get github.com/jteeuwen/go-bindata/...

COPY . /go/src/github.com/convox/build
WORKDIR /go/src/github.com/convox/build
RUN go-bindata data/
RUN go get .

ENTRYPOINT ["build"]
