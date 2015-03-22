FROM convox/dind-go

RUN go get github.com/ddollar/init
RUN go get github.com/ddollar/rerun

ENV PORT 3000
WORKDIR /go/src/github.com/convox/kernel/builder
COPY . /go/src/github.com/convox/kernel/builder
RUN go get .

ENTRYPOINT ["init"]
CMD ["bin/builder"]
