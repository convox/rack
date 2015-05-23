# convox/app

Build Convox application stacks.

## Usage

    $ docker run convox/app -balancers front -processes web,worker -listeners front:web

## Parameters

The stacks created by this tool will have the following parameters:

### Production Mode

##### Global Parameters

| Name      | Default      | Description                                        |
|-----------|--------------|----------------------------------------------------|
| `Ami`     | **required** | AMI to use for this application                    |
| `EnvUrl`  | *optional*   | URL to an environment for this app (`.env` format) |
| `EnvKey`  | *optional*   | ARN of KMS key used to encrypt the environment     |
| `SshKey`  | *optional*   | SSH key name to use to allow access to this app    |

##### Balancer Parameters

| Name         | Default    | Description                                                          |
|--------------|------------|----------------------------------------------------------------------|
| `FrontCheck` | *optional* | If left blank will default to checking `GET /` on the first listener |

##### Process Parameters

| Name         | Default      |                                          |
|--------------|--------------|------------------------------------------|
| `WebCommand` | *optional*   | Override the default command             |
| `WebImage`   | **required** | The docker image to use for this process |
| `WebScale`   | 1            | Number of instances to run               |
| `WebSize`    | t2.micro     | Instance size to use for this process    |

##### Listener Parameters

| Name                   | Default      | Description                 |
|------------------------|--------------|-----------------------------|
| `FrontWebBalancerPort` | **required** | Listen port on the balancer |
| `FrontWebProcessPort`  | **required** | Listen port on the process  |

### Staging Mode

##### Global Parameters

| Name      | Default      | Description                                        |
|-----------|--------------|----------------------------------------------------|
| `EnvUrl`  | *optional*   | URL to an environment for this app (`.env` format) |
| `EnvKey`  | *optional*   | ARN of KMS key used to encrypt the environment     |
| `Subnets` | **required** | VPC subnets for this app                           |
| `Vpc`     | **required** | VPC for this app                                   |

##### Balancer Parameters

| Name         | Default    | Description                                                          |
|--------------|------------|----------------------------------------------------------------------|
| `FrontCheck` | *optional* | If left blank will default to checking `GET /` on the first listener |

##### Process Parameters

| Name         | Default      |                                          |
|--------------|--------------|------------------------------------------|
| `WebCommand` | *optional*   | Override the default command             |
| `WebImage`   | **required** | The docker image to use for this process |
| `WebScale`   | 1            | Number of instances to run               |

##### Listener Parameters

| Name                   | Default      | Description                 |
|------------------------|--------------|-----------------------------|
| `FrontWebBalancerPort` | **required** | Listen port on the balancer |
| `FrontWebProcessPort`  | **required** | Listen port on the process  |

## Help

    usage: app [-mode mode] [-processes p1[,p2]] [-balancers b1[,b2]] [-links b1:p1[,b1:p2]]]

    options:
      -mode="production": convox application mode, see:
      -balancers="": load balancer list
      -processes="": process list
      -listeners="": links between load balancers and processes

    examples:

      $ app -mode staging -balancers front -processes web,worker -listeners front:web

      $ app -processes worker


## License

Apache 2.0 &copy; 2015 Convox, Inc.
