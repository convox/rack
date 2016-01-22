# Rack Master Release

- confirm all changes are in master

  `git checkout master`
  `git show`

- login to demo

  `convox switch demo`

- upgrade demo to the latest head

  `convox rack update $(convox rack releases | head -n1)`

- demo is on the right version

  `convox rack`
  `convox ps -a demo`

- demo is working
