package workers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/rack/api/models"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

// Set up rack instance Docker environment for builds
// Log into all configured private registries
// Pull down latest images for all apps
func StartImages() {
	var log = logger.New("ns=app_images")

	if os.Getenv("DEVELOPMENT") == "true" {
		return
	}

	// doing this in development updates a ~/.docker file and causes a rerun loop
	models.LoginPrivateRegistries()

	maxRetries := 5
	var err error

	for i := 0; i < maxRetries; i++ {
		err = models.DockerLogin(docker.AuthConfiguration{
			Email:         "user@convox.com",
			Username:      "convox",
			Password:      os.Getenv("PASSWORD"),
			ServerAddress: os.Getenv("REGISTRY_HOST"),
		})

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
