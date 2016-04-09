# composure

The composure package defines structure to describe everything about an app, and tooling to take an app seamlessly from development to production.

This package borrows a lot of ideas and functionality from `docker-compose`, but executes everything "the convox way" which:

* Has strong opinions about how apps will run in production, and works backwards to development
* Uses conventions (not configuration) to build and run apps
* Removes Docker options (like networks and linking) that do not work well in production
* Follows 12 factor, especially using environment variables to connect to backing services
* Adds simple things to a development environment that mimic production, like a TCP load balancer

This package facilitates a universal `convox build` that runs as part of `convox start` on a local machine, and `convox deploy` on a production Rack API. It also defines a canonical app configuration that can be turned into production resources on cloud provider.