# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.14.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := v0.20.1
# https://github.com/golangci/golangci-lint/releases
GOLANGCI_LINT_VERSION := v2.11.4
# https://github.com/helm/helm/releases
HELM_VERSION := 4.1.4
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.31.0
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := v5.8.1

# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.35

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_K8S_VERSION := $(KUBERNETES_VERSION).0

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f
ifeq ($(KUBERNETES_VERSION), 1.34)
	KIND_NODE_VERSION := kindest/node:v1.34.3@sha256:08497ee19eace7b4b5348db5c6a1591d7752b164530a36f855cb0f2bdcbadd48
else ifeq ($(KUBERNETES_VERSION), 1.33)
	KIND_NODE_VERSION := kindest/node:v1.33.7@sha256:d26ef333bdb2cbe9862a0f7c3803ecc7b4303d8cea8e814b481b09949d353040
endif
