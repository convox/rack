package appinit

import (
	"testing"

	"github.com/convox/rack/manifest"
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
		pf := ReadProcfileData(pt.data)
		assert.Equal(t, pt.procfile, pf)
	}

}

func TestInitReadAppfile(t *testing.T) {

	appfileTests := []struct {
		data []byte
		af   Appfile
	}{
		{[]byte(appWithNoAddonsNoEnv), Appfile{[]string{}, nil}},
		{[]byte(appWithPostgres), Appfile{[]string{"heroku-postgresql"}, nil}},
		{[]byte(appWithEnvAndPostgres), Appfile{[]string{"heroku-postgresql"}, map[string]EnvEntry{"DEBUG": {"1"}, "SECRET_TOKEN": {"secret"}}}},
	}

	for _, at := range appfileTests {
		af, err := ReadAppfileData(at.data)
		assert.NoError(t, err)
		assert.Equal(t, at.af, af)
	}

}

func TestGenerateManifest(t *testing.T) {
	manifestTests := []struct {
		pf Procfile
		af Appfile
		r  Release
		m  manifest.Manifest
	}{
		{ //
			Procfile{},
			Appfile{Env: map[string]EnvEntry{"SECRET": {"top secret"}}},
			Release{Addons: []string{"heroku-postgres"}, ProcessTypes: map[string]string{"web": "gunicorn gettingstarted.wsgi --log-file -"}},
			manifest.Manifest{
				Version: "2",
				Services: map[string]manifest.Service{
					"web": {
						Build: manifest.Build{
							Context: ".",
						},
						Command: manifest.Command{
							String: "gunicorn gettingstarted.wsgi --log-file -",
						},
						Environment: manifest.Environment{
							{
								Name:  "PORT",
								Value: "4001",
							},
							{
								Name:  "SECRET",
								Value: "top secret",
							},
						},
						Ports: manifest.Ports{
							{
								Name:      "80",
								Balancer:  80,
								Container: 4001,
								Public:    true,
								Protocol:  manifest.TCP,
							},
							{
								Name:      "443",
								Balancer:  443,
								Container: 4001,
								Public:    true,
								Protocol:  manifest.TCP,
							},
						},
					},
				},
			},
		}, /////////

		{ //
			Procfile{{Name: "web", Command: "python server.py"}, {Name: "worker", Command: "python worker.py"}},
			Appfile{Env: map[string]EnvEntry{"SECRET": {"top secret"}}},
			Release{Addons: []string{"heroku-postgres"}, ProcessTypes: map[string]string{"web": "gunicorn gettingstarted.wsgi --log-file -"}},
			manifest.Manifest{
				Version: "2",
				Services: map[string]manifest.Service{
					"web": {
						Name: "web",
						Build: manifest.Build{
							Context: ".",
						},
						Command: manifest.Command{
							String: "python server.py",
						},
						Environment: manifest.Environment{
							{
								Name:  "PORT",
								Value: "4001",
							},
							{
								Name:  "SECRET",
								Value: "top secret",
							},
						},
						Labels: manifest.Labels{
							"convox.port.443.protocol": "tls",
						},
						Ports: manifest.Ports{
							{
								Name:      "80",
								Balancer:  80,
								Container: 4001,
								Public:    true,
								Protocol:  manifest.TCP,
							},
							{
								Name:      "443",
								Balancer:  443,
								Container: 4001,
								Public:    true,
								Protocol:  manifest.TCP,
							},
						},
					},
					"worker": {
						Build: manifest.Build{
							Context: ".",
						},
						Command: manifest.Command{
							String: "python worker.py",
						},
						Environment: manifest.Environment{
							{
								Name:  "SECRET",
								Value: "top secret",
							},
						},
						Labels: manifest.Labels{},
						Ports:  manifest.Ports{},
					},
				},
			},
		}, /////////
	}

	for _, mt := range manifestTests {
		m := GenerateManifest(mt.pf, mt.af, mt.r)
		assert.Equal(t, mt.m, m)
	}
}

func TestInitInvalidAppfileJson(t *testing.T) {
	_, err := ReadAppfileData([]byte("foobar" + appWithNoAddonsNoEnv))
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
