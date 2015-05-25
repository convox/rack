# convox/app

Build Convox application stacks.

## Usage

    $ docker run convox/app -balancers front -processes web,worker -listeners front:web

## Parameters ([Production Mode](doc:deployment-modes)) 

| Name         | Default      | Description                                        |
|--------------|--------------|----------------------------------------------------|
| `Ami`        | **required** | AMI to use for this application                    |
| `EnvUrl`     | *optional*   | URL to an environment for this app (`.env` format) |
| `EnvKey`     | *optional*   | ARN of KMS key used to encrypt the environment     |
| `Repository` | *optional*   | The canonical source repository for this app       |
| `SshKey`     | *optional*   | SSH key name to use to allow access to this app    |

These parameters will appear once per balancer:

| Name         | Default    | Description                                                          |
|--------------|------------|----------------------------------------------------------------------|
| `FrontCheck` | *optional* | If left blank will default to checking `GET /` on the first listener |

These parameters will appear once per process:

| Name         | Default      |                                          |
|--------------|--------------|------------------------------------------|
| `WebCommand` | *optional*   | Override the default command             |
| `WebImage`   | **required** | The docker image to use for this process |
| `WebScale`   | 1            | Number of instances to run               |
| `WebSize`    | t2.micro     | Instance size to use for this process    |

These parameters will appear once per listener:

| Name                   | Default      | Description                 |
|------------------------|--------------|-----------------------------|
| `FrontWebBalancerPort` | **required** | Listen port on the balancer |
| `FrontWebProcessPort`  | **required** | Listen port on the process  |

## Parameters ([Staging Mode](doc:deployment-modes)) 

| Name      | Default      | Description                                        |
|-----------|--------------|----------------------------------------------------|
| `Cluster`  | **required**   | Cluster for this app (see convox/cluster)       |
| `EnvUrl`  | *optional*   | URL to an environment for this app (`.env` format) |
| `EnvKey`  | *optional*   | ARN of KMS key used to encrypt the environment     |
| `Kernel` | **required** | Kernel notification endpoint (see convox/kernel)    |
| `Repository` | *optional* | The canonical source repository for this app      |

These parameters will appear once per balancer:

| Name         | Default    | Description                                                          |
|--------------|------------|----------------------------------------------------------------------|
| `FrontCheck` | *optional* | If left blank will default to checking `GET /` on the first listener |

These parameters will appear once per process:

| Name         | Default      |                                          |
|--------------|--------------|------------------------------------------|
| `WebCommand` | *optional*   | Override the default command             |
| `WebImage`   | **required** | The docker image to use for this process |
| `WebScale`   | 1            | Number of instances to run               |

These parameters will appear once per listener:

| Name                   | Default      | Description                           |
|------------------------|--------------|---------------------------------------|
| `FrontWebBalancerPort` | **required** | Port the load balancer will listed on |
| `FrontWebHostPort`  | **required** | Host port (must be unique per cluster)   |
| `FrontWebContainerPort`  | **required** | Port that container exposes         |

## Help

    usage: convox/app [options]

    options:
      -mode="production": deployment mode
      -balancers="": load balancer list
      -processes="": process list
      -listeners="": links between load balancers and processes

    examples:
      $ docker run convox/app -balancers front -processes web,worker -listeners front:web
      $ docker run convox/app -mode staging -processes worker
