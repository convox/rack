package controllers

import "net/http"

func UtilityBoom(rw http.ResponseWriter, r *http.Request) {
	panic("Controlled Panic")
}

func UtilityCheck(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("ok"))
}
