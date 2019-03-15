package stdapi

func EnsureHTTPS(fn HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		if c.Request().Header.Get("X-Forwarded-Proto") == "http" {
			u := *(c.Request().URL)
			u.Host = c.Request().Host
			u.Scheme = "https"
			return c.Redirect(301, u.String())
		}
		return fn(c)
	}
}
