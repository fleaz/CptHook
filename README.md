# CptHook
[![Build Status](https://travis-ci.org/fleaz/CptHook.svg?branch=master)](https://travis-ci.org/fleaz/CptHook)
[![Go Report Card](https://goreportcard.com/badge/github.com/fleaz/CptHook)](https://goreportcard.com/report/github.com/fleaz/CptHook)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://github.com/fleaz/CptHook/blob/master/LICENSE)

CptHook can be used as a single endpoint for all your webhook notifications which then get parsed and send to
different IRC channels according to the configuration.

Take a look at the [input](https://github.com/fleaz/CptHook/tree/master/input) folder to find out which services are
already supported by CptHook.

**If you have questions or problems visit #CptHook on HackInt** -> [WebChat](https://webirc.hackint.org/#irc://irc.hackint.org/#CptHook)

## Installation

**Requirements**
 * Go > 1.7

You can either install CptHook manually
```
go get -u github.com/fleaz/CptHook
cp $GOPATH/bin/CptHook /usr/local/bin/CptHook
vim /etc/cpthook.yml
CptHook
```

or use the prebuild Docker container

```
vim cpthook.yml
docker run --rm -it -v $(pwd)/cpthook.yml:/etc/cpthook.yml -p 8086:8086 fleaz/cpthook:stable
```

or download the [prebuild binarys](https://github.com/fleaz/CptHook/releases/latest)

## Authentication
SASL support is available to authenticate to the server.
The following methods are supported:
 - `SASL-Plain` uses plaintext username and password authentication
 - `SASL-External` can be used with external authentication mechanism like CertFP

To use CertFP, a client certificate (`certfile`) and key (`keyfile`) must be specified in the `irc.ssl.client_cert`
section and the `SASL-External` authentication method must be used.

## Build a new module
When you want to create a new module, e.g. for the service 'Foo', follow these steps to get started:
  - Add a section 'foo' to `cpthook.yml.example`. Everything below `cpthook.foo` will be provided to your module. 
  - Add a case to the `main.createModuleObject` function
  - Create `foo.go` file in the `input` folder
  - Implement the `Module` interface according to `input/helper.go`

## Already available modules

**General configuration configuration**
```
- enabled
Defines if CptHook should load this module
- default_channel / default
Defines a fallback channel where messages should go if no filtering has matched
```

### Prometheus
Receives webhooks from Alertmanager.

**Module specific configuration**
```
- hostname_filter
This regex is used to shorten the hostname of instance name in an alert. This regex should contain exactly one
capture group which will be used as the hostname if the regex matches.
```

### Gitlab
Receives webhooks from Gitlab. *Currently not all event types are implemented!* When a webhook is received this
module will first check if there is an explicit mapping in the configuration especially for this project. If yes,
this channel will be used. If not, the module will look if there exists for the group. If yes, this channel will be
used. If not, the `default_channel` will be used.
**Module specific configuration**
```
- groups
This dictionary maps Gitlab groups to IRC-channels.
- explicit
This dictionary maps full project paths (groupname/projectname) to IRC-channels.
```

### Simple
Receives arbitrary messages as text via a HTTP `POST` request and forwards this message line by line to a channel.
The channel can be specified by the `channel` query parameter, otherwise the `default_channel` from the config will
be used.

### Icinga2
Receives webhooks from Icinga2. Add [icinga2-notifications-webhook] to your icinga2 installation to send the
required webhooks.

When a webhook is received this module will first check if there is an explicit
mapping in the configuration especially for this host. If yes, this channel will be used. If not, the module will
look if there exists for the hostgroup. If yes, this channel will be used. If not, the `default` channel will be used.

**Module specific configuration**
```
- hostgroups
This dictionary maps Icinga2 hostgroups to IRC-channels.
- explicit
This dictionary maps hostnames to IRC-channels.
```
[icinga2-notifications-webhook]: https://git.s7t.de/ManiacTwister/icinga2-notifications-webhook
