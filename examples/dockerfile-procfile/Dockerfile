FROM convox/ruby

RUN gem install sinatra

ENV PORT 3000
EXPOSE 3000

WORKDIR /app
COPY . /app

CMD ["ruby", "web.rb"]
