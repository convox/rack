# Rack Branch Release

- confirm you are on the right branch

  `git status`

- push your changes

  `git push`

- create the release in slack
  /release create <branch-name>

- login to demo

  `convox switch demo`

- observe process formation and version

  `convox rack`
  `convox ps -a demo`

- upgrade demo to the branch

  ```
  convox rack update \
  $(convox rack releases --unpublished | \
    grep $(git branch --no-color 2> /dev/null | sed -e "/^[^*]/d" -e "s/* \(.*\)/\1/") | \
    head -n1)
  ```

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
