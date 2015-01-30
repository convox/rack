FROM convox/dind-go

EXPOSE 3000
RUN apt-get update
RUN apt-get -y install nginx
RUN mkdir -p /var/cache/nginx/body
RUN mkdir -p /var/cache/nginx/fastcgi
RUN mkdir -p /var/cache/nginx/proxy
RUN mkdir -p /var/cache/nginx/scgi
RUN mkdir -p /var/cache/nginx/uwsgi
RUN mkdir -p /var/run/nginx
COPY conf/nginx.conf /etc/nginx/nginx.conf

RUN go get github.com/ddollar/gin
RUN go get github.com/ddollar/init

ENV PORT 5000
WORKDIR /go/src/github.com/convox/kernel/builder
COPY . /go/src/github.com/convox/kernel/builder
RUN go get .

ENTRYPOINT ["init"]
CMD ["bin/builder"]
