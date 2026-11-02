FROM golang:1.26.10-bookworm AS package

ENV PATH=$PATH:/go/bin

# Add backports to get upx-ucl in Bookworm
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/backports.list && \
    apt-get update && apt-get install -y -t bookworm-backports upx-ucl

WORKDIR /go/src/github.com/convox/rack

COPY . /go/src/github.com/convox/rack
RUN make build compress

# Kaniko runtime
FROM ghcr.io/osscontainertools/kaniko:v1.29.0-debug
ENV FF_KANIKO_SHARED_BASE_CACHE=0

# Create /tmp so Kaniko can write
RUN mkdir -p /tmp && chmod 1777 /tmp

# Copy the build output
COPY --from=package /go/bin/build /busybox/
COPY --from=package /go/bin/convox-env /busybox/

ENTRYPOINT []
CMD ["/busybox/build"]
