SERVER_IMAGE ?= llmariner/job-manager-server
DISPATCHER_IMAGE ?= llmariner/job-manager-dispatcher
TAG ?= latest

.PHONY: default
default: test

include common.mk

.PHONY: test
test: go-test-all

.PHONY: lint
lint: go-lint-all helm-lint git-clean-check

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
	docker build --build-arg TARGETARCH=amd64 -t $(SERVER_IMAGE):$(TAG) -f build/server/Dockerfile .

.PHONY: build-docker-dispatcher
build-docker-dispatcher:
	docker build --build-arg TARGETARCH=amd64 -t $(DISPATCHER_IMAGE):$(TAG) -f build/dispatcher/Dockerfile .

.PHONY: build-docker-fine-tuning
build-docker-fine-tuning:
	docker build --build-arg TARGETARCH=amd64 -t llmariner/fine-tuning:latest -f build/fine-tuning/Dockerfile build/fine-tuning

.PHONY: build-docker-fake-job
build-docker-fake-job:
	docker build --build-arg TARGETARCH=amd64 -t llmariner/fake-job:latest -f build/fake-job/Dockerfile build/fake-job

.PHONY: check-helm-tool
check-helm-tool:
	@command -v helm-tool >/dev/null 2>&1 || $(MAKE) install-helm-tool

.PHONY: install-helm-tool
install-helm-tool:
	go install github.com/cert-manager/helm-tool@latest

.PHONY: generate-chart-schema
generate-chart-schema: generate-chart-schema-server generate-chart-schema-dispatcher

.PHONY: generate-chart-schema-server
generate-chart-schema-server: check-helm-tool
	@cd ./deployments/server && helm-tool schema > values.schema.json

.PHONY: generate-chart-schema-dispatcher
generate-chart-schema-dispatcher: check-helm-tool
	@cd ./deployments/dispatcher && helm-tool schema > values.schema.json

.PHONY: helm-lint
helm-lint: helm-lint-server helm-lint-dispatcher

.PHONY: helm-lint-server
helm-lint-server: generate-chart-schema-server
	cd ./deployments/server && helm-tool lint
	helm lint ./deployments/server

.PHONY: helm-lint-dispatcher
helm-lint-dispatcher: generate-chart-schema-dispatcher
	cd ./deployments/dispatcher && helm-tool lint
	helm lint ./deployments/dispatcher

include provision.mk
