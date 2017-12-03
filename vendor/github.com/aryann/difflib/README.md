[![GoDoc](https://godoc.org/github.com/aryann/difflib?status.svg)](http://godoc.org/github.com/aryann/difflib)

difflib
=======

difflib is a simple library written in [Go](http://golang.org/) for
diffing two sequences of text.


Installing
----------

To install, issue:

    go get github.com/aryann/difflib


Using
-----

To start using difflib, create a new file in your workspace and import
difflib:

    import (
            ...
            "fmt"
            "github.com/aryann/difflib"
            ...
    )

Then call either `difflib.Diff` or `difflib.HTMLDiff`:

    fmt.Println(difflib.HTMLDiff([]string{"one", "two", "three"}, []string{"two", "four", "three"}))

If you'd like more control over the output, see how the function
`HTMLDiff` relies on `Diff` in difflib.go.


Running the Demo
----------------

There is a demo application in the difflib_demo directory. To run it,
navigate to your `$GOPATH` and run:

    go run src/github.com/aryann/difflib/difflib_server/difflib_demo.go <file-1> <file-2>

Where `<file-1>` and `<file-2>` are two text files you'd like to
diff. The demo will launch a web server that will contain a table of
the diff results.
