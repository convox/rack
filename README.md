# convox/service

Create a CloudFormation stack for a Convox service.

## Usage

    $ docker run convox/service <service>

## Available Services

### postgres

###### Parameters

| Name                | Description                                                     |
|---------------------|-----------------------------------------------------------------|
| `AllocatedStorage`  | Allocated storage size (GB)                                     |
| `AvailabilityZones` | A comma-delimited list of availability zones to use (specify 3) |
| `InstanceClass`     | Instance class for database nodes                               |
| `Name`              | Default database name                                           |
| `Password`          | Server password                                                 |

###### Outputs

| Name       | Description       |
|------------|-------------------|
| `Addr`     | Database hostname |
| `Port`     | Database port     |
| `Name`     | Database name     |
| `Password` | Database password |

### redis

###### Parameters

| Name                | Description                                                     |
|---------------------|-----------------------------------------------------------------|
| `AllowSSHFrom`      | Allow SSH from this CIDR block                                  |
| `AvailabilityZones` | A comma-delimited list of availability zones to use (specify 3) |
| `Password`          | Server password                                                 |
| `SSHKey`            | Key name for SSH access                                         |

###### Outputs

| Name       | Description    |
|------------|----------------|
| `Addr`     | Redis hostname |
| `Port`     | Redis port     |
| `Password` | Redis password |

## License

Apache 2.0 &copy; 2015 Convox, Inc.
