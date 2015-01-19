FROM golang:1.4

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install unzip

RUN curl -L https://dl.bintray.com/mitchellh/packer/packer_0.7.5_linux_amd64.zip -o /tmp/packer.zip
RUN unzip /tmp/packer.zip -d /usr/local/bin

ADD . /go/src/github.com/convox/builder
WORKDIR /go/src/github.com/convox/builder
RUN go get .

ENTRYPOINT ["builder"]
