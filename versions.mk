# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.12.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := v0.18.0
# https://github.com/helm/helm/releases
HELM_VERSION := 3.18.1
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.29.0
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := v5.6.0

# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.33

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_K8S_VERSION := $(KUBERNETES_VERSION).0

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.33.1@sha256:050072256b9a903bd914c0b2866828150cb229cea0efe5892e2b644d5dd3b34f
ifeq ($(KUBERNETES_VERSION), 1.32)
	KIND_NODE_VERSION := kindest/node:v1.32.5@sha256:e3b2327e3a5ab8c76f5ece68936e4cafaa82edf58486b769727ab0b3b97a5b0d
else ifeq ($(KUBERNETES_VERSION), 1.31)
	KIND_NODE_VERSION := kindest/node:v1.31.9@sha256:b94a3a6c06198d17f59cca8c6f486236fa05e2fb359cbd75dabbfc348a10b211
endif
