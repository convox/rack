FROM golang:1.4

RUN curl -O https://test.docker.com/builds/Linux/x86_64/docker-1.7.1 && \
    chmod +x docker-1.7.1 && \
    mv docker-1.7.1 /usr/local/bin/docker

RUN go get github.com/ddollar/rerun

WORKDIR /go/src/github.com/convox/agent
COPY . /go/src/github.com/convox/agent
RUN go get .

ENV DOCKER_HOST unix:///var/run/docker.sock

ENTRYPOINT ["agent"]
