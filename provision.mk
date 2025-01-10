LLMA_REPO ?= https://github.com/llmariner/llmariner.git
CLONE_PATH ?= work

TN_CLUSTER ?= job-tn
CP_CLUSTER ?= job-cp
WP_CLUSTER ?= job-wp
WORKER_NUM ?= 1

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

# ------------------------------------------------------------------------------
# deploy dependencies
# ------------------------------------------------------------------------------

CP_DEP_APPS ?= kong,postgres
WP_DEP_APPS ?= kong

.PHONY: helm-apply-cp-deps
helm-apply-cp-deps:
	$(MAKE) helm-apply-deps DEP_APPS=$(CP_DEP_APPS) KUBE_CTX=kind-$(CP_CLUSTER) HELM_ENV=control

.PHONY: helm-apply-wp-deps
helm-apply-wp-deps:
	kind get clusters|grep $(WP_CLUSTER)|xargs -n1 -I{} $(MAKE) helm-apply-deps DEP_APPS=$(WP_DEP_APPS) KUBE_CTX=kind-{} HELM_ENV=worker

.PHONY: helm-apply-deps
helm-apply-deps:
	hack/helm-apply-deps.sh $(CLONE_PATH) $(DEP_APPS) $(KUBE_CTX) $(HELM_ENV)

# ------------------------------------------------------------------------------
# deploy llmariner
# ------------------------------------------------------------------------------

EXTRA_CP_VALS ?= values-cp.yaml
EXTRA_WP_VALS ?= values-wp.yaml

.PHONY: helm-apply-cp-llma
helm-apply-cp-llma:
	$(MAKE) helm-apply-llma EXTRA_VALS=$(EXTRA_CP_VALS) KUBE_CTX=kind-$(CP_CLUSTER) HELM_ENV=control

.PHONY: helm-apply-wp-llma
helm-apply-wp-llma:
	for cluster in $(shell kind get clusters | grep $(WP_CLUSTER)); do \
		export REGISTRATION_KEY=$$(cat $(CLONE_PATH)/.regkey-kind-$$cluster||curl --request POST http://localhost:8080/v1/clusters -d "{\"name\":\"$$cluster\"}"|jq -r .registration_key); \
		echo "$$REGISTRATION_KEY" > $(CLONE_PATH)/.regkey-kind-$$cluster; \
		$(MAKE) helm-apply-llma EXTRA_VALS=$(EXTRA_WP_VALS) KUBE_CTX=kind-$$cluster HELM_ENV=worker; \
	done

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

# ------------------------------------------------------------------------------
# rollout pods
# ------------------------------------------------------------------------------

.PHONY: rollout-job-server
rollout-job-server:
	@kubectl --context kind-$(CP_CLUSTER) rollout restart deployment -n llmariner job-manager-server

.PHONY: rollout-job-dispatcher
rollout-job-dispatcher:
	@kind get clusters|grep $(WP_CLUSTER)|xargs -n1 -I{} kubectl --context kind-{} rollout restart deployment -n llmariner job-manager-dispatcher
