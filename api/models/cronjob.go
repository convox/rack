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
		Schedule: fmt.Sprintf("cron(%s)", strings.Join(tokens[0:6], " ")),
		Command:  strings.Join(tokens[6:], " "),
	}
	return cronjob
}

func (cr *CronJob) AppName() string {
	return cr.ManifestEntry.app.Name
}

func (cr *CronJob) Process() string {
	return cr.ManifestEntry.Name
}

func (cr *CronJob) ShortName() string {
	return fmt.Sprintf("%s%s", strings.Title(cr.ManifestEntry.Name), strings.Title(cr.Name))
}

func (cr *CronJob) LongName() string {
	app := cr.ManifestEntry.app
	return fmt.Sprintf("%s-%s-%s", app.StackName(), cr.Process(), cr.Name)
}

func (cr *CronJob) UploadLambdaFunction() error {
	bucket := cr.ManifestEntry.app.Outputs["Settings"]
	path := "functions/cron.zip"

	exists, err := s3Exists(bucket, path)

	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	input := map[string]interface{}{
		"CronJob": cr,
		"Rack":    os.Getenv("RACK"),
	}

	// build cron lambda JS
	data, err := buildTemplate("cronjob.js", "cronjob", input)

	if err != nil {
		return err
	}

	// zip it
	zipfile, err := os.Create("/tmp/cron.zip")

	if err != nil {
		return err
	}

	w := zip.NewWriter(zipfile)

	f, err := w.Create("index.js")

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
	err = S3PutFile(bucket, path, zipfile, false)

	if err != nil {
		return err
	}

	return nil
}
