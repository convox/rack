package workers

import "fmt"

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		// coerce r to error type
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		f(err)
	}
}
