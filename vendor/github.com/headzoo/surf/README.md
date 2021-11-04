Surf
====

[![Build Status](https://img.shields.io/travis/headzoo/surf/master.svg?style=flat-square)](https://travis-ci.org/headzoo/surf)
[![GoDoc](https://godoc.org/github.com/headzoo/surf?status.svg)](http://godoc.org/github.com/headzoo/surf)
[![Documentation](https://img.shields.io/badge/documentation-readthedocs-blue.svg?style=flat-square)](http://surf.readthedocs.io/)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://raw.githubusercontent.com/headzoo/surf/master/LICENSE.md)

Surf is a Go (golang) library that implements a virtual web browser that you control programmatically.
Surf isn't just another Go solution for downloading content from the web. Surf is designed to behave
like web browser, and includes: cookie management, history, bookmarking, user agent spoofing
(with a nifty user agent builder), submitting forms, DOM selection and traversal via jQuery style
CSS selectors, scraping assets like images, stylesheets, and other features.

**NOTE: DEVELOPMENT ON THIS PROJECT IS VERY SLOW!**  
I'm accepting pull requests and fixing bugs at a snail's pace right now. Partially because Surf has taken a backseat to projects which are more important to me, and the way Go handles vendoring makes me hesitant to make changes. Development will pick up again when I have more free time.

Work has started on v2.0 on the [v2 branch](https://github.com/headzoo/surf/tree/v2).

* [Installation](#installation)
* [General Usage](#quick-start)
* [Documentation](#documentation)
* [Credits](#credits)
* [Contributing](#contributing)
* [License](#license)


### Installation
Download the library using go.  
`go get gopkg.in/headzoo/surf.v1`

Import the library into your project.  
`import "gopkg.in/headzoo/surf.v1"`


### Quick Start
```go
package main

import (
	"gopkg.in/headzoo/surf.v1"
	"fmt"
)

func main() {
	bow := surf.NewBrowser()
	err := bow.Open("http://golang.org")
	if err != nil {
		panic(err)
	}
	
	// Outputs: "The Go Programming Language"
	fmt.Println(bow.Title())
}
```

### Documentation

Complete documentation is available on [Read the Docs](http://surf.readthedocs.io/).


### Credits
Surf was started by [Sean Hickey](https://github.com/headzoo) (headzoo) to learn more about the Go programming language.
The idea to create Surf was born in [this Reddit thread](http://www.reddit.com/r/golang/comments/2efw1q/mechanize_in_go/cjz4lze).

[![Twitter](https://img.shields.io/badge/follow-%40WebSeanHickey-orange.svg?style=flat-square)](https://twitter.com/WebSeanHickey)

Surf uses the awesome [goquery](https://github.com/PuerkitoBio/goquery) by Martin Angers, and
was written using [Intellij](http://www.jetbrains.com/idea/) and
the [golang plugin](http://plugins.jetbrains.com/plugin/5047).

Contributions have been made to Surf by the following awesome developers:

* [Haruyama Seigo](https://github.com/haruyama)
* [Tatsushi Demachi](https://github.com/tatsushid)
* [Charl Matthee](https://github.com/charl)
* [Matt Holt](https://github.com/mholt)
* [lalyos](https://github.com/lalyos)
* [lestrrat](https://github.com/lestrrat)
* [Carl Henrik Lunde](https://github.com/chlunde)
* [cornerot](https://github.com/cornerot)
* [lennyxc](https://github.com/lennyxc)
* [tlianza](https://github.com/tlianza)
* [joshuamorris3](https://github.com/joshuamorris3)
* [sqs](https://github.com/sqs)
* [nicot](https://github.com/nicot)


### Contributing
Issues and pull requests are _always_ welcome. Code changes are made to the
[`dev`](https://github.com/headzoo/surf/tree/dev) branch. Once a milestone has been reached the branch will
be merged in with `master`, and a new version tag created. **Do not** make your changes against the
`master` branch, or they will _may_ be ignored.

See [CONTRIBUTING.md](https://raw.githubusercontent.com/headzoo/surf/master/CONTRIBUTING.md) for more information.


### License
Surf is released open source software released under The MIT License (MIT).
See [LICENSE.md](https://raw.githubusercontent.com/headzoo/surf/master/LICENSE.md) for more information.
