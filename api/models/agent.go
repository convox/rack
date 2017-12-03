package models

// Agent represents a Service which runs exactly once on every ECS agent
// type Agent struct {
//   Service *manifest1.Service
//   App     *App
// }

// //Agents is a wrapper for sorting
// type Agents []Agent

// func (a Agents) Len() int           { return len(a) }
// func (a Agents) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
// func (a Agents) Less(i, j int) bool { return a[i].Service.Name < a[j].Service.Name }

// var shortNameRegex = regexp.MustCompile("[^A-Za-z0-9]+")

// // ShortName returns the name of the Agent Service, sans any invalid characters
// func (d *Agent) ShortName() string {
//   shortName := strings.Title(d.Service.Name)
//   return shortNameRegex.ReplaceAllString(shortName, "")
// }

// // LongName returns the name of the Agent Service in [stack name]-[service name]-[hash] format
// func (d *Agent) LongName() string {
//   prefix := fmt.Sprintf("%s-%s", d.App.StackName(), d.Service.Name)
//   hash := sha256.Sum256([]byte(prefix))
//   suffix := "-" + base32.StdEncoding.EncodeToString(hash[:])[:7]

//   // $prefix-$suffix-change" needs to be <= 64 characters
//   if len(prefix) > 57-len(suffix) {
//     prefix = prefix[:57-len(suffix)]
//   }
//   return prefix + suffix
// }

// // Agents returns any Agent Services defined in the given Manifest
// func (a App) Agents(m manifest1.Manifest) []Agent {
//   agents := Agents{}

//   for _, entry := range m.Services {
//     if !entry.IsAgent() {
//       continue
//     }

//     e := entry
//     agent := Agent{
//       Service: &e,
//       App:     &a,
//     }
//     agents = append(agents, agent)
//   }
//   sort.Sort(agents)
//   return agents
// }

// // AgentFunctionCode returns the Node.js code used by the AgentFunction lambda
// func (a App) AgentFunctionCode() map[string]template.HTML {
//   code := `
// 'use strict';

// const aws = require('aws-sdk');
// const ecs = new aws.ECS({ maxRetries: 10 });

// const STARTED_BY = 'convox agent';
// const STOPPED_REASON = 'convox agent convergence';

// // arn:aws:ecs:<region>:<aws_account_id>:task-definition/<task name>:<task def revision>
// const TASK_DEF_ARNS = [
//     /* TASK DEFINITION ARNs */
// ];

// // Task Definition ARN, minus revision
// function tdName(td) {
//     return td.split(':').slice(0, -1).join(':');
// }

// // Only the revision
// function tdRev(td) {
//     return parseInt(td.split(':').slice(-1)[0]);
// }

// function startTask(event, desiredTD) {
//     let options = {
//         containerInstances: [event.detail.containerInstanceArn],
//         taskDefinition: desiredTD,
//         cluster: event.detail.clusterArn,
//         startedBy: STARTED_BY
//     };

//     return ecs.startTask(options).promise()
//     .then(data => {
//         if (data.tasks.length === 0) {
//             throw new Error('Task not started');
//         }

//         console.log('startTask Data: ', data);
//         return data.tasks[0].taskArn;
//     });
// }

// function stopTask(event, runningTask) {
//     if (runningTask.startedBy !== STARTED_BY) {
//         console.log('Warning: Non-agent task running (scale count > 0?)');
//         return;
//     }

//     let options = {
//         task: runningTask.taskArn,
//         cluster: event.detail.clusterArn,
//         reason: STOPPED_REASON
//     };

//     return ecs.stopTask(options).promise()
//     .then(data => {
//         console.log('stopTask Data: ', data);
//         return data.task.taskArn;
//     });
// }

// exports.handler = (event, context, callback) => {
//     console.log('Event: ', event);

//     if (event.detail.stoppedReason === STOPPED_REASON) {
//         return callback(null, 'Ignored');
//     }

//     let options = {
//         cluster: event.detail.clusterArn,
//         containerInstance: event.detail.containerInstanceArn
//     };

//     ecs.listTasks(options).promise()
//     .then(data => {
//         console.log('listTasks Data: ', data);

//         // Can't call ecs.describeTasks if data.taskArns is empty
//         if (!data.taskArns || !data.taskArns.length) {
//             return {
//                 tasks: []
//             };
//         }

//         let options = {
//             cluster: event.detail.clusterArn,
//             tasks: data.taskArns
//         };

//         return ecs.describeTasks(options).promise();
//     })
//     .then(data => {
//         console.log('describeTasks Data: ', data);

//         let tasksToStop = [];
//         let tasksToStart = [];
//         for (let tdArn of TASK_DEF_ARNS) {
//             let alreadyRunning = false;

//             for (let task of data.tasks) {
//                 if (tdName(tdArn) !== tdName(task.taskDefinitionArn)) {
//                     continue;
//                 }

//                 if (tdRev(tdArn) === tdRev(task.taskDefinitionArn)) {
//                     alreadyRunning = true;
//                 } else {
//                     tasksToStop.push(task);
//                 }
//             }

//             if (!alreadyRunning) {
//                 tasksToStart.push(tdArn);
//             }
//         }

//         // Stop all tasks, then start new ones (to try and avoid port conflicts)
//         return Promise.all(tasksToStop.map(t => stopTask(event, t)))
//         .then( () => Promise.all(tasksToStart.map(t => startTask(event, t))) );
//     })
//     .then(() => {
//         console.log('Success');
//         return callback(null, 'Success');
//     })
//     .catch(err => {
//         console.log('Error: ', err);
//         return callback(err);
//     });
// };
// `

//   // Format JS code for embedding in app.tmpl
//   halves := strings.Split(code, "/* TASK DEFINITION ARNs */")
//   for i := range halves {
//     oldLines := strings.Split(halves[i], "\n")
//     newLines := []string{}

//     for _, v := range oldLines {
//       // Skip empty/comment lines (inline lambda code is limited to 4096 chars)
//       t := strings.TrimSpace(v)
//       if t == "" || strings.HasPrefix(t, "//") {
//         continue
//       }

//       newLines = append(newLines, fmt.Sprintf(`"%s",`, v))
//     }

//     halves[i] = strings.Join(newLines, "\n")
//   }

//   // Remove trailing comma
//   halves[1] = strings.TrimSuffix(halves[1], ",")

//   return map[string]template.HTML{
//     "head": template.HTML(halves[0]),
//     "body": template.HTML(halves[1]),
//   }
// }
