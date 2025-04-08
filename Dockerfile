## package #####################################################################

FROM golang:1.23-bookworm AS package

# Add backports to get upx-ucl in Bookworm
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/backports.list && \
    apt-get update && apt-get install -y -t bookworm-backports upx-ucl

WORKDIR /go/src/github.com/convox/rack

COPY . /go/src/github.com/convox/rack
RUN make build compress

## production ##################################################################

FROM ubuntu:22.04

ARG DOCKER_ARCH=x86_64
ARG KUBECTL_ARCH=amd64

RUN echo "$(uname -a)"
RUN apt-get -qq update && apt-get -qq -y install curl

ENV DOCKER_BUILDKIT=1

RUN curl -s https://download.docker.com/linux/static/stable/${DOCKER_ARCH}/docker-18.09.9.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/${KUBECTL_ARCH}/kubectl -o /usr/bin/kubectl && \
    chmod +x /usr/bin/kubectl

ENV DEVELOPMENT=false
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin

WORKDIR /rack

COPY --from=package /go/bin/build /go/bin/
COPY --from=package /go/bin/convox-env /go/bin/
COPY --from=package /go/bin/monitor /go/bin/
COPY --from=package /go/bin/rack /go/bin/

# aws templates
COPY --from=package /go/src/github.com/convox/rack/provider/aws/formation/ provider/aws/formation/
COPY --from=package /go/src/github.com/convox/rack/provider/aws/templates/ provider/aws/templates/

CMD ["/go/bin/rack"]
