package build

import (
	"testing"
)

func TestManifestFromInspect(t *testing.T) {
	q1 := []byte(`{"3000/tcp":{}}`)
	d1, _ := ManifestFromInspect("", q1)

	q2 := []byte(`{"3000/tcp":{},"3001/tcp":{},"3002/tcp":{},"3003/tcp":{},"3004/tcp":{},"3005/tcp":{}}`)
	d2, _ := ManifestFromInspect("", q2)

	cases := Cases{
		{d1, []byte(`web:
  build: .
  environment: []
  ports:
  - 5000:3000
`)},
		{d2, []byte(`web:
  build: .
  environment: []
  ports:
  - 5000:3000
  - 5100:3001
  - 5200:3002
  - 5300:3003
  - 5400:3004
  - 5500:3005
`)},
	}

	_assert(t, cases)
}
