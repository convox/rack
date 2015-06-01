# convox/kernel

##### Install docker-compose:

```
curl -L https://github.com/docker/compose/releases/download/1.1.0/docker-compose-`uname -s`-`uname -m` > /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

#### Bootstrap

Bootstrap a kernel on AWS. See dist/staging.json.

##### Create `.env`:

```
AWS_REGION=us-east-1
AWS_ACCESS=foo
AWS_SECRET=bar
CUSTOM_TOPIC=< Copy from staging stack CustomTopic Output >
REGISTRY=< Copy from staging stack RegistryHost Output >
```

If using Boot2Docker, configure for an your insecure Registry:

```
boot2docker ssh "echo $'EXTRA_ARGS=\"--insecure-registry kernel-staging-1392086461.us-east-1.elb.amazonaws.com:5000\"' | sudo tee -a /var/lib/boot2docker/profile && sudo /etc/init.d/docker stop && sleep 2 && sudo /etc/init.d/docker start"
```

##### Run the kernel for local development:

`make dev`


##### Release

Publish formation.json to public S3, and push the kernel image to Docker Hub. If necessary, `export AWS_DEFAULT_PROFILE=...` for proper credentials.

`make release`

## License

Apache 2.0 &copy; 2015 Convox, Inc.
