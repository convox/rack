# procmeminfo

[![GoDoc](http://godoc.org/github.com/guillermo/go.procmeminfo?status.png)](http://godoc.org/github.com/guillermo/go.procmeminfo)

Package procmeminfo provides an interface for /proc/meminfo

```golang
    import "github.com/guillermo/go.procmeminfo"
    meminfo := &procmeminfo.MemInfo{}
    meminfo.Update()

    (*meminfo)['Cached'] // Get cached memory
    (*meminfo)['Buffers'] // Get buffers size
    (*meminfo)['...'] // Any field in /proc/meminfo

    meminfo.Total() // Total memory size in bytes
    meminfo.Free() // Free Memory (Free + Cached + Buffers)
    meminfo.Used() // Total - Used
```


## Docs

Visit: http://godoc.org/github.com/guillermo/go.procmeminfo

## LICENSE

BSD
