.PHONY: default
default: test

include common.mk

.PHONY: test
test: go-test-all

.PHONY: lint
lint: go-lint-all git-clean-check

.PHONY: generate
generate: buf-generate-all

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

.PHONY: build-docker-experiments-fine-tuning
build-docker-experiments-fine-tuning:
	docker build --build-arg TARGETARCH=amd64 -t llm-operator/experiments-fine-tuning:latest -f build/experiments/fine-tuning/Dockerfile build/experiments/fine-tuning

.PHONY: build-docker-experiments-fake-job
build-docker-experiments-fake-job:
	docker build --build-arg TARGETARCH=amd64 -t llm-operator/experiments-fake-job:latest -f build/experiments/fake-job/Dockerfile build/experiments/fake-job
