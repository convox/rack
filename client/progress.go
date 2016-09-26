package client

import "io"

type Progress interface {
	Start(total int64)
	Progress(current int64)
	Finish()
}

type ProgressReader struct {
	io.Reader
	progress int64
	tick     func(int64)
}

func NewProgressReader(r io.Reader, tick func(int64)) *ProgressReader {
	return &ProgressReader{
		Reader: r,
		tick:   tick,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)

	pr.progress += int64(n)
	pr.tick(pr.progress)

	return n, err
}
