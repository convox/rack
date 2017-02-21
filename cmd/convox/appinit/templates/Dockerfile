FROM heroku/cedar

RUN cd /tmp && git clone https://github.com/heroku/heroku-buildpack-{{ .kind }}
{{ if .environment }}
ENV {{ range $key, $value := .environment }}{{ $key }}={{ $value }} {{ end }}
{{ end }}
ENV HOME=/app
WORKDIR /app

COPY . /app
RUN chmod +x /app/entrypoint.sh

RUN /tmp/heroku-buildpack-{{ .kind }}/bin/compile /app /tmp/cache

ENTRYPOINT ["/app/entrypoint.sh"]
