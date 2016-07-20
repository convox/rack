FROM ruby:2.2.2

EXPOSE 3000
ENV PORT 3000

WORKDIR /app

COPY Gemfile /app/Gemfile
COPY Gemfile.lock /app/Gemfile.lock
RUN bundle install

COPY . /app
