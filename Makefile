TAG ?= latest
DOCKER_ADDITIONAL_ARGS ?= 

syslog2websocket: export CGO_ENABLED=0
syslog2websocket: cmd/syslog2websocket/main.go
	go build -ldflags="-extldflags=-static" ./cmd/$@

container-tag: Dockerfile syslog2websocket
	docker build -t syslogstream:$(TAG) $(DOCKER_ADDITIONAL_ARGS) .
	echo -n "$(TAG)" >container-tag
