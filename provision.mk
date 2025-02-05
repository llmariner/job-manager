LLMA_REPO ?= https://github.com/llmariner/llmariner.git
CLONE_PATH ?= work

TN_CLUSTER ?= job-tn
CP_CLUSTER ?= job-cp
WP_CLUSTER ?= job-wp
WORKER_NUM ?= 1

.PHONY: provision-all
provision-all: pull-llma-chart configure-llma-chart create-kind-cluster helm-apply-cp-deps load-server-image helm-apply-cp-llma helm-apply-wp-deps load-dispatcher-image helm-apply-wp-llma load-syncer-image helm-apply-tn-llma

.PHONY: reapply-job-server
reapply-job-server: load-server-image helm-apply-cp-llma rollout-job-server
.PHONY: reapply-job-dispatcher
reapply-job-dispatcher: load-dispatcher-image helm-apply-wp-llma rollout-job-dispatcher
.PHONY: reapply-job-syncer
reapply-job-syncer: load-syncer-image helm-apply-tn-llma rollout-job-syncer

# ------------------------------------------------------------------------------
# chart repository
# ------------------------------------------------------------------------------

.PHONY: pull-llma-chart
pull-llma-chart:
	@if [ -d $(CLONE_PATH) ]; then \
		cd $(CLONE_PATH) && \
		git checkout -- deployments/llmariner/Chart.yaml && \
		git pull; \
	else \
		git clone $(LLMA_REPO) $(CLONE_PATH); \
	fi

.PHONY: configure-llma-chart
configure-llma-chart:
	hack/overwrite-llma-chart-for-test.sh $(CLONE_PATH)
	-rm $(CLONE_PATH)/deployments/llmariner/Chart.lock

# ------------------------------------------------------------------------------
# kind cluster
# ------------------------------------------------------------------------------

.PHONY: create-kind-cluster
create-kind-cluster:
	hack/create-kind-cluster.sh "$(TN_CLUSTER)" "$(CP_CLUSTER)" "$(WP_CLUSTER)" $(WORKER_NUM)

.PHONY: delete-kind-cluster
delete-kind-cluster:
	-kind delete cluster --name $(TN_CLUSTER)
	-kind delete cluster --name $(CP_CLUSTER)
	for cluster in $(shell kind get clusters | grep $(WP_CLUSTER)); do \
		rm $(CLONE_PATH)/.regkey-kind-$$cluster; \
		kind delete cluster --name $$cluster; \
	done

# ------------------------------------------------------------------------------
# deploy dependencies
# ------------------------------------------------------------------------------

CP_DEP_APPS ?= kong,postgres
WP_DEP_APPS ?= kong,fake-gpu-operator

.PHONY: helm-apply-cp-deps
helm-apply-cp-deps:
	$(MAKE) helm-apply-deps DEP_APPS=$(CP_DEP_APPS) KUBE_CTX=kind-$(CP_CLUSTER) HELM_ENV=control

.PHONY: helm-apply-wp-deps
helm-apply-wp-deps:
	for cluster in $(shell kind get clusters | grep $(WP_CLUSTER)); do \
		export REGISTRATION_KEY=$$(cat $(CLONE_PATH)/.regkey-kind-$$cluster||curl --request POST http://localhost:8080/v1/clusters -d "{\"name\":\"$$cluster\"}"|jq -r .registration_key); \
		echo "$$REGISTRATION_KEY" > $(CLONE_PATH)/.regkey-kind-$$cluster; \
		$(MAKE) helm-apply-deps DEP_APPS=$(WP_DEP_APPS) KUBE_CTX=kind-$$cluster HELM_ENV=worker; \
	done

.PHONY: helm-apply-deps
helm-apply-deps:
	hack/helm-apply-deps.sh $(CLONE_PATH) $(DEP_APPS) $(KUBE_CTX) $(HELM_ENV)

# ------------------------------------------------------------------------------
# deploy llmariner
# ------------------------------------------------------------------------------

EXTRA_CP_VALS ?= values-cp.yaml
EXTRA_WP_VALS ?= values-wp.yaml
EXTRA_TN_VALS ?= values-tn.yaml

.PHONY: helm-apply-cp-llma
helm-apply-cp-llma:
	$(MAKE) helm-apply-llma EXTRA_VALS=$(EXTRA_CP_VALS) KUBE_CTX=kind-$(CP_CLUSTER) HELM_ENV=control

.PHONY: helm-apply-wp-llma
helm-apply-wp-llma:
	for cluster in $(shell kind get clusters | grep $(WP_CLUSTER)); do \
		export REGISTRATION_KEY=$$(cat $(CLONE_PATH)/.regkey-kind-$$cluster); \
		$(MAKE) helm-apply-llma EXTRA_VALS=$(EXTRA_WP_VALS) KUBE_CTX=kind-$$cluster HELM_ENV=worker; \
	done

.PHONY: helm-apply-tn-llma
helm-apply-tn-llma:
	$(MAKE) helm-apply-llma EXTRA_VALS=$(EXTRA_TN_VALS) KUBE_CTX=kind-$(TN_CLUSTER)

.PHONY: helm-apply-llma
helm-apply-llma:
	hack/helm-apply-llma.sh $(CLONE_PATH) $(EXTRA_VALS) $(KUBE_CTX) $(HELM_ENV)

# ------------------------------------------------------------------------------
# load images
# ------------------------------------------------------------------------------

.PHONY: load-server-image
load-server-image: build-docker-server
	@kind load docker-image $(SERVER_IMAGE):$(TAG) --name $(CP_CLUSTER)

.PHONY: load-dispatcher-image
load-dispatcher-image: build-docker-dispatcher
	@kind get clusters|grep $(WP_CLUSTER)|xargs -n1 kind load docker-image $(DISPATCHER_IMAGE):$(TAG) --name

.PHONY: load-syncer-image
load-syncer-image: build-docker-syncer
	@kind load docker-image $(SYNCER_IMAGE):$(TAG) --name $(TN_CLUSTER)

# ------------------------------------------------------------------------------
# rollout pods
# ------------------------------------------------------------------------------

.PHONY: rollout-job-server
rollout-job-server:
	@kubectl --context kind-$(CP_CLUSTER) rollout restart deployment -n llmariner job-manager-server

.PHONY: rollout-job-dispatcher
rollout-job-dispatcher:
	@kind get clusters|grep $(WP_CLUSTER)|xargs -n1 -I{} kubectl --context kind-{} rollout restart deployment -n llmariner-wp job-manager-dispatcher

.PHONY: rollout-job-syncer
rollout-job-syncer:
	@kubectl --context kind-$(TN_CLUSTER) rollout restart deployment -n llmariner job-manager-syncer
