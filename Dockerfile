## development #################################################################

FROM golang:1.11 AS development

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.11.0/bin/linux/amd64/kubectl -o /usr/bin/kubectl && \
    chmod +x /usr/bin/kubectl

RUN curl -Ls https://github.com/mattgreen/watchexec/releases/download/1.8.6/watchexec-1.8.6-x86_64-unknown-linux-gnu.tar.gz | \
    tar -C /usr/bin --strip-components 1 -xz

ENV DEVELOPMENT=true

WORKDIR /go/src/github.com/convox/rack

COPY . .
RUN make build

CMD ["bin/web"]

## package #####################################################################

FROM golang:1.11 AS package

WORKDIR /go/src/github.com/convox/rack

COPY --from=development /go/src/github.com/convox/rack .
RUN make package build

## production ##################################################################

FROM debian:stretch

RUN apt-get -qq update && apt-get -qq -y install curl

RUN curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz | \
    tar -C /usr/bin --strip-components 1 -xz

RUN curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.11.0/bin/linux/amd64/kubectl -o /usr/bin/kubectl && \
    chmod +x /usr/bin/kubectl

ENV DEVELOPMENT=false
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin

WORKDIR /rack

COPY --from=package /go/bin/build /go/bin/
COPY --from=package /go/bin/convox-env /go/bin/
COPY --from=package /go/bin/monitor /go/bin/
COPY --from=package /go/bin/rack /go/bin/
COPY --from=package /go/bin/router /go/bin/

# aws templates
COPY --from=development /go/src/github.com/convox/rack/provider/aws/formation/ provider/aws/formation/
COPY --from=development /go/src/github.com/convox/rack/provider/aws/templates/ provider/aws/templates/

# k8s templates
# COPY --from=development /go/src/github.com/convox/rack/provider/k8s/template/ provider/k8s/template/

CMD ["/go/bin/rack"]
