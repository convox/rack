package test

import "testing"

type Run interface {
	Test(*testing.T)
}

func Runs(t *testing.T, runs ...Run) {
	for _, run := range runs {
		run.Test(t)
	}
}
