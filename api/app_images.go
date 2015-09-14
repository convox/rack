package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/rack/api/models"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/ddollar/logger"
)

func pullAppImages() {
	var log = logger.New("ns=app_images")

	if os.Getenv("DEVELOPMENT") == "true" {
		return
	}

	maxRetries := 5
	var err error

	for i := 0; i < maxRetries; i++ {
		err := dockerLogin()

		if err == nil {
			break
		}

		time.Sleep(30 * time.Second)
	}

	if err != nil {
		return
	}

	apps, err := models.ListApps()

	if err != nil {
		log.Error(err)
		return
	}

	for _, app := range apps {
		a, err := models.GetApp(app.Name)

		if err != nil {
			log.Error(err)
			continue
		}

		for key, value := range a.Parameters {
			if strings.HasSuffix(key, "Image") {

				log.Log("cmd=%q", fmt.Sprintf("docker pull %s", value))
				data, err := exec.Command("docker", "pull", value).CombinedOutput()

				if err != nil {
					fmt.Printf("%+v\n", string(data))
					log.Error(err)
					continue
				}
			}
		}
	}
}

func dockerLogin() error {
	var log = logger.New("ns=app_images")

	log.Log("cmd=%q", fmt.Sprintf("docker login -e user@convox.com -u convox -p ***** %s", os.Getenv("REGISTRY_HOST")))
	data, err := exec.Command("docker", "login", "-e", "user@convox.io", "-u", "convox", "-p", os.Getenv("PASSWORD"), os.Getenv("REGISTRY_HOST")).CombinedOutput()

	if err != nil {
		fmt.Printf("%+v\n", string(data))
		log.Error(err)
	}

	return err
}
