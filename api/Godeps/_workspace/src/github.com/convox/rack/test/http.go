package test

type Http struct {
	Method   string
	Path     string
	Code     int
	Body     string
	Response interface{}
}
