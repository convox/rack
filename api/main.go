package main

import "fmt"

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		if err, ok := r.(error); ok {
			f(err)
		} else {
			f(fmt.Errorf("%v", r))
		}
	}
}

func main() {
	startWeb()
}
