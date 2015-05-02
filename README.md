# convox/kernel

##### Install docker-compose:

```
curl -L https://github.com/docker/compose/releases/download/1.1.0/docker-compose-`uname -s`-`uname -m` > /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

##### Create `.env`:

```
AWS_REGION=us-east-1
AWS_ACCESS=foo
AWS_SECRET=bar
```

##### Run the kernel for local development:

`make dev`


##### Release

Publish formation.json to public S3, and push the kernel image to Docker Hub. If necessary, `export AWS_DEFAULT_PROFILE=...` for proper credentials.

`make release`

## License

Apache 2.0 &copy; 2015 Convox, Inc.
