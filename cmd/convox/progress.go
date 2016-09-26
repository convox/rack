package main

import (
	"io"
	"time"

	"github.com/cheggaaa/pb"
)

type ProgressMeter struct {
	after    string
	bar      *pb.ProgressBar
	finished bool
	prefix   string
	out      io.Writer
	total    int64
}

func (pm *ProgressMeter) Start(total int64) {
	pm.bar = pb.New64(total)
	pm.bar.Prefix(pm.prefix)
	pm.bar.SetMaxWidth(70)
	pm.bar.SetUnits(pb.U_BYTES)
	pm.bar.SetRefreshRate(200 * time.Millisecond)
	pm.bar.Output = pm.out
	pm.bar.Start()

	pm.total = total
}

func (pm *ProgressMeter) Progress(current int64) {
	pm.bar.Set64(current)

	if current >= pm.total {
		pm.Finish()
	}
}

func (pm *ProgressMeter) Finish() {
	if pm.finished {
		return
	}

	pm.bar.Finish()

	if pm.after != "" {
		pm.out.Write([]byte(pm.after))
	}

	pm.finished = true
}

func progress(prefix, after string, out io.Writer) *ProgressMeter {
	return &ProgressMeter{
		after:  after,
		out:    out,
		prefix: prefix,
	}
}
