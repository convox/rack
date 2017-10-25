package appinit

import (
	"testing"

	"github.com/convox/rack/manifest1"
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
		m  manifest1.Manifest
	}{
		{ //
			Procfile{},
			Appfile{Env: map[string]EnvEntry{"SECRET": {"top secret"}}},
			Release{Addons: []string{"heroku-postgres"}, ProcessTypes: map[string]string{"web": "gunicorn gettingstarted.wsgi --log-file -"}},
			manifest1.Manifest{
				Version: "2",
				Services: map[string]manifest1.Service{
					"web": {
						Build: manifest1.Build{
							Context: ".",
						},
						Command: manifest1.Command{
							String: "gunicorn gettingstarted.wsgi --log-file -",
						},
						Environment: manifest1.Environment{
							{
								Name:  "PORT",
								Value: "4001",
							},
							{
								Name:  "SECRET",
								Value: "top secret",
							},
						},
						Ports: manifest1.Ports{
							{
								Name:      "80",
								Balancer:  80,
								Container: 4001,
								Public:    true,
								Protocol:  manifest1.TCP,
							},
							{
								Name:      "443",
								Balancer:  443,
								Container: 4001,
								Public:    true,
								Protocol:  manifest1.TCP,
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
			manifest1.Manifest{
				Version: "2",
				Services: map[string]manifest1.Service{
					"web": {
						Name: "web",
						Build: manifest1.Build{
							Context: ".",
						},
						Command: manifest1.Command{
							String: "python server.py",
						},
						Environment: manifest1.Environment{
							{
								Name:  "PORT",
								Value: "4001",
							},
							{
								Name:  "SECRET",
								Value: "top secret",
							},
						},
						Labels: manifest1.Labels{
							"convox.port.443.protocol": "tls",
						},
						Ports: manifest1.Ports{
							{
								Name:      "80",
								Balancer:  80,
								Container: 4001,
								Public:    true,
								Protocol:  manifest1.TCP,
							},
							{
								Name:      "443",
								Balancer:  443,
								Container: 4001,
								Public:    true,
								Protocol:  manifest1.TCP,
							},
						},
					},
					"worker": {
						Build: manifest1.Build{
							Context: ".",
						},
						Command: manifest1.Command{
							String: "python worker.py",
						},
						Environment: manifest1.Environment{
							{
								Name:  "SECRET",
								Value: "top secret",
							},
						},
						Labels: manifest1.Labels{},
						Ports:  manifest1.Ports{},
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
