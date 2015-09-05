package controllers

import "net/http"

func Boom(rw http.ResponseWriter, r *http.Request) {
	panic("Controlled Panic")
}
