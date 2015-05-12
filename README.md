# convox/service

Create a CloudFormation stack for a Convox service.

## Usage

    $ docker run convox/service <service>

## Available Services

### postgres

###### Parameters

| Name               | Description                           |
|--------------------|---------------------------------------|
| `AllocatedStorage` | Allocated storage size (GB)           |
| `InstanceClass`    | Instance class for database nodes     |
| `Database`         | Default database name (default 'app') |
| `Username`         | Server username (default 'postgres')  |
| `Password`         | Server password (required)            |

###### Outputs

| Name                  | Description           |
|-----------------------|-----------------------|
| `EnvPostgresDatabase` | Default database name |
| `EnvPostgresUsername` | Database username     |
| `EnvPostgresPassword` | Database password     |
| `Port5432TcpAddr`     | Database hostname     |
| `Port5432TcpPort`     | Database port         |

### redis

###### Parameters

| Name                | Description                          |
|---------------------|--------------------------------------|
| `AllowSSHFrom`      | Allow SSH from this CIDR block       |
| `SSHKey`            | Key name for SSH access              |
| `Database`          | Default database index (default '0') |
| `Password`          | Server password (required)           |

###### Outputs

| Name               | Description                  |
|--------------------|------------------------------|
| `EnvRedisDatabase` | Redis default database index |
| `EnvRedisPassword` | Redis password               |
| `Port6379TcpAddr`  | Redis hostname               |
| `Port6379TcpPort`  | Redis port                   |

## License

Apache 2.0 &copy; 2015 Convox, Inc.
