package controllers

import "net/http"

func Panic(rw http.ResponseWriter, r *http.Request) {
	panic("Controlled Panic")
}
