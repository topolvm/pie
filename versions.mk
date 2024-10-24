# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.11.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := v0.16.4
# https://github.com/helm/helm/releases
HELM_VERSION := 3.16.2
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.24.0
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := v5.5.0

# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.31

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_BRANCH := release-$(shell echo $(CONTROLLER_RUNTIME_VERSION) | cut -d "." -f 1-2)
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := $(KUBERNETES_VERSION).x

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
ifeq ($(KUBERNETES_VERSION), 1.30)
	KIND_NODE_VERSION := kindest/node:v1.30.4@sha256:976ea815844d5fa93be213437e3ff5754cd599b040946b5cca43ca45c2047114
else ifeq ($(KUBERNETES_VERSION), 1.29)
	KIND_NODE_VERSION := kindest/node:v1.29.8@sha256:d46b7aa29567e93b27f7531d258c372e829d7224b25e3fc6ffdefed12476d3aa
endif
