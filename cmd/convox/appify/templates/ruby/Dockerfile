FROM convox/ruby

# copy only the files needed for bundle install
# uncomment the vendor/cache line if you `bundle package` your gems
COPY Gemfile      /app/Gemfile
COPY Gemfile.lock /app/Gemfile.lock
# COPY vendor/cache /app/vendor/cache
RUN bundle install

# copy the rest of the app
COPY . /app
