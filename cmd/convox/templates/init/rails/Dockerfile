FROM convox/rails

# copy only the files needed for bundle install
# uncomment the vendor/cache line if you `bundle package` your gems
COPY Gemfile      /app/Gemfile
COPY Gemfile.lock /app/Gemfile.lock
# COPY vendor/cache /app/vendor/cache
RUN bundle install

# copy just the files needed for assets:precompile
COPY Rakefile   /app/Rakefile
COPY config     /app/config
COPY public     /app/public
COPY app/assets /app/app/assets
RUN rake assets:precompile

# copy the rest of the app
COPY . /app
