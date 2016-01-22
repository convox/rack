# Rack Branch Release

- confirm you are on the right branch

  `git status`

- push your changes

  `git push`

- create the release in slack
  /release create <branch-name>

- login to demo

  `convox switch demo`

- upgrade demo to the branch

  ```
  convox rack update \
  $(convox rack releases --unpublished | \
    grep $(git branch --no-color 2> /dev/null | sed -e "/^[^*]/d" -e "s/* \(.*\)/\1/") | \
    head -n1)
  ```

- demo is on the right version

  `convox rack`
  `convox ps -a demo`

- demo is working
