package version

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	vs := Versions{
		Version{
			Version:   "1",
			Published: true,
			Required:  false,
		},
		Version{
			Version:   "2",
			Published: false,
			Required:  false,
		},
		Version{
			Version:   "3",
			Published: true,
			Required:  false,
		},
		Version{
			Version:   "4",
			Published: false,
			Required:  false,
		},
		Version{
			Version:   "5",
			Published: false,
			Required:  false,
		},
	}

	resolved, err := vs.Resolve("latest")
	if assert.Nil(t, err) {
		assert.Equal(t, "3", resolved.Version)
	}

	resolved, err = vs.Resolve("stable")
	if assert.Nil(t, err) {
		assert.Equal(t, "3", resolved.Version)
	}

	resolved, err = vs.Resolve("edge")
	if assert.Nil(t, err) {
		assert.Equal(t, "5", resolved.Version)
	}
}

func TestAppendVersion(t *testing.T) {
	t.Skip("Currently only runs with real credentials. Skipping until awsutil integration.")

	vs, err := AppendVersion(Version{
		Version:   "5678",
		Published: false,
		Required:  false,
	})

	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v\n", vs)
}

func TestNext(t *testing.T) {
	vs := Versions{
		Version{
			Version:   "1",
			Published: true,
		},
		Version{
			Version:   "2",
			Published: false,
		},
		Version{
			Version:   "3",
			Published: true,
		},
		Version{
			Version:   "4",
			Published: true,
			Required:  true,
		},
		Version{
			Version:   "5",
			Published: true,
			Required:  true,
		},
		Version{
			Version:   "6",
			Published: true,
		},
		Version{
			Version:   "7",
			Published: true,
		},
		Version{
			Version:   "8",
			Published: true,
		},
		Version{
			Version:   "9",
			Published: false,
		},
	}

	next, err := vs.Next("10")
	assert.Equal(t, "", next)
	assert.EqualError(t, err, `current version "10" not found`)

	next, err = vs.Next("1")
	assert.Equal(t, "4", next, "from version 1, next required version is 4")
	assert.Nil(t, err)

	next, err = vs.Next("4")
	assert.Equal(t, "5", next, "from version 4, next required version is 5")
	assert.Nil(t, err)

	next, err = vs.Next("5")
	assert.Equal(t, "8", next, "from version 5, latest published version is 8")
	assert.Nil(t, err)

	next, err = vs.Next("8")
	assert.Equal(t, "", next)
	assert.EqualError(t, err, `current version "8" is latest`)
}

func TestLatest(t *testing.T) {
	vs := Versions{
		Version{
			Version:   "1",
			Published: true,
		},
		Version{
			Version:   "2",
			Published: true,
		},
		Version{
			Version:   "3",
			Published: false,
		},
	}

	latest, err := vs.Latest()
	assert.Equal(t, "2", latest.Version)
	assert.Nil(t, err)
}

func TestNextBadVersionData(t *testing.T) {
	vs := Versions{
		Version{
			Version:   "1",
			Published: true,
		},
		Version{
			Version:   "2",
			Published: false,
			Required:  true, // Required but not Published makes no sense
		},
	}

	next, err := vs.Next("1")
	assert.Equal(t, "", next)
	assert.EqualError(t, err, `current version "1" is latest`)

	vs = Versions{
		Version{
			Version:   "1",
			Published: false,
		},
		Version{
			Version:   "2",
			Published: false, // nothing Published is not helpful
		},
	}

	latest, err := vs.Latest()
	assert.Equal(t, "", latest.Version)
	assert.EqualError(t, err, `no published versions`)
}

func TestSortVersions(t *testing.T) {
	vs := Versions{
		Version{
			Version:   "2",
			Published: true,
		},
		Version{
			Version:   "1",
			Published: false,
			Required:  true, // Required but not Published makes no sense
		},
	}

	sort.Sort(vs)

	assert.Equal(t, "1", vs[0].Version)
	assert.Equal(t, "2", vs[1].Version)
}
