CHART_TESTING_VERSION := 3.10.1
CONTROLLER_TOOLS_VERSION := v0.14.0
# ENVTEST_VERSION is usually latest, but might need to be pinned from time to time.
# Version pinning is needed due to version incompatibility between controller-runtime and setup-envtest.
# For more information: https://github.com/kubernetes-sigs/controller-runtime/issues/2744
ENVTEST_VERSION := bf15e44028f908c790721fc8fe67c7bf2d06a611
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := 1.29.x
HELM_VERSION := 3.14.3
KIND_VERSION := v0.22.0
# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.29
KUSTOMIZE_VERSION := v5.3.0

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
ifeq ($(KUBERNETES_VERSION), 1.28)
	KIND_NODE_VERSION := kindest/node:v1.28.7@sha256:9bc6c451a289cf96ad0bbaf33d416901de6fd632415b076ab05f5fa7e4f65c58
else ifeq ($(KUBERNETES_VERSION), 1.27)
	KIND_NODE_VERSION := kindest/node:v1.27.11@sha256:681253009e68069b8e01aad36a1e0fa8cf18bb0ab3e5c4069b2e65cafdd70843
endif
