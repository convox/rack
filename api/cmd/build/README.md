# convox/build

Build and push Docker images for Convox apps

## Development

    # rebuild the API image and build cmd

    $ cd convox/rack
    $ docker build -t rack/api .
    ...
    RUN go install ./...

    # build a tarball on stdin

    $ cd httpd
    $ tar cz . | docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build httpd -

    manifest|web:
    manifest|  image: httpd
    manifest|  ports:
    manifest|  - 80:80
    manifest|  - 443:80
    manifest|web2:
    manifest|  image: httpd
    manifest|  ports:
    manifest|  - 8000:80
    build|RUNNING: docker tag -f httpd httpd/web

    # build a directory as long as its mounted into the container

    $ cd httpd
    $ docker run -i -v /var/run/docker.sock:/var/run/docker.sock -v $(pwd):/tmp/app rack/api \
      build httpd /tmp/app

    manifest|web:
    manifest|  image: httpd
    manifest|  ports:
    manifest|  - 80:80
    manifest|  - 443:80
    manifest|web2:
    manifest|  image: httpd
    manifest|  ports:
    manifest|  - 8000:80
    build|RUNNING: docker tag -f httpd httpd/web

    # build an http git repo

    $ docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build httpd https://github.com/nzoschke/httpd.git

    git|Cloning into '/tmp/repo655149862/clone'...
    git|POST git-upload-pack (190 bytes)
    git|remote: Counting objects: 20, done.
    remote: Compressing objects: 100% (16/16), done.
    git|remote: Total 20 (delta 11), reused 11 (delta 2), pack-reused 0
    git|Checking connectivity... done.
    manifest|web:
    manifest|  image: httpd
    manifest|  ports:
    manifest|  - 80:80
    build|RUNNING: docker tag -f httpd httpd/web

    # build a git repo with a commit-ish

    $ docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build httpd https://github.com/nzoschke/httpd.git#thunk

    git|Cloning into '/tmp/repo824977679/clone'...
    git|POST git-upload-pack (190 bytes)
    git|remote: Counting objects: 20, done.
    remote: Compressing objects: 100% (16/16), done.
    git|remote: Total 20 (delta 11), reused 11 (delta 2), pack-reused 0
    git|Checking connectivity... done.
    git|error: pathspec 'thunk' did not match any file(s) known to git.
    ERROR: exit status 1

    # try an SSH git repo

    $ docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build httpd ssh://git@gitlab.com:nzoschke/httpd.git

    git|Cloning into '/tmp/repo295143718/clone'...
    git|Warning: Permanently added 'gitlab.com,104.210.2.228' (ECDSA) to the list of known hosts.
    git|Permission denied (publickey).
    git|fatal: Could not read from remote repository.
    git|
    git|Please make sure you have the correct access rights
    git|and the repository exists.
    ERROR: exit status 128
    2016/03/09 22:47:54 exit status 1

    # build an SSH git repo by passing in a public key that's configured as an SSH deploy key

    $ KEY=$(base64 /tmp/id_rsa)
    $ docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build httpd ssh://git:$KEY@gitlab.com:nzoschke/httpd.git

    git|Cloning into '/tmp/repo839273553/clone'...
    git|Warning: Permanently added 'gitlab.com,104.210.2.228' (ECDSA) to the list of known hosts.
    git|remote: Counting objects: 10, done.
    remote: Compressing objects: 100% (8/8), done.
    git|remote: Total 10 (delta 3), reused 0 (delta 0)
    Receiving objects: 100% (10/10), done.
    Resolving deltas: 100% (3/3), done.
    git|Checking connectivity... done.
    manifest|web:
    manifest|  image: httpd
    manifest|  ports:
    manifest|  - 80:80
    build|RUNNING: docker tag -f httpd httpd/web

## License

Apache 2.0 &copy; 2015 Convox, Inc.
