FROM heroku/cedar

RUN cd /tmp && git clone https://github.com/heroku/heroku-buildpack-{{ .kind }}
ENV HOME=/app
WORKDIR /app

{{ if .environment }}
ENV {{ range $key, $value := .environment }}{{ $key }}={{ $value }} {{ end }}
{{ end }}
{{ range $line := .precompile }}
{{ $line }}
{{ end }}

COPY . /app
RUN chmod +x /app/entrypoint.sh

RUN /tmp/heroku-buildpack-{{ .kind }}/bin/compile /app /tmp/cache

ENTRYPOINT ["/app/entrypoint.sh"]
