FROM golang:1.4

RUN go get github.com/ddollar/init

WORKDIR /go/src/github.com/convox/agent
COPY . /go/src/github.com/convox/agent
RUN go get .

ENTRYPOINT ["init", "agent"]
