FROM heroku/cedar

RUN cd /tmp && git clone https://github.com/heroku/heroku-buildpack-{{ .framework }}

ENV HOME=/app
WORKDIR /app

COPY . /app
RUN chmod +x /app/entrypoint.sh

RUN /tmp/heroku-buildpack-{{ .framework }}/bin/compile /app /tmp/cache

ENTRYPOINT ["/app/entrypoint.sh"]
