package models

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"regexp"
	"strings"
	"html/template"

	"github.com/convox/rack/manifest"
)

// Agent represents a Service which runs exactly once on every ECS agent
type Agent struct {
	Service *manifest.Service
	App     *App
}

var shortNameRegex = regexp.MustCompile("[^A-Za-z0-9]+")

// ShortName returns the name of the Agent Service, sans any invalid characters
func (d *Agent) ShortName() string {
	shortName := strings.Title(d.Service.Name)
	return shortNameRegex.ReplaceAllString(shortName, "")
}

// LongName returns the name of the Agent Service in [stack name]-[service name]-[hash] format
func (d *Agent) LongName() string {
	prefix := fmt.Sprintf("%s-%s", d.App.StackName(), d.Service.Name)
	hash := sha256.Sum256([]byte(prefix))
	suffix := "-" + base32.StdEncoding.EncodeToString(hash[:])[:7]

	// $prefix-$suffix-change" needs to be <= 64 characters
	if len(prefix) > 57-len(suffix) {
		prefix = prefix[:57-len(suffix)]
	}
	return prefix + suffix
}

// Agents returns any Agent Services defined in the given Manifest
func (a App) Agents(m manifest.Manifest) []Agent {
	agents := []Agent{}

	for _, entry := range m.Services {
		if !entry.IsAgent() {
			continue
		}

		e := entry
		agent := Agent{
			Service: &e,
			App:     &a,
		}
		agents = append(agents, agent)
	}
	return agents
}

// AgentFunctionCode returns the Node.js code used by the AgentFunction lambda
func (a App) AgentFunctionCode() map[string]template.HTML {
	code := `
'use strict';

const aws = require('aws-sdk');
const ecs = new aws.ECS({ maxRetries: 10 });

// arn:aws:ecs:<region>:<aws_account_id>:task-definition/<task name>:<task def revision>
const taskDefinitions = [
    /* TASK DEFINITIONS */
];

// Task Definition ARN, minus revision
function tdName(td) {
    return td.split(':').slice(0, -1).join(':');
}

// Only the revision
function tdRev(td) {
    return parseInt(td.split(':').slice(-1)[0]);
}

function startTask(event, desiredTD) {
    let options = {
        containerInstances: [event.detail.containerInstanceArn],
        taskDefinition: desiredTD,
        cluster: event.detail.clusterArn,
        startedBy: 'convox agent'
    };

    return ecs.startTask(options).promise()
    .then(data => {
        if (data.tasks.length === 0) {
            throw new Error('Task not started');
        }

        console.log('startTask Data: ', data);
        return data.tasks[0].taskArn;
    });
}

function stopTask(event, runningTask) {
    if (runningTask.startedBy !== 'convox agent') {
        return;
    }

    let options = {
        task: runningTask.taskArn,
        cluster: event.detail.clusterArn,
        reason: 'convox agent convergence'
    };

    return ecs.stopTask(options).promise()
    .then(data => {
        console.log('stopTask Data: ', data);
        return data.tasks[0].taskArn;
    });
}

exports.handler = (event, context, callback) => {
    console.log('Event: ', event);

    let options = {
        cluster: event.detail.clusterArn,
        containerInstance: event.detail.containerInstanceArn
    };

    ecs.listTasks(options).promise()
    .then(data => {
        console.log('listTasks Data: ', data);

        // Can't call ecs.describeTasks if data.taskArns is empty
        if (!data.taskArns || !data.taskArns.length) {
            return {
                tasks: []
            };
        }

        let options = {
            cluster: event.detail.clusterArn,
            tasks: data.taskArns
        };

        return ecs.describeTasks(options).promise();
    })
    .then(data => {
        console.log('describeTasks Data: ', data);

        let promises = [];
        for (let desiredTD of taskDefinitions) {
            let alreadyRunning = false;
            for (let task of data.tasks) {
                if (desiredTD === task.taskDefinitionArn) {
                    alreadyRunning = true;
                    continue;
                }

                // Stop old tasks
                if (tdName(desiredTD) === tdName(task.taskDefinitionArn) &&
                    tdRev(desiredTD) !== tdRev(task.taskDefinitionArn))
                    promises.push(stopTask(event, task));
            }

            // Start new tasks
            if (!alreadyRunning) {
                promises.push(startTask(event, desiredTD));
            }
        }

        // Wait for tasks to start/stop
        return Promise.all(promises);
    })
    .then(() => {
        console.log('Success');
        return callback(null, 'Success');
    })
    .catch(err => {
        console.log('Error: ', err);
        return callback(err);
    });
};
`

	// Format JS code for embedding in app.tmpl
	halves := strings.Split(code, "/* TASK DEFINITIONS */")
	for i := range halves {
		oldLines := strings.Split(halves[i], "\n")
		newLines := []string{}

		for _, v := range oldLines {
			// Skip empty/comment lines (inline lambda code is limited to 4096 chars)
			t := strings.TrimSpace(v)
			if t == "" || strings.HasPrefix(t, "//") {
				continue
			}

			newLines = append(newLines, fmt.Sprintf(`"%s",`, v))
		}

		halves[i] = strings.Join(newLines, "\n")
	}

	// Remove trailing comma
	halves[1] = strings.TrimSuffix(halves[1], ",")

	return map[string]template.HTML{
		"head": template.HTML(halves[0]),
		"body": template.HTML(halves[1]),
	}
}
