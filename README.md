# CptHook
[![Build Status](https://travis-ci.org/fleaz/CptHook.svg?branch=main)](https://travis-ci.org/fleaz/CptHook)
[![Go Report Card](https://goreportcard.com/badge/github.com/fleaz/CptHook)](https://goreportcard.com/report/github.com/fleaz/CptHook)
MIT[![License: ](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/fleaz/CptHook/blob/main/LICENSE)

CptHook provides a single endpoint where you can point all your webhook notifications to and get the nicely formatted in an IRC channel of your choice.

Take a look at the [input](https://github.com/fleaz/CptHook/tree/main/input) folder to find out which services are
already supported by CptHook.

**If you have questions or problems come chat with us in #CptHook on HackInt** -> [WebChat](https://webirc.hackint.org/#irc://irc.hackint.org/#CptHook)

## Installation


### Manual

**Requirements**
 * Go >= 1.11

```
# go get -u github.com/fleaz/CptHook
# cp $GOPATH/bin/CptHook /usr/local/bin/CptHook
# vim /etc/cpthook.yml
# CptHook
```

### Docker container

```
vim cpthook.yml
docker run --rm -it -v $(pwd)/cpthook.yml:/etc/cpthook.yml -p 8086:8086 ghcr.io/fleaz/cpthook:stable
```

### Prebuild binaries
Visit the GitHub [release page](https://github.com/fleaz/CptHook/releases/latest) to download them.

## IRC authentication
SASL support is available to authenticate to the server.
The following methods are supported:
 - `SASL-Plain` uses plaintext username and password authentication
 - `SASL-External` can be used with external authentication mechanism like CertFP

To use CertFP, a client certificate (`certfile`) and key (`keyfile`) must be specified in the `irc.ssl.client_cert`
section and the `SASL-External` authentication method must be used.


## Configuration

### General

These settings are available for all modules

```
- endpoint
Defines the URI where the module is reachable.

- type
The input module you want to initialize in this block.

- default_channel
Defines a fallback channel where messages should go if none of the defined filters has matched. Only used in modules which have some kind of routing for events, e.g. the Gitlab module.
```

### Prometheus
Receives webhooks from Alertmanager.

```
- hostname_filter
This regex is used to shorten the hostname of instance name in an alert. This
regex must contain exactly one capture group which will be used as the
hostname if the regex matches.
```

### Gitlab
Receives webhooks from Gitlab. *Currently not all event types are implemented!* When a webhook is received this
module will first check if there is an explicit mapping in the configuration especially for this project. If yes,
this channel will be used. If not, the module will look if there exists for the group. If yes, this channel will be
used. If not, the `default_channel` will be used.
```
- groups
This dictionary maps Gitlab groups to IRC-channels.

- explicit
This dictionary maps full project paths (groupname/projectname) to IRC-channels.
```

### Simple
Receives arbitrary messages as text via a HTTP `POST` request and forwards this message line by line to a channel.
The channel can be specified per request by the `channel` query parameter, otherwise the `default_channel` from the config will
be used.

### Icinga2
Receives webhooks from Icinga2. Add [icinga2-notifications-webhook](https://git.s7t.de/ManiacTwister/icinga2-notifications-webhook) to your
Icinga2 installation to send the required webhooks.

When a webhook is received this module will first check if there is an explicit
mapping in the configuration especially for this host. If yes, this channel will be used. If not, the module will
look if there exists for the hostgroup. If yes, this channel will be used. If not, the `default_channel` channel will be used.

```
- hostgroups
This dictionary maps Icinga2 hostgroups to IRC-channels.

- explicit
This dictionary maps hostnames to IRC-channels.
```

## Build a new module
When you want to create a new module, e.g. for the service 'Foo', follow these steps to get started:
  - Add a section 'foo' to `cpthook.yml.example`. Everything below `cpthook.foo` will be provided to your module. 
  - Add a case to the `main.createModuleObject` function
  - Create `foo.go` and `foo_test.go` files in the `input` folder
  - Implement the `Module` interface according to `input/helper.go`
  - Bonus task: Be a good programmer and write a test :)
