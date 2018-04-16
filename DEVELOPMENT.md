# Rack Development

* Install a new rack to be used for development purposes
* Use `convox login` to log directly into that rack
* Set development mode with `convox rack params set Development=Yes`
* Switch to your local rack with `convox switch local`
* Create an app to host your development rack with `convox apps create rack`
* Set your development environment with `bin/export-env <dev-stack-name> | convox env set -a rack`
* Run the development rack with `convox start`
* Point your CLI at the development rack with `convox login web.rack.convox` and use the password from your env
