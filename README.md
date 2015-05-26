# convox/app

Build Convox application stacks.

## Usage

    $ cat docker-compose.yml | docker run convox/app

## Parameters ([Production Mode](doc:deployment-modes)) 

| Name             | Default      | Description                                     |
|------------------|--------------|-------------------------------------------------|
| `Ami`            | **required** | AMI to use for this application                 |
| `Environment`    | *optional*   | Encrypted environment                           |
| `EnvironmentKey` | *optional*   | ARN of KMS key used to encrypt the environment  |
| `Repository`     | *optional*   | The canonical source repository for this app    |
| `SshKey`         | *optional*   | SSH key name to use to allow access to this app |

These parameters will appear once per process:

| Name         | Default      |                                          |
|--------------|--------------|------------------------------------------|
| `WebCommand` | *optional*   | Override the default command             |
| `WebImage`   | **required** | The docker image to use for this process |
| `WebScale`   | 1            | Number of instances to run               |
| `WebSize`    | t2.micro     | Instance size to use for this process    |

These parameters will appear if there are any port mappings:

| Name          | Default    | Description          |
|---------------|------------|----------------------|
| `HealthCheck` | *optional* | Healthcheck endpoint |

These parameters will appear once per port mapping:

| Name          | Default      | Description                                             |
|---------------|--------------|---------------------------------------------------------|
| `WebPort5000` | 5000         | Port to listen on the load balancer for a given mapping |

## Parameters ([Staging Mode](doc:deployment-modes)) 

| Name             | Default      | Description                                      |
|------------------|--------------|--------------------------------------------------|
| `Cluster`        | **required** | Cluster for this app (see convox/cluster)        |
| `Environment`    | *optional*   | Encrypted environment                            |
| `EnvironmentKey` | *optional*   | ARN of KMS key used to encrypt the environment   |
| `Kernel`         | **required** | Kernel notification endpoint (see convox/kernel) |
| `Repository`     | *optional*   | The canonical source repository for this app     |
| `Subnets`        | **required** | The VPC subnets for this app's cluster           |
| `Vpc`            | **required** | The VPC for this app's cluster                   |

These parameters will appear once per process:

| Name         | Default      |                                          |
|--------------|--------------|------------------------------------------|
| `WebCommand` | *optional*   | Override the default command             |
| `WebImage`   | **required** | The docker image to use for this process |
| `WebScale`   | 1            | Number of instances to run               |

These parameters will appear if there are any port mappings:

| Name          | Default    | Description          |
|---------------|------------|----------------------|
| `HealthCheck` | *optional* | Healthcheck endpoint |

These parameters will appear once per port mapping:

| Name          | Default      | Description                                             |
|---------------|--------------|---------------------------------------------------------|
| `WebPort5000` | 5000         | Port to listen on the load balancer for a given mapping |

## Help

    usage: convox/app [options]
      expects an optional docker-compose.yml on stdin

    options:
      -mode="production": deployment mode

    examples:
      $ cat docker-compose.yml | docker run -i convox/app -mode staging
