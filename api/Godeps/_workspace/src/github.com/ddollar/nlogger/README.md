# nlogger

Logging for Negroni

## Examples

```go
server := negroni.New(negroni.NewRecovery(), nlogger.New("ns=myapp", nil), negroni.NewStatic(http.Dir("public")))
```

## License

Apache 2.0 &copy; 2015 David Dollar
