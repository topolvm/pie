# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.10.1
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := v0.14.0
# https://github.com/helm/helm/releases
HELM_VERSION := 3.14.3
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.22.0
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := v5.3.0

# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.29

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_BRANCH := release-$(shell echo $(CONTROLLER_RUNTIME_VERSION) | cut -d "." -f 1-2)
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := $(KUBERNETES_VERSION).x

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
ifeq ($(KUBERNETES_VERSION), 1.28)
	KIND_NODE_VERSION := kindest/node:v1.28.7@sha256:9bc6c451a289cf96ad0bbaf33d416901de6fd632415b076ab05f5fa7e4f65c58
else ifeq ($(KUBERNETES_VERSION), 1.27)
	KIND_NODE_VERSION := kindest/node:v1.27.11@sha256:681253009e68069b8e01aad36a1e0fa8cf18bb0ab3e5c4069b2e65cafdd70843
endif
