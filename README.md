# KeyValue Storage backend

The solution is made of two package.
The main storage package, that implements the storage key value engine.
The http package, that implements the access through REST api on HTTP transport to the engine 
Different engine can be built as backend of the REST api

## Run

```
$ make build-image
$ docker-compose up
```


## Usage
Parameter | Description | Value | Env
--- | --- | --- | ---
listener | port to use for http (0.0.0.0:80) | |
provider | which storage provider to use | (local) |
basedir | path storage for local provider| |

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
docker-compose up
```