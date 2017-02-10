package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitReadProcfile(t *testing.T) {

	procfileTests := []struct {
		data     []byte
		procfile Procfile
	}{
		{[]byte(rubyProcfile), Procfile{{"web", "bundle exec puma -C config/puma.rb"}}},
		{[]byte(pythonProcfile), Procfile{{"web", "gunicorn gettingstarted.wsgi --log-file -"}}},
		{[]byte(nodejsProcfile), Procfile{{"web", "node index.js"}}},
	}

	for _, pt := range procfileTests {
		pf := ReadProcfile(pt.data)
		assert.Equal(t, pt.procfile, pf)
	}

}

func TestInitReadAppManifest(t *testing.T) {

	appManifestTests := []struct {
		data []byte
		am   AppManifest
	}{
		{[]byte(appWithNoAddonsNoEnv), AppManifest{[]string{}, nil}},
		{[]byte(appWithPostgres), AppManifest{[]string{"heroku-postgresql"}, nil}},
		{[]byte(appWithEnvAndPostgres), AppManifest{[]string{"heroku-postgresql"}, map[string]EnvEntry{"DEBUG": {"1"}, "SECRET_TOKEN": {"secret"}}}},
	}

	for _, at := range appManifestTests {
		am, err := ReadAppManifest(at.data)
		assert.NoError(t, err)
		assert.Equal(t, at.am, am)
	}

}

func TestInitInvalidAppManifestJson(t *testing.T) {
	_, err := ReadAppManifest([]byte("foobar" + appWithNoAddonsNoEnv))
	assert.Error(t, err)
}

var (
	rubyProcfile   = `web: bundle exec puma -C config/puma.rb`
	pythonProcfile = `web: gunicorn gettingstarted.wsgi --log-file -`
	nodejsProcfile = `web: node index.js`

	appWithNoAddonsNoEnv = `{
  "name": "Start on Heroku: Node.js",
  "description": "A barebones Node.js app using Express 4",
  "repository": "https://github.com/heroku/node-js-getting-started",
  "logo": "http://node-js-sample.herokuapp.com/node.svg",
  "keywords": ["node", "express", "static"],
  "image": "heroku/nodejs"
}`

	appWithPostgres = `{
  "name": "Start on Heroku: Ruby",
  "description": "A barebones Rails app, which can easily be deployed to Heroku",
  "image": "heroku/ruby",
  "repository": "https://github.com/heroku/ruby-getting-started",
  "keywords": ["ruby", "rails" ],
  "addons": [ "heroku-postgresql" ]
}`

	appWithEnvAndPostgres = `{
  "name": "Start on Heroku: Ruby",
  "description": "A barebones Rails app, which can easily be deployed to Heroku",
  "image": "heroku/ruby",
  "repository": "https://github.com/heroku/ruby-getting-started",
  "keywords": ["ruby", "rails" ],
  "addons": [ "heroku-postgresql" ],
  "env": {
          "SECRET_TOKEN": {"value": "secret"},
          "DEBUG": {"value": "1"}
  }
}`
)
