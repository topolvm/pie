# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.12.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := v0.17.2
# https://github.com/helm/helm/releases
HELM_VERSION := 3.17.1
# https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION := v0.27.0
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := v5.6.0

# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.32

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_BRANCH := release-$(shell echo $(CONTROLLER_RUNTIME_VERSION) | cut -d "." -f 1-2)
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := $(KUBERNETES_VERSION).x

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.32.2@sha256:f226345927d7e348497136874b6d207e0b32cc52154ad8323129352923a3142f
ifeq ($(KUBERNETES_VERSION), 1.31)
	KIND_NODE_VERSION := kindest/node:v1.31.6@sha256:28b7cbb993dfe093c76641a0c95807637213c9109b761f1d422c2400e22b8e87
else ifeq ($(KUBERNETES_VERSION), 1.30)
	KIND_NODE_VERSION := kindest/node:v1.30.10@sha256:4de75d0e82481ea846c0ed1de86328d821c1e6a6a91ac37bf804e5313670e507
endif
