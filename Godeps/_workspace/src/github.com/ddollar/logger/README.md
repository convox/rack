# logger

Easy logging in Go

## Examples

```go
var log = logger.New("ns=project")

// ns=project foo=bar num=5 pct=68.9 arg="test"
log.Log("foo=bar num=%d pct=%0.1f arg=%q", 5, 68.99, "test")

// ns=project sub=worker state=success foo=bar
log.Namespace("sub=worker").Success("foo=bar")

// ns=project at=buyProduct state=error error="invalid token"
log.At("buyProduct").Error(fmt.Errorf("invalid token"))

// ns=project foo=bar elapsed=2.398
l := log.Start()
l.Log("foo=bar")
```

## License

Apache 2.0 &copy; 2015 David Dollar
