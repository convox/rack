# Docker Setup

## Versions

Convox requires an up to date Docker environment:

    $ docker -v
    Docker version 1.7.0, build 0baf609

    $ boot2docker version
    Boot2Docker-cli version: v1.7.0
    Git commit: 7d89508

If your versions do not match, you might consider following the 
[uninstall](#uninstall) section first.

## Install

**OS X**

* Follow the [latest Boot2Docker installer](https://github.com/boot2docker/osx-installer/releases/latest) instructions.
* Download and open Boot2Docker-$latest.pkg
* Click through the Installer, entering your administrator password when prompted
* Use the terminal to initialize Boot2Docker and your Docker environment variables
* Use the terminal to start your first Convox application

```bash
$ boot2docker up
Waiting for VM and Docker daemon to start...
........................ooooooooooooooooo
Started.
Writing /Users/noah/.boot2docker/certs/boot2docker-vm/ca.pem
Writing /Users/noah/.boot2docker/certs/boot2docker-vm/cert.pem
Writing /Users/noah/.boot2docker/certs/boot2docker-vm/key.pem
    export DOCKER_HOST=tcp://192.168.59.103:2376
    export DOCKER_CERT_PATH=/Users/user/.boot2docker/certs/boot2docker-vm
    export DOCKER_TLS_VERIFY=1

$ $(boot2docker shellinit)

$ git clone https://github.com/convox-examples/sinatra
$ cd sinatra

$ convox start
RUNNING: docker-compose -p sinatra build
Pulling redis (convox/redis:latest)...
Pulling postgres (convox/postgres:latest)...
Building web...
...
Building worker...
...
redis_1    |       _.-``    `.  `_.  ''-._           Redis 3.0.1 (d614dd0c/0) 64 bit
postgres_1 | LOG:  database system is ready to accept connections
web_1      | I, [2015-07-08T19:50:34.328584 #7]  INFO -- : listening on addr=0.0.0.0:3000 fd=9

$ open http://$(boot2docker ip):3000
```

## Uninstall

**OS X**

Uninstall any Homebrew packages if present:

    $ brew uninstall boot2docker
    $ brew uninstall docker

Uninstall older VirtualBox if present, by typing 'Yes', 'Yes' and your administrator password:

    $ curl -O https://raw.githubusercontent.com/nzoschke/install2docker/master/VirtualBox_Uninstall.tool
    $ bash VirtualBox_Uninstall.tool
    Welcome to the VirtualBox uninstaller script.
    
    Warning! Found the following active VirtualBox processes:
    32803   503 VBoxXPCOMIPCD
    32805   503 VBoxSVC
    32816   503 VBoxNetDHCP
    43584   503 VBoxHeadless
    
    
    We recommend that you quit all VirtualBox processes before
    uninstalling the product.

    Do you wish to continue none the less (Yes/No)?
    Yes

    ...

    And the following KEXTs will be unloaded:
        org.virtualbox.kext.VBoxUSB
        org.virtualbox.kext.VBoxNetFlt
        org.virtualbox.kext.VBoxNetAdp
        org.virtualbox.kext.VBoxDrv

    And the traces of following packages will be removed:
        org.virtualbox.pkg.vboxkexts
        org.virtualbox.pkg.virtualbox
        org.virtualbox.pkg.virtualboxcli
    
    Do you wish to uninstall VirtualBox (Yes/No)?
    Yes

    The uninstallation processes requires administrative privileges
    because some of the installed files cannot be removed by a normal
    user. You may be prompted for your password now...

    Please enter users's password:
    ...
    Done.

If you get an error like "Failed to unload one or more KEXTs, please reboot the 
machine to complete the uninstall.", reboot your computer and run the uninstall
tool again.
