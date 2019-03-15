FROM convox/rails

ENV DIR /app
ENV FOO bar

# copy only the files needed for bundle install
# uncomment the vendor/cache line if you `bundle package` your gems
COPY Gemfile      $DIR/Gemfile
COPY Gemfile.lock $DIR/Gemfile.lock
# COPY vendor/cache $DIR/vendor/cache
RUN bundle install

# copy just the files needed for assets:precompile
COPY Rakefile   $DIR/Rakefile
COPY config     $DIR/config/$FOO
COPY public     $DIR/public/$FAKE
COPY app/assets /app$DIR/assets
RUN rake assets:precompile

ADD http://example.org/example /tmp/example
ADD https://example.org/example /tmp/example

# copy the rest of the app
COPY . $DIR
