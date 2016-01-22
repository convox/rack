# Rack Master Release

- create the release in slack

  /release create

- confirm all changes are in master

  `git checkout master`
  `git show`

- login to demo

  `convox switch demo`

- observe process formation and version

  `convox rack`
  `convox ps -a demo`

- upgrade demo to the latest head

  `convox rack update $(convox rack releases | head -n1)`

- login to console and watch cloudformation

Upgrading takes some time. Short upgrades include
just starting new Rack processes and can take
around 5 minutes. Rolling instances can take a lot longer.

  `open https://console.aws.amazon.com/`

  wait for "UPDATE_COMPLETE"

- confirm upgrade

  `convox rack`

- confirm processes back

  `convox ps -a demo`

- test release

- publish the release in slack

  /release publish <version>
