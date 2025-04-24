FROM golang:1.23-bookworm AS package

# Add backports to get upx-ucl in Bookworm
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/backports.list && \
    apt-get update && apt-get install -y -t bookworm-backports upx-ucl

WORKDIR /go/src/github.com/convox/rack

COPY . /go/src/github.com/convox/rack
RUN make build compress

# add crane (single static binary)
ENV VERSION=v0.20.3
ENV OS=Linux
ENV ARCH=x86_64

RUN curl -sL "https://github.com/google/go-containerregistry/releases/download/${VERSION}/go-containerregistry_${OS}_${ARCH}.tar.gz" > go-containerregistry.tar.gz && \
    mkdir -p /usr/local/bin/crane && \
    tar -zxvf go-containerregistry.tar.gz -C /usr/local/bin/ crane && \
    rm go-containerregistry.tar.gz

# Kaniko runtime
FROM gcr.io/kaniko-project/executor:debug

# Copy crane
COPY --from=package /usr/local/bin/crane /usr/local/bin/crane

# copy the Convox tools
COPY --from=package /go/bin/build /usr/local/bin/
COPY --from=package /go/bin/convox-env /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/build"]
CMD ["-h"]
