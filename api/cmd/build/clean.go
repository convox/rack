package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/convox/rack/manifest"
)

// MaxBuilds is the number of most recent builds to retain
const MaxBuilds = 3

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

// ImageBuilds builds is a map of image IDs to build IDs
type ImageBuilds map[string]string

func (is Images) Len() int           { return len(is) }
func (is Images) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }
func (is Images) Less(i, j int) bool { return is[i].CreatedAt.Before(is[j].CreatedAt) }

func clean() {
	// list image data in CSV format, e.g.:
	// 2017-06-01 17:43:39 -0700 PDT,0c64199163c6,web.BYDGVRTEDIW,782231114432.dkr.ecr.us-west-2.amazonaws.com/convox-myapp-qxawtfsdxt
	// 2017-06-01 17:36:04 -0700 PDT,7035bfa510e2,web.BBONMZTXNMA,782231114432.dkr.ecr.us-west-2.amazonaws.com/convox-myapp-qxawtfsdxt
	cmd := manifest.Docker("images", "--format", "{{.CreatedAt}},{{.ID}},{{.Tag}},{{.Repository}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	images := Images{}

	r := csv.NewReader(bytes.NewReader(out))
	r.FieldsPerRecord = 4

	records, err := r.ReadAll()
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
	imageBuilds := ImageBuilds{}

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

		// if never seen image, append to list of images
		if _, ok := imageBuilds[i.ID]; !ok {
			imageBuilds[i.ID] = i.BuildID
			builds[i.BuildID] = append(builds[i.BuildID], i)
		}
	}

	for repo, bids := range appBuilds {
		if len(bids) < MaxBuilds {
			fmt.Printf("Skipping %q with %d builds.\n", repo, len(bids))
			continue
		}

		for i := MaxBuilds; i < len(bids); i++ {
			bid := bids[i]

			for _, image := range builds[bid] {
				fmt.Printf("Deleting %q %q %s\n", repo, bid, image.ID)
				cmd := manifest.Docker("rmi", "-f", image.ID)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("ERROR: %s\n", err)
				}
			}
		}
	}
}
