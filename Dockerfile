FROM golang:1.4

WORKDIR /go/src/github.com/convox/agent
COPY . /go/src/github.com/convox/agent
RUN go get .

ENV DOCKER_HOST unix:///var/run/docker.sock

ENTRYPOINT ["agent"]
