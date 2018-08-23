package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/convox/rack/pkg/manifest1"
)

// MaxBuilds is the number of most recent builds to retain
const MaxBuilds = 5

// BuildRe is a regex that matches a build tag like web.BYDGVRTEDIW
var BuildRe = regexp.MustCompile(`.*\.(B[A-Z]{10})`)

// Image is a representation of a Docker Image
type Image struct {
	CreatedAt  time.Time
	ID         string
	BuildID    string
	Repository string
	Tag        string
}

// Images is a list Docker Images
type Images []Image

// AppBuilds is a map of app image repository to time ordered list of build IDs
type AppBuilds map[string][]string

// Builds is a map of build IDs to a list of images
type Builds map[string]Images

func (is Images) Len() int           { return len(is) }
func (is Images) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }
func (is Images) Less(i, j int) bool { return is[i].CreatedAt.Before(is[j].CreatedAt) }

func clean() {
	// list image data in CSV format, e.g.:
	// 2017-06-01 17:43:39 -0700 PDT,0c64199163c6,web.BYDGVRTEDIW,782231114432.dkr.ecr.us-west-2.amazonaws.com/convox-myapp-qxawtfsdxt
	// 2017-06-01 17:36:04 -0700 PDT,7035bfa510e2,web.BBONMZTXNMA,782231114432.dkr.ecr.us-west-2.amazonaws.com/convox-myapp-qxawtfsdxt
	cmd := manifest1.Docker("images", "--format", "{{.CreatedAt}},{{.ID}},{{.Tag}},{{.Repository}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("WARNING: %s\n", err)
		return
	}

	images := Images{}

	r := csv.NewReader(bytes.NewReader(out))
	r.FieldsPerRecord = 4

	records, err := r.ReadAll()
	if err != nil {
		fmt.Printf("WARNING: %s\n", err)
		return
	}
	for _, r := range records {
		// Filter out non-build images
		matches := BuildRe.FindStringSubmatch(r[2])
		if len(matches) != 2 {
			continue
		}

		t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", r[0])
		if err != nil {
			continue
		}

		images = append(images, Image{
			CreatedAt:  t,
			ID:         r[1],
			BuildID:    matches[1],
			Tag:        r[2],
			Repository: r[3],
		})
	}

	sort.Sort(images)

	appBuilds := AppBuilds{}
	builds := Builds{}

	for _, i := range images {
		// if never seen app repo, initialize list of builds
		if _, ok := appBuilds[i.Repository]; !ok {
			appBuilds[i.Repository] = []string{}
		}

		// if never seen build, initialize list of images, append to list of builds
		if _, ok := builds[i.BuildID]; !ok {
			builds[i.BuildID] = Images{}
			appBuilds[i.Repository] = append(appBuilds[i.Repository], i.BuildID)
		}

		// append to list of images
		builds[i.BuildID] = append(builds[i.BuildID], i)
	}

	// collect old builds and related repo:tags to remove
	tags := map[string]bool{}
	for _, bids := range appBuilds {
		if len(bids) <= MaxBuilds {
			continue
		}

		for i := MaxBuilds; i < len(bids); i++ {
			bid := bids[i]

			for _, image := range builds[bid] {
				t := fmt.Sprintf("%s:%s", image.Repository, image.Tag)
				tags[t] = true
			}
		}
	}

	if len(tags) == 0 {
		return
	}

	// remove images
	for tag := range tags {
		args := []string{"rmi", tag}
		cmd = manifest1.Docker(args...)
		_, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("WARNING: %s\n", err)
		}
	}
}
