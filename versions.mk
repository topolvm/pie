# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.11.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := v0.15.0
# https://github.com/helm/helm/releases
HELM_VERSION := 3.15.2
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.23.0
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := v5.4.2

# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.30

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_BRANCH := release-$(shell echo $(CONTROLLER_RUNTIME_VERSION) | cut -d "." -f 1-2)
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := $(KUBERNETES_VERSION).x

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.30.0@sha256:047357ac0cfea04663786a612ba1eaba9702bef25227a794b52890dd8bcd692e
ifeq ($(KUBERNETES_VERSION), 1.29)
	KIND_NODE_VERSION := kindest/node:v1.29.4@sha256:3abb816a5b1061fb15c6e9e60856ec40d56b7b52bcea5f5f1350bc6e2320b6f8
else ifeq ($(KUBERNETES_VERSION), 1.28)
	KIND_NODE_VERSION := kindest/node:v1.28.9@sha256:dca54bc6a6079dd34699d53d7d4ffa2e853e46a20cd12d619a09207e35300bd0
endif
