# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
# NOTE: the suffix .x means wildcard match so specifying the latest patch version.
ENVTEST_K8S_VERSION := 1.27.x
CHART_TESTING_VERSION := 3.7.1
KUSTOMIZE_VERSION := v5.0.3
CONTROLLER_TOOLS_VERSION := v0.12.0
KUBERNETES_VERSION := 1.27
KIND_VERSION := v0.19.0
HELM_VERSION := 3.12.0

# The container version of kind must be with the digest.
# ref. https://github.com/kubernetes-sigs/kind/releases
KIND_NODE_VERSION := kindest/node:v1.27.1@sha256:b7d12ed662b873bd8510879c1846e87c7e676a79fefc93e17b2a52989d3ff42b
ifeq ($(KUBERNETES_VERSION), 1.26)
	KIND_NODE_VERSION := kindest/node:v1.26.4@sha256:f4c0d87be03d6bea69f5e5dc0adb678bb498a190ee5c38422bf751541cebe92e
else ifeq ($(KUBERNETES_VERSION), 1.25)
	KIND_NODE_VERSION := kindest/node:v1.25.9@sha256:c08d6c52820aa42e533b70bce0c2901183326d86dcdcbedecc9343681db45161
endif
