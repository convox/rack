package main

import (
	"fmt"
	"runtime/debug"
)

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		// coerce r to error type
		err, ok := r.(error)

		if !ok {
			err = fmt.Errorf("%v", r)
		}

		stack := debug.Stack()
		fmt.Printf("stack %+v\n", string(stack))

		f(err)
	}
}

func main() {
	startWeb()
}
