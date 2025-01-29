.PHONY: generate
generate:
	go generate api/v0040/api.go
	go generate api/v0041/api.go


.PHONY: build-server
build-server:
	go build -o ./bin/server ./server/cmd/
