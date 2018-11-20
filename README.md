# KeyValue Storage backend [![Go Report Card](https://goreportcard.com/badge/github.com/aspacca/keyvaluestorage)](https://goreportcard.com/report/github.com/aspacca/keyvaluestorage)

The solution is made of two package.
The main storage package, that implements the storage key value engine.
The http package, that implements the access through REST api on HTTP transport to the engine 
Different engine can be built as backend of the REST api
Current engine supported: filesystem and memory

## Run

```
$ make build-image
$ docker-compose up
```


## Usage
Parameter | Description | Value
--- | --- | ---
listener | port to use for http (0.0.0.0:80) |
provider | which storage provider to use | (fs\|memory)
basedir | path storage for filesystem provider|

## Build

```
go build -o kvs main.go
```

## Test

```
go test ./...
```

## Docker

For easy deployment, we've created a Docker container.

```
docker-compose run keyvaluestorage --provider [fs|memory]
```