package models

import (
	"archive/zip"
	"fmt"
	"os"
	"strings"
)

type CronJob struct {
	Name          string `yaml:"name"`
	Schedule      string `yaml:"schedule"`
	Command       string `yaml:"command"`
	ManifestEntry *ManifestEntry
}

func NewCronJobFromLabel(key, value string) CronJob {
	keySlice := strings.Split(key, ".")
	name := keySlice[len(keySlice)-1]
	tokens := strings.Fields(value)
	cronjob := CronJob{
		Name:     name,
		Schedule: fmt.Sprintf("cron(%s)", strings.Join(tokens[0:5], " ")),
		Command:  strings.Join(tokens[6:], " "),
	}
	return cronjob
}

func (cr *CronJob) ShortName() string {
	return fmt.Sprintf("%s%s", cr.ManifestEntry.Name, cr.Name)
}

func (cr *CronJob) LongName() string {
	app := cr.ManifestEntry.app
	return fmt.Sprintf("%s-%s%s", app.StackName, app.Name, cr.ShortName())
}

func (cr *CronJob) UploadLambdaFunction() error {
	data, err := buildTemplate("cronjob.js", "cronjob", cr)

	if err != nil {
		return err
	}

	// zip it
	zipfile, err := os.Create(fmt.Sprintf("/tmp/%s.zip", cr.ShortName))

	if err != nil {
		return err
	}

	w := zip.NewWriter(zipfile)

	f, err := w.Create(fmt.Sprintf("%s.js", cr.ShortName))

	if err != nil {
		return err
	}

	_, err = f.Write([]byte(data))

	if err != nil {
		return err
	}

	err = w.Close()

	if err != nil {
		return err
	}

	// upload it to S3
	err = S3PutFile(cr.ManifestEntry.app.Outputs["Settings"], fmt.Sprintf("cronjobs/%s.zip", cr.ShortName), zipfile, false)

	if err != nil {
		return err
	}

	return nil
}
