# Convox FluentD Driver

This service gets installed as an AWS lambda and triggers off cloudwatch log group stream creation. This is a lambda function that will parse the container output of an application running on a convox cluster and forward them to a given FluentD endpoint.

## Modifying

Get yer golang set up locally then modify `main.go`.

## Viewing errors/logs

Go to AWS > CloudWatch > Logs > Streams for /aws/lambda/${rack-name}-fluentd.

Here you will be able to see the stderr and stdout of the function as it runs.

## License

See LICENSE.

Created by Adam Enger @ reverb.com
