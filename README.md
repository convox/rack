# convox/kernel

<a name="installation">
## Installation

[![Install](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-east-1#cstack=sn%7Econvox%7Cturl%7Ehttp://convox.s3.amazonaws.com/kernel.json)

**Parameters**

| Parameter           | Value        | Description                                   |
|---------------------|--------------|-----------------------------------------------|
| `AvailabilityZones` | *optional*   | see [Availability Zones](#availability-zones) |
| `Key`               | *optional*   | name of ssh key in your aws account           |
| `Password`          | **required** | password used to authorize kernel access      |
| `RegistryImage`     | *[image]*    | docker image for kernel registry              |
| `WebImage`          | *[image]*    | docker image for kernel api                   |

<a name="availability-zones">
**Availability Zones**

If you have an older AWS account you may have some availability zones on which VPC does not function. If you see an error during installation referencing a list of valid availability zones then you can pick three of those and set the value of the `AvailabilityZone` parameter to `zone1,zone2,zone3`

## Development

**Prerequisites**

* working [docker](https://docs.docker.com/installation/) environment (`docker ps` should work)
* [docker-compose](https://docs.docker.com/compose/install/)

**Install kernel into an AWS account**

See [Installation](#installation)

**Create `.env`**

Look at the **Outputs** tab of the CloudFormation stack of the kernel you installed and build a `.env`  accordingly:

```
AWS_REGION=
AWS_ACCESS=
AWS_SECRET=
CUSTOM_TOPIC=
REGISTRY_HOST=
REGISTRY_PASSWORD=
```

**Install docker-compose**

```
curl -L https://github.com/docker/compose/releases/download/1.2.0/docker-compose-`uname -s`-`uname -m` > /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

**Configure an insecure registry with your docker daemon**

If you're using [boot2docker](http://boot2docker.io/) you can run:

```
boot2docker ssh "echo $'EXTRA_ARGS=\"--insecure-registry kernel-staging-1392086461.us-east-1.elb.amazonaws.com:5000\"' | sudo tee -a /var/lib/boot2docker/profile && sudo /etc/init.d/docker stop && sleep 2 && sudo /etc/init.d/docker start"
```

**Run the kernel in development mode**

Development mode uses `docker-compose`. Changes you make to the local project will be synced into the running containers and the project will be reloaded as needed.

`make dev`

**Open in a browser**

Go to `http://$DOCKER_HOST:5000` in your browser.

## License

Apache 2.0 &copy; 2015 Convox, Inc.
