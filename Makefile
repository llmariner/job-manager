.PHONY: default
default: test

include common.mk

.PHONY: test
test: go-test-all

.PHONY: lint
lint: go-lint-all git-clean-check

.PHONY: generate
generate: buf-generate-all typescript-compile

.PHONY: build-server
build-server:
	go build -o ./bin/server ./server/cmd/

.PHONY: build-dispatcher
build-dispatcher:
	go build -o ./bin/dispatcher ./dispatcher/cmd/

.PHONY: build-docker-server
build-docker-server:
	docker build --build-arg TARGETARCH=amd64 -t llm-operator/job-manager-server:latest -f build/server/Dockerfile .

.PHONY: build-docker-dispatcher
build-docker-dispatcher:
	docker build --build-arg TARGETARCH=amd64 -t llm-operator/job-manager-dispatcher:latest -f build/dispatcher/Dockerfile .

.PHONY: build-docker-fine-tuning
build-docker-fine-tuning:
	docker build --build-arg TARGETARCH=amd64 -t llm-operator/fine-tuning:latest -f build/fine-tuning/Dockerfile build/fine-tuning

.PHONY: build-docker-fake-job
build-docker-fake-job:
	docker build --build-arg TARGETARCH=amd64 -t llm-operator/fake-job:latest -f build/fake-job/Dockerfile build/fake-job
