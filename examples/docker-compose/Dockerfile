FROM ruby:2.2.2

WORKDIR /app

COPY Gemfile /app/Gemfile
COPY Gemfile.lock /app/Gemfile.lock
RUN bundle install

COPY . /app

CMD ["ruby", "web.rb"]
