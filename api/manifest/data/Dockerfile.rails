FROM rails

WORKDIR /app

COPY Gemfile /app/Gemfile
COPY Gemfile.lock /app/Gemfile.lock
RUN bundle install

COPY . /app
RUN rake assets:precompile

CMD ["rails", "server", "-b", "0.0.0.0", "-P", "/tmp/server.pid"]
