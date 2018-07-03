# CptHook
Receive webhooks from different applications and post them to different IRC channels

After writing a webhook receiver for both Prometheus and Gitlab, to post notifications to
an IRC channel, I decided to merge the code an build something more generic. Many applications
have webhooks which get triggered on an event and HTTP POST a bunch of json to you. Handling this
is nearly identical all the time:
  1. Read some config from a file (e.g. list of IRC channels) 
  2. Provide HTTP endpoint to receive webhooks
  3. Validate and parse the received json
  4. Construct a message
  5. Take the message from `4`and write it to the IRC channels defined in `1`
 
The points `3` and `4` get handled by the specific modules. All the other points are generic
and can be reused for every module.

## Installation
Local installation
```
cp cpthook.yml.example cpthook.yml
dep ensure
go build
go run
```
or use the prebuild Dockercontainer
```
cp cpthook.yml.example config.yml
docker run --rm -it -v $(pwd)/cpthook.yml:/cpthook.yml -v 8086:8086 fleaz/cpthook
```

## Modules
When you want to create a new module, e.g. 'Foo' you have to do three things:
  - Add a section 'foo' to the `cpthook.yml.example`. Everything under `cpthook.foo` will be provided to your module
  - Add the '/foo' endpoint to the `main.go` file
  - Create a `foo.go` file in the `main` package and provide a handler function

### Status
This will soon be used to have a small web interface to show e.g. the last received webhooks.

### Prometheus
Receives webhooks from Alertmanager.

### Gitlab
Receives webhooks from Gitlab. Currently not all types are implemented!
