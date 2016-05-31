FROM gliderlabs/alpine:3.1

RUN apk-install ca-certificates ruby ruby-dev

RUN gem install sinatra

ENV PORT 3000
EXPOSE 3000

WORKDIR /app
COPY . /app

CMD ["ruby", "web.rb"]
