package models

import (
	"fmt"
	"strings"

	"github.com/convox/rack/manifest"
)

type CronJob struct {
	Name     string `yaml:"name"`
	Schedule string `yaml:"schedule"`
	Command  string `yaml:"command"`
	Service  *manifest.Service
	App      *App
}

func NewCronJobFromLabel(key, value string) CronJob {
	keySlice := strings.Split(key, ".")
	name := keySlice[len(keySlice)-1]
	tokens := strings.Fields(value)
	cronjob := CronJob{
		Name:     name,
		Schedule: fmt.Sprintf("cron(%s *)", strings.Join(tokens[0:5], " ")),
		Command:  strings.Join(tokens[5:], " "),
	}
	return cronjob
}

func (cr *CronJob) AppName() string {
	return cr.App.Name
}

func (cr *CronJob) Process() string {
	return cr.Service.Name
}

func (cr *CronJob) ShortName() string {
	return fmt.Sprintf("%s%s", strings.Title(cr.Service.Name), strings.Title(cr.Name))
}

func (cr *CronJob) LongName() string {
	return fmt.Sprintf("%s-%s-%s", cr.App.StackName(), cr.Process(), cr.Name)
}
