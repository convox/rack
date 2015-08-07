package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/convox/kernel/models"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
)

func pullAppImages() {
	if os.Getenv("DEVELOPMENT") == "true" {
		return
	}

	var log = logger.New("ns=app_images")

	apps, err := models.ListApps()

	if err != nil {
		log.Error(err)
		return
	}

	log.Log("cmd=%q", fmt.Sprintf("docker login -e user@convox.com -u convox -p ***** %s", os.Getenv("REGISTRY_HOST")))
	data, err := exec.Command("docker", "login", "-e", "user@convox.io", "-u", "convox", "-p", os.Getenv("PASSWORD"), os.Getenv("REGISTRY_HOST")).CombinedOutput()

	if err != nil {
		fmt.Printf("%+v\n", string(data))
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
