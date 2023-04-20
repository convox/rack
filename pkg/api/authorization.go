package api

import (
	"net/http"
	"strings"

	"github.com/convox/stdapi"
)

const ConvoxRoleParam = "CONVOX_ROLE"
const ConvoxRoleRead = "r"
const ConvoxRoleReadWrite = "rw"

func (s *Server) Authorize(next stdapi.HandlerFunc) stdapi.HandlerFunc {
	return func(c *stdapi.Context) error {
		switch c.Request().Method {
		case http.MethodGet:
			if !CanRead(c) {
				return stdapi.Errorf(401, "you are unauthorized to access this")
			}
		default:
			if !CanWrite(c) {
				return stdapi.Errorf(401, "you are unauthorized to access this")
			}
		}
		return next(c)
	}
}

func CanRead(c *stdapi.Context) bool {
	if d := c.Get(ConvoxRoleParam); d != nil {
		v, _ := d.(string)
		return strings.Contains(v, "r")
	}
	return false
}

func CanWrite(c *stdapi.Context) bool {
	if d := c.Get(ConvoxRoleParam); d != nil {
		v, _ := d.(string)
		return strings.Contains(v, "w")
	}
	return false
}

func SetReadRole(c *stdapi.Context) {
	c.Set(ConvoxRoleParam, ConvoxRoleRead)
}

func SetReadWriteRole(c *stdapi.Context) {
	c.Set(ConvoxRoleParam, ConvoxRoleReadWrite)
}
