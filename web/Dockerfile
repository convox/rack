FROM golang:1.4

RUN go get github.com/ddollar/gin

EXPOSE 3000
RUN apt-get update
RUN apt-get -y install nginx
RUN mkdir -p /nginx/client_body_temp
RUN mkdir -p /nginx/fastcgi_temp
RUN mkdir -p /nginx/proxy_temp
RUN mkdir -p /nginx/scgi_temp
RUN mkdir -p /nginx/uwsgi_temp
COPY conf/nginx.conf /etc/nginx/nginx.conf

ENV PORT 5000
WORKDIR /go/src/github.com/convox/kernel/web
COPY . /go/src/github.com/convox/kernel/web
RUN go get .

CMD ["bin/web"]
