# convox/service

Create a CloudFormation stack for a Convox service.

## postgres

#### Usage

    $ docker run convox/service postgres

#### Parameters

| Name                | Description                                                     |
|---------------------|-----------------------------------------------------------------|
| `AllocatedStorage`  | Allocated storage size (GB)                                     |
| `AvailabilityZones` | A comma-delimited list of availability zones to use (specify 3) |
| `Database`          | Default database name                                           |
| `InstanceClass`     | Instance class for database nodes                               |
| `Password`          | Server password                                                 |

#### Outputs

| Name       | Description       |
|------------|-------------------|
| `Addr`     | Database hostname |
| `Port`     | Database port     |
| `Database` | Database name     |
| `Password` | Database password |

## redis

#### Usage

    $ docker run convox/service redis

#### Parameters

| Name                | Description                                                     |
|---------------------|-----------------------------------------------------------------|
| `AllowSSHFrom`      | Allow SSH from this CIDR block                                  |
| `AvailabilityZones` | A comma-delimited list of availability zones to use (specify 3) |
| `Password`          | Server password                                                 |
| `SSHKey`            | Key name for SSH access                                         |

#### Outputs

| Name       | Description    |
|------------|----------------|
| `Addr`     | Redis hostname |
| `Port`     | Redis port     |
| `Password` | Redis password |

## License

Apache 2.0 &copy; 2015 Convox, Inc.
