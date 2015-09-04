package controllers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func init() {
}

func SystemShow(rw http.ResponseWriter, r *http.Request) {
	log := systemLogger("show").Start()

	rack := os.Getenv("RACK")

	a, err := models.GetApp(rack)

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such stack: %s", rack))
		return
	}

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	switch r.Header.Get("Content-Type") {
	case "application/json":
		RenderJson(rw, a)
	default:
		RenderTemplate(rw, "app", a)
	}
}

func SystemUpdate(rw http.ResponseWriter, r *http.Request) {
	log := systemLogger("update").Start()

	app, err := models.GetApp(os.Getenv("RACK"))

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	p := map[string]string{}

	if version := GetForm(r, "version"); version != "" {
		p["Version"] = version
	}

	if count := GetForm(r, "count"); count != "" {
		p["InstanceCount"] = count
	}

	if t := GetForm(r, "type"); t != "" {
		p["InstanceType"] = t
	}

	if len(p) > 0 {
		req := &cloudformation.UpdateStackInput{
			StackName:    aws.String(app.Name),
			Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		}

		if p["Version"] == "" {
			req.UsePreviousTemplate = aws.Boolean(true)
		} else {
			req.TemplateURL = aws.String(fmt.Sprintf("http://convox.s3.amazonaws.com/release/%s/formation.json", p["Version"]))
		}

		params := app.Parameters

		for key, val := range p {
			params[key] = val
		}

		for key, val := range params {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:   aws.String(key),
				ParameterValue: aws.String(val),
			})
		}

		_, err := models.CloudFormation().UpdateStack(req)

		if ae, ok := err.(awserr.Error); ok {
			if ae.Code() == "ValidationError" {
				switch {
				case strings.Index(ae.Error(), "No updates are to be performed") > -1:
					RenderNotFound(rw, fmt.Sprintf("no system updates are to be performed."))
					return
				case strings.Index(ae.Error(), "can not be updated") > -1:
					RenderNotFound(rw, fmt.Sprintf("system is already updating."))
					return
				}
			}
		}

		if err != nil {
			log.Error(err)
			RenderError(rw, err)
			return
		}
	}

	Redirect(rw, r, "/system")
}

func systemLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=system").At(at)
}
