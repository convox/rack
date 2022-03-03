## test ########################################################################

FROM golang:1.16 AS test

ARG DOCKER_ARCH=x86_64
ARG KUBECTL_ARCH=amd64

RUN curl -s https://download.docker.com/linux/static/stable/${DOCKER_ARCH}/docker-18.09.9.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/${KUBECTL_ARCH}/kubectl -o /usr/bin/kubectl && \
    chmod +x /usr/bin/kubectl

RUN curl -Ls https://github.com/mattgreen/watchexec/releases/download/1.8.6/watchexec-1.8.6-x86_64-unknown-linux-gnu.tar.gz | \
    tar -C /usr/bin --strip-components 1 -xz

WORKDIR /go/src/github.com/convox/rack

COPY . .

RUN go install --ldflags="-s -w" ./vendor/...


## development #################################################################

FROM test AS development

# RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.09.6.tgz | \
#     tar -C /usr/bin --strip-components 1 -xz

# RUN curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl -o /usr/bin/kubectl && \
#     chmod +x /usr/bin/kubectl

# RUN curl -Ls https://github.com/mattgreen/watchexec/releases/download/1.8.6/watchexec-1.8.6-x86_64-unknown-linux-gnu.tar.gz | \
#     tar -C /usr/bin --strip-components 1 -xz

ENV DEVELOPMENT=true

# WORKDIR /go/src/github.com/convox/rack

# COPY vendor vendor
# RUN go install --ldflags="-s -w" ./vendor/...

# COPY . .
RUN make build

CMD ["bin/web"]

## package #####################################################################

FROM golang:1.16 AS package

RUN apt-get update && apt-get -y install upx-ucl

RUN go get -u github.com/gobuffalo/packr/packr

WORKDIR /go/src/github.com/convox/rack

COPY --from=development /go/src/github.com/convox/rack .
RUN make package build compress

## production ##################################################################

FROM ubuntu:18.04

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

COPY --from=package /go/bin/atom /go/bin/
COPY --from=package /go/bin/build /go/bin/
COPY --from=package /go/bin/convox-env /go/bin/
COPY --from=package /go/bin/monitor /go/bin/
COPY --from=package /go/bin/rack /go/bin/
COPY --from=package /go/bin/router /go/bin/

# aws templates
COPY --from=development /go/src/github.com/convox/rack/provider/aws/formation/ provider/aws/formation/
COPY --from=development /go/src/github.com/convox/rack/provider/aws/templates/ provider/aws/templates/

CMD ["/go/bin/rack"]
