// Reading and writing files are basic tasks needed for
// many Go programs. First we'll look at some examples of
// reading files.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	targetFile := os.Args[1]
	tags := os.Args[2:]

	// go run tries to compile the targetfile if it has the .go extension, so
	// we don't provide that on the command line and append it here.
	targetFile = fmt.Sprintf("%s.go", targetFile)

	absPath, err := filepath.Abs(targetFile)
	if nil != err {
		panic(err)
	}
	fileContents, err := ioutil.ReadFile(absPath)
	if nil != err {
		panic(err)
	}
	fmt.Printf("Opened file: %s\n", absPath)
	tagString := ""
	for _, eachTag := range tags {
		tagString = fmt.Sprintf("%s %s", tagString, eachTag)
	}
	fmt.Printf("Prepending tags: %s\n", tagString)

	updatedContents := fmt.Sprintf("// +build %s\n\n%s", tagString, fileContents)
	err = ioutil.WriteFile(absPath, []byte(updatedContents), 0644)
	if nil != err {
		panic(err)
	}
}
