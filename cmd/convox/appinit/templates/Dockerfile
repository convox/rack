FROM heroku/cedar

RUN cd /tmp && git clone https://github.com/heroku/heroku-buildpack-{{ .kind }}
ENV HOME=/app
WORKDIR /app
{{ if .environment }}
ENV {{ range $key, $value := .environment }}{{ $key }}={{ $value }} {{ end }}
{{ end }}
{{ if .precompile }}{{ .precompile }}{{ end }}

COPY . /app

RUN output=$(/tmp/heroku-buildpack-{{ .kind }}/bin/compile /app /tmp/cache) || echo $output
