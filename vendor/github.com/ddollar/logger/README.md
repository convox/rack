# logger

<a href="https://travis-ci.org/ddollar/logger">
  <img align="right" src="https://travis-ci.org/ddollar/logger.svg?branch=master">
</a>

Easy logging in Go

## Examples

```go
var log = logger.New("ns=project")

// ns=project foo=bar num=5 pct=68.9 arg="test"
log.Log("foo=bar num=%d pct=%0.1f arg=%q", 5, 68.99, "test")

// ns=project sub=worker state=success foo=bar
log.Namespace("sub=worker").Success("foo=bar")

// ns=project state=error id=1298498081 message="invalid token"
// ns=project state=error id=1298498081 line=1 trace="goroutine 7 [running:"
// ns=project state=error id=1298498081 line=2 trace="github.com/ddollar/logger.(*Logger).Error(0x208290500, 0x220826d780, 0x208268810)\"
log.Error(fmt.Errorf("invalid token"))

// ns=project foo=bar elapsed=2.398
l := log.Start()
l.Log("foo=bar")
```

## License

Apache 2.0 &copy; 2015 David Dollar
